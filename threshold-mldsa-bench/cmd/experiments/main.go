// experiments launches its own local five-process cluster and real backend,
// measures sequential business HTTP requests, and writes public-only evidence.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"time"

	"threshold-mldsa/backend/clearing"
	"threshold-mldsa/backend/signer"
	"threshold-mldsa/network"
)

type result struct {
	TransactionID  string           `json:"transaction_id"`
	Status         string           `json:"status"`
	Verified       bool             `json:"verified"`
	KeyID          string           `json:"key_id"`
	Signature      string           `json:"signature_base64"`
	Attempts       int              `json:"attempts"`
	DurationMS     float64          `json:"duration_ms"`
	SigningMS      float64          `json:"signing_duration_ms"`
	VerificationMS float64          `json:"verification_duration_ms"`
	Counts         [3]int           `json:"round_message_counts"`
	Traffic        *network.Traffic `json:"traffic"`
	Error          string           `json:"error"`
}
type sample struct {
	Index          int              `json:"index"`
	Participants   []int            `json:"participants"`
	HTTPStatus     int              `json:"http_status"`
	Verified       bool             `json:"verified"`
	Attempts       int              `json:"attempts"`
	BackendMS      float64          `json:"backend_ms"`
	SigningMS      float64          `json:"signing_ms"`
	VerificationMS float64          `json:"verification_ms"`
	EndToEndMS     float64          `json:"end_to_end_ms"`
	Counts         [3]int           `json:"round_message_counts"`
	Traffic        *network.Traffic `json:"traffic,omitempty"`
	Error          string           `json:"error,omitempty"`
}
type stats struct{ Mean, P50, P95, Min, Max float64 }
type report struct {
	TimeUTC     string          `json:"time_utc"`
	OS          string          `json:"os"`
	Arch        string          `json:"arch"`
	GoVersion   string          `json:"go_version"`
	LogicalCPUs int             `json:"logical_cpus"`
	KeyID       string          `json:"key_id"`
	Warmup      int             `json:"warmup_excluded"`
	Requested   int             `json:"requested"`
	Succeeded   int             `json:"succeeded"`
	Backend     stats           `json:"backend_ms_successes"`
	EndToEnd    stats           `json:"end_to_end_ms_successes"`
	Checks      map[string]bool `json:"checks"`
	Samples     []sample        `json:"samples"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	iterations := flag.Int("iterations", 100, "measured sequential signatures (at least 30 recommended)")
	warmup := flag.Int("warmup", 5, "excluded warmup signatures")
	out := flag.String("output", "experiments/results", "public evidence directory")
	preview := flag.Duration("preview", 0, "keep this owned demo cluster alive for UI preview after experiments")
	base := flag.Int("base-port", 0, "first of 5 node ports; 0 selects a free range")
	port := flag.Int("port", 0, "backend port; 0 selects a free port")
	flag.Parse()
	if *iterations < 1 || *iterations > 1000 || *warmup < 0 || *warmup > 100 || *preview < 0 {
		return errors.New("invalid experiment bounds")
	}
	if *out == "" {
		return errors.New("output directory must not be empty")
	}
	outputPath, err := filepath.Abs(*out)
	if err != nil {
		return err
	}
	*out = outputPath
	launcher, e := filepath.Abs("bin/launcher.exe")
	if e != nil {
		return e
	}
	node, _ := filepath.Abs("bin/signer.exe")
	backend, _ := filepath.Abs("bin/backend.exe")
	for _, p := range []string{launcher, node, backend} {
		if _, e := os.Stat(p); e != nil {
			return errors.New("build bin/launcher.exe, bin/signer.exe and bin/backend.exe first")
		}
	}
	if *base == 0 {
		*base, e = freeRange(5)
		if e != nil {
			return e
		}
	}
	if *port == 0 {
		// Node ports have been reserved conceptually but launcher has not bound
		// them yet. A second ephemeral allocation can otherwise reuse one.
		candidate, err := freeRange(6)
		e = err
		if e != nil {
			return e
		}
		*port = candidate
		if *port >= *base && *port < *base+5 {
			*port = candidate + 5
		}
	}
	if *port < 1 || *port > 65535 || *port >= *base && *port < *base+5 {
		return errors.New("backend port must not overlap the five node ports")
	}
	var secret [32]byte
	if _, e = rand.Read(secret[:]); e != nil {
		return e
	}
	token := hex.EncodeToString(secret[:])
	clear(secret[:])
	launch := exec.Command(launcher, "-signer="+node, fmt.Sprintf("-base-port=%d", *base))
	launch.Env = append(os.Environ(), network.CoordinatorTokenEnv+"="+token)
	public, e := launch.Output()
	if e != nil {
		return errors.New("trusted launcher failed (check ports and binaries)")
	}
	cluster, e := network.ReadClusterConfig(bytes.NewReader(public))
	if e != nil {
		return e
	}
	var processes []*os.Process
	defer func() {
		for _, p := range processes {
			_ = p.Kill()
			_, _ = p.Wait()
			_ = p.Release()
		}
	}()
	for _, n := range cluster.Nodes {
		p, e := os.FindProcess(n.PID)
		if e != nil {
			return e
		}
		processes = append(processes, p)
	}
	// Only public material is written. Credentials exist in memory and the
	// backend environment; node secrets travel solely through launcher's pipes.
	if e = os.MkdirAll(".runtime", 0700); e != nil {
		return e
	}
	temp, e := os.MkdirTemp(".runtime", "c3-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(temp) // Exact private temporary directory created above.
	configPath, _ := filepath.Abs(filepath.Join(temp, "cluster.public.json"))
	if e = os.WriteFile(configPath, public, 0600); e != nil {
		return e
	}
	start := exec.Command(backend)
	start.Env = append(os.Environ(), "JMB_SIGNER_MODE=distributed", "JMB_CLUSTER_CONFIG="+configPath, network.CoordinatorTokenEnv+"="+token, fmt.Sprintf("JMB_BACKEND_ADDR=127.0.0.1:%d", *port), "JMB_NODE_TIMEOUT=3s", "JMB_SIGN_TIMEOUT=20s")
	start.Stdout = io.Discard
	// Backend startup logs contain only public address/mode/KeyID. Keep them
	// visible so configuration or listener failures cannot look like a timeout.
	start.Stderr = os.Stderr
	if e = start.Start(); e != nil {
		return e
	}
	processes = append(processes, start.Process)
	url := fmt.Sprintf("http://127.0.0.1:%d", *port)
	client := &http.Client{Timeout: 35 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	ready := false
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(80 * time.Millisecond) {
		var c signer.ClusterStatus
		status, err := call(client, url+"/api/cluster", http.MethodGet, nil, &c)
		if err == nil && status == 200 && c.KeyID == cluster.KeyID && len(c.Nodes) == 5 {
			ready = true
			for _, n := range c.Nodes {
				ready = ready && n.Status == "CRYPTO_READY"
			}
			if ready {
				break
			}
		}
	}
	if !ready {
		return errors.New("backend did not observe 5 CRYPTO_READY processes")
	}
	fmt.Printf("Dashboard ready: %s (5 independent signing processes; credential not printed)\n", url)
	verifier, e := signer.NewVerifier(cluster.PublicConfig, cluster.KeyID)
	if e != nil {
		return e
	}
	combos := [][]int{{0, 1, 2}, {0, 1, 3}, {0, 1, 4}, {0, 2, 3}, {0, 2, 4}, {0, 3, 4}, {1, 2, 3}, {1, 2, 4}, {1, 3, 4}, {2, 3, 4}}
	r := report{TimeUTC: time.Now().UTC().Format(time.RFC3339), OS: runtime.GOOS, Arch: runtime.GOARCH, GoVersion: runtime.Version(), LogicalCPUs: runtime.NumCPU(), KeyID: cluster.KeyID, Warmup: *warmup, Requested: *iterations, Checks: map[string]bool{"five_crypto_ready_processes": true}}
	comboSuccess := make(map[string]int)
	var backendValues, endValues []float64
	for i := -*warmup; i < *iterations; i++ {
		selection := combos[max(i, 0)%len(combos)]
		tx := clearing.Transaction{Version: "clearing-v1", TransactionID: fmt.Sprintf("EXP-%d-%04d", time.Now().Unix(), i+*warmup), FromBank: "BankA", ToBank: "BankB", AmountFen: 100000000 + int64(i+*warmup), Currency: "CNY", Nonce: fmt.Sprintf("experiment-%s-%d", cluster.KeyID[:8], i)}
		var created map[string]any
		status, e := call(client, url+"/api/transactions", http.MethodPost, tx, &created)
		if e != nil || status != 201 {
			return fmt.Errorf("create experiment transaction failed: %d %v", status, e)
		}
		var value result
		begin := time.Now()
		status, e = call(client, url+"/api/transactions/"+tx.TransactionID+"/sign", http.MethodPost, map[string]any{"participants": selection}, &value)
		elapsed := float64(time.Since(begin)) / float64(time.Millisecond)
		message, _ := clearing.SigningMessage(tx)
		signature, decodeErr := base64.StdEncoding.DecodeString(value.Signature)
		verified := e == nil && decodeErr == nil && status == 200 && value.Status == "VERIFIED" && value.Verified && value.KeyID == cluster.KeyID && verifier.Verify(value.KeyID, message, signature)
		modified := tx
		modified.AmountFen++
		changed, _ := clearing.SigningMessage(modified)
		if verified && verifier.Verify(value.KeyID, changed, signature) {
			return errors.New("tampered transaction unexpectedly verified")
		}
		if i < 0 {
			if !verified {
				return errors.New("warmup signature failed")
			}
			continue
		}
		s := sample{Index: i + 1, Participants: selection, HTTPStatus: status, Verified: verified, Attempts: value.Attempts, BackendMS: value.DurationMS, SigningMS: value.SigningMS, VerificationMS: value.VerificationMS, EndToEndMS: elapsed, Counts: value.Counts, Traffic: value.Traffic, Error: value.Error}
		if e != nil {
			s.Error = e.Error()
		}
		r.Samples = append(r.Samples, s)
		if verified {
			r.Succeeded++
			backendValues = append(backendValues, value.DurationMS)
			endValues = append(endValues, elapsed)
			comboSuccess[fmt.Sprint(selection)]++
		}
		if i == 0 && verified {
			var check struct {
				Verified bool `json:"verified"`
			}
			code, err := call(client, url+"/api/transactions/"+tx.TransactionID+"/verify", http.MethodPost, tx, &check)
			r.Checks["original_verified_via_public_api"] = err == nil && code == 200 && check.Verified
			code, err = call(client, url+"/api/transactions/"+tx.TransactionID+"/verify", http.MethodPost, modified, &check)
			r.Checks["amount_plus_one_fen_rejected"] = err == nil && code == 200 && !check.Verified
			var duplicate result
			code, err = call(client, url+"/api/transactions/"+tx.TransactionID+"/sign", http.MethodPost, map[string]any{"participants": selection}, &duplicate)
			r.Checks["duplicate_sign_rejected_409"] = err == nil && code == 409
		}
		if (i+1)%10 == 0 {
			fmt.Printf("Measured %d/%d signatures; verified=%d\n", i+1, *iterations, r.Succeeded)
		}
	}
	r.Checks["all_ten_legal_combinations"] = len(comboSuccess) == 10
	r.Checks["all_measured_signatures_and_tamper_checks_passed"] = r.Succeeded == r.Requested
	two := clearing.Transaction{Version: "clearing-v1", TransactionID: "EXP-TWO-PARTIES", FromBank: "BankA", ToBank: "BankB", AmountFen: 100, Currency: "CNY", Nonce: "negative-case"}
	var body result
	code, err := call(client, url+"/api/transactions", http.MethodPost, two, &body)
	if err == nil && code == 201 {
		code, err = call(client, url+"/api/transactions/EXP-TWO-PARTIES/sign", http.MethodPost, map[string]any{"participants": []int{0, 2}}, &body)
	}
	r.Checks["two_participants_rejected_400"] = err == nil && code == 400
	r.Backend = statistics(backendValues)
	r.EndToEnd = statistics(endValues)
	if e = writeReport(*out, r, comboSuccess); e != nil {
		return e
	}
	fmt.Printf("Evidence: %s; success %d/%d; mean %.4f ms, P50 %.4f ms, P95 %.4f ms\n", *out, r.Succeeded, r.Requested, r.Backend.Mean, r.Backend.P50, r.Backend.P95)
	if *preview > 0 {
		fmt.Printf("Preview remains available for %s: %s\n", *preview, url)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		timer := time.NewTimer(*preview)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
	}
	for name, passed := range r.Checks {
		if !passed {
			return fmt.Errorf("acceptance check failed or insufficient samples: %s", name)
		}
	}
	return nil
}
func freeRange(n int) (int, error) {
	for attempt := 0; attempt < 60; attempt++ {
		first, e := net.Listen("tcp", "127.0.0.1:0")
		if e != nil {
			return 0, e
		}
		base := first.Addr().(*net.TCPAddr).Port
		ls := []net.Listener{first}
		if base+n <= 65536 {
			for i := 1; i < n; i++ {
				l, e := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base+i))
				if e != nil {
					break
				}
				ls = append(ls, l)
			}
		}
		for _, l := range ls {
			l.Close()
		}
		if len(ls) == n {
			return base, nil
		}
	}
	return 0, errors.New("no free local port range")
}
func call(c *http.Client, url, method string, body, target any) (int, error) {
	var reader io.Reader
	if body != nil {
		data, e := json.Marshal(body)
		if e != nil {
			return 0, e
		}
		reader = bytes.NewReader(data)
	}
	req, e := http.NewRequest(method, url, reader)
	if e != nil {
		return 0, e
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := c.Do(req)
	if e != nil {
		return 0, e
	}
	defer resp.Body.Close()
	e = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(target)
	return resp.StatusCode, e
}
func statistics(values []float64) stats {
	if len(values) == 0 {
		return stats{}
	}
	v := slices.Clone(values)
	slices.Sort(v)
	sum := 0.0
	for _, n := range v {
		sum += n
	}
	return stats{Mean: sum / float64(len(v)), P50: v[int(math.Ceil(float64(len(v))*.5))-1], P95: v[int(math.Ceil(float64(len(v))*.95))-1], Min: v[0], Max: v[len(v)-1]}
}
func writeReport(dir string, r report, combos map[string]int) error {
	if e := os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	data, e := json.MarshalIndent(r, "", "  ")
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(dir, "results.json"), append(data, '\n'), 0644); e != nil {
		return e
	}
	f, e := os.Create(filepath.Join(dir, "samples.csv"))
	if e != nil {
		return e
	}
	w := csv.NewWriter(f)
	header := []string{"index", "participants", "verified", "http_status", "attempts", "backend_ms", "signing_ms", "verification_ms", "end_to_end_ms"}
	for round := 1; round <= 3; round++ {
		for _, suffix := range []string{"messages", "http_sent_bytes", "http_received_bytes", "protocol_response_bytes"} {
			header = append(header, fmt.Sprintf("round%d_%s", round, suffix))
		}
	}
	_ = w.Write(header)
	for _, s := range r.Samples {
		row := []string{strconv.Itoa(s.Index), fmt.Sprint(s.Participants), strconv.FormatBool(s.Verified), strconv.Itoa(s.HTTPStatus), strconv.Itoa(s.Attempts), fmt.Sprintf("%.6f", s.BackendMS), fmt.Sprintf("%.6f", s.SigningMS), fmt.Sprintf("%.6f", s.VerificationMS), fmt.Sprintf("%.6f", s.EndToEndMS)}
		for i := 0; i < 3; i++ {
			row = append(row, strconv.Itoa(s.Counts[i]))
			if s.Traffic == nil {
				row = append(row, "", "", "")
			} else {
				t := s.Traffic.Rounds[i]
				row = append(row, strconv.FormatInt(t.Sent, 10), strconv.FormatInt(t.Received, 10), strconv.FormatInt(t.ProtocolResponses, 10))
			}
		}
		_ = w.Write(row)
	}
	w.Flush()
	e = w.Error()
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	var text bytes.Buffer
	fmt.Fprintf(&text, "# C3 正式实验记录\n\n时间：%s。环境：%s/%s，%s，逻辑 CPU 数 %d。\n\n五个独立 signer 进程 + 独立业务后端；localhost HTTP/1.1，无 TLS；串行负载。可信初始化仅执行一次；KeyID `%s`。\n\n预热 %d 次不纳入统计；正式请求 %d 次，独立验签成功 %d 次。各组合轮流执行；成功率 %.2f%%。\n\n| 口径（成功样本，ms） | 平均值 | P50 | P95 | 最小 | 最大 |\n| --- | ---: | ---: | ---: | ---: | ---: |\n", r.TimeUTC, r.OS, r.Arch, r.GoVersion, r.LogicalCPUs, r.KeyID, r.Warmup, r.Requested, r.Succeeded, 100*float64(r.Succeeded)/float64(r.Requested))
	for _, v := range []struct {
		name  string
		value stats
	}{{"后端签名 + 独立验签", r.Backend}, {"客户端 HTTP 请求端到端", r.EndToEnd}} {
		s := v.value
		fmt.Fprintf(&text, "| %s | %.4f | %.4f | %.4f | %.4f | %.4f |\n", v.name, s.Mean, s.P50, s.P95, s.Min, s.Max)
	}
	fmt.Fprintln(&text, "\nP50/P95 使用 nearest-rank；后端耗时含选中节点健康检查、公钥核对、全部尝试、必要的 Abort 及独立验签，不含创建交易 HTTP。客户端端到端还包含签名 API 的 HTTP 往返。未扣除随机 Combine 拒绝后的新尝试。")
	fmt.Fprint(&text, "\n## 正确性与安全检查\n\n")
	keys := make([]string, 0, len(r.Checks))
	for k := range r.Checks {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		fmt.Fprintf(&text, "- %s：%t\n", k, r.Checks[k])
	}
	fmt.Fprintln(&text, "\n合法组合成功次数：")
	keys = keys[:0]
	for k := range combos {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		fmt.Fprintf(&text, "- `%s`：%d\n", k, combos[k])
	}
	fmt.Fprintln(&text, "\n## 通信开销（实测，所有成功样本平均）\n\n| 轮次 | HTTP 请求字节 | HTTP 响应字节 | 原始协议响应 Payload 字节 |\n| --- | ---: | ---: | ---: |")
	for i := 0; i < 3; i++ {
		var sent, received, protocol int64
		var n int
		for _, s := range r.Samples {
			if s.Verified && s.Traffic != nil {
				t := s.Traffic.Rounds[i]
				sent += t.Sent
				received += t.Received
				protocol += t.ProtocolResponses
				n++
			}
		}
		if n > 0 {
			fmt.Fprintf(&text, "| Round %d | %.2f | %.2f | %.2f |\n", i+1, float64(sent)/float64(n), float64(received)/float64(n), float64(protocol)/float64(n))
		}
	}
	fmt.Fprintln(&text, "\nHTTP 字节由协调器 net.Conn.Read/Write 计数，包含请求/响应头（只计长度，不保存凭据）、JSON、Base64、分块编码等 HTTP/1.1 字节；计入所有三轮尝试，不含健康检查、公钥查询、状态查询、Abort、浏览器与后端流量，不含 TCP/IP、重传或二层头。因此这是三轮 HTTP 应用流量，不是抓包总流量。原始 Payload 仅统计节点返回的原生公开密码消息，既不等于 HTTP 字节，也不包含协调器转发给各节点的重复消息。逐样本、逐轮数据见 samples.csv。\n\n## 安全边界与补充证据\n\n本次性能实验未注入节点故障。节点进程中断、执行后响应超时、Abort 未确认安全停机、Combine 拒绝后全新 Round1，以及并发重复提交由真实进程集成测试单独验证，详见 acceptance-test.log；不要把门限参数检查等同于节点中断实验。\n\n初始化通过独立 stdin 管道交付每个节点自己的份额；公开配置、日志和结果不含私钥或协调器 Token。密钥隔离结论基于初始化代码及份额测试，不宣称进行了进程内存取证。防重放和幂等均限定当前受控启动生命周期；不提供生产密钥持久化、认证用户管理或真实资金转移。")
	return os.WriteFile(filepath.Join(dir, "REPORT.md"), text.Bytes(), 0644)
}
