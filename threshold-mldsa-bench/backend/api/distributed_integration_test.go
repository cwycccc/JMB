package api_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"threshold-mldsa/backend/api"
	"threshold-mldsa/backend/clearing"
	"threshold-mldsa/backend/signer"
	transactionstore "threshold-mldsa/backend/store"
	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
)

type businessView struct {
	Transaction              clearing.Transaction `json:"transaction"`
	TransactionID            string               `json:"transaction_id"`
	Status                   string               `json:"status"`
	Verified                 bool                 `json:"verified"`
	Participants             []int                `json:"participants"`
	KeyID                    string               `json:"key_id"`
	SigningMode              string               `json:"signing_mode"`
	SessionID                string               `json:"session_id"`
	Attempts                 int                  `json:"attempts"`
	Signature                string               `json:"signature_base64"`
	DurationMS               float64              `json:"duration_ms"`
	VerificationDurationMS   float64              `json:"verification_duration_ms"`
	RoundMessages            [3]int               `json:"round_message_counts"`
	AbortConfirmed           *bool                `json:"abort_confirmed"`
	RequiresReinitialization bool                 `json:"requires_reinitialization"`
	Error                    string               `json:"error"`
}

type clusterFixture struct {
	config    network.ClusterConfig
	token     string
	processes map[int]*os.Process
}

func startBusinessCluster(t *testing.T) *clusterFixture {
	t.Helper()
	launcher, signerPath := os.Getenv("C24_LAUNCHER_BINARY"), os.Getenv("C24_SIGNER_BINARY")
	if launcher == "" || signerPath == "" {
		t.Skip("set C24_LAUNCHER_BINARY and C24_SIGNER_BINARY to run real financial HTTP/five-process acceptance")
	}
	base := freeBusinessPorts(t)
	var credential [32]byte
	if _, err := rand.Read(credential[:]); err != nil {
		t.Fatal(err)
	}
	token := hex.EncodeToString(credential[:])
	clear(credential[:])
	command := exec.Command(launcher, "-signer="+signerPath, fmt.Sprintf("-base-port=%d", base))
	command.Env = append(os.Environ(), network.CoordinatorTokenEnv+"="+token)
	output, err := command.Output()
	if err != nil {
		t.Fatal("trusted launcher failed")
	}
	config, err := network.ReadClusterConfig(bytes.NewReader(output))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"secret_share"`, `"coordinator_token"`, `"seed"`} {
		if bytes.Contains(output, []byte(forbidden)) {
			t.Fatal("secret field in public cluster summary")
		}
	}
	fixture := &clusterFixture{config: config, token: token, processes: make(map[int]*os.Process)}
	t.Cleanup(func() {
		for id := range fixture.processes {
			fixture.kill(id)
		}
	})
	seen := make(map[int]bool)
	for _, node := range config.Nodes {
		if seen[node.PID] || node.PID == command.Process.Pid {
			t.Fatal("signer nodes are not independent processes")
		}
		seen[node.PID] = true
		process, err := os.FindProcess(node.PID)
		if err != nil {
			t.Fatal(err)
		}
		fixture.processes[node.PartyID] = process
	}
	return fixture
}

func (f *clusterFixture) kill(id int) {
	if process := f.processes[id]; process != nil {
		_ = process.Kill()
		_, _ = process.Wait()
		_ = process.Release()
		delete(f.processes, id)
	}
}

func freeBusinessPorts(t *testing.T) int {
	t.Helper()
	for tries := 0; tries < 30; tries++ {
		first, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		base := first.Addr().(*net.TCPAddr).Port
		listeners := []net.Listener{first}
		if base <= 65531 {
			for offset := 1; offset < 5; offset++ {
				listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base+offset))
				if err != nil {
					break
				}
				listeners = append(listeners, listener)
			}
		}
		for _, listener := range listeners {
			listener.Close()
		}
		if len(listeners) == 5 {
			return base
		}
	}
	t.Fatal("no five-port range available")
	return 0
}

func openBusinessBackend(t *testing.T, f *clusterFixture, config network.ClusterConfig, nodeTimeout time.Duration) string {
	t.Helper()
	service, err := signer.NewDistributedSigner(config, f.token, nodeTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)
	verifier, err := signer.NewVerifier(config.PublicConfig, config.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := api.NewHandler(transactionstore.New(), service, verifier, api.Options{Mode: "distributed", SignTimeout: 8 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server.URL
}

func sampleBusinessTx(id string) clearing.Transaction {
	return clearing.Transaction{Version: "clearing-v1", TransactionID: id, FromBank: "BankA", ToBank: "BankB", AmountFen: 100000000, Currency: "CNY", Nonce: id + "-nonce"}
}

func businessCall(method, address string, body any) (int, businessView, error) {
	var result businessView
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, result, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, address, reader)
	if err != nil {
		return 0, result, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return 0, result, err
	}
	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&result)
	return response.StatusCode, result, err
}

func mustBusinessCall(t *testing.T, method, address string, body any, want int) businessView {
	t.Helper()
	status, result, err := businessCall(method, address, body)
	if err != nil || status != want {
		t.Fatalf("business HTTP status=%d want=%d error=%v service_error=%s", status, want, err, result.Error)
	}
	return result
}

func TestDistributedClearingFiveProcesses(t *testing.T) {
	f := startBusinessCluster(t)
	t.Run("create_network_sign_independent_verify_get_and_modified_amount", func(t *testing.T) {
		config, proxies := observedConfig(t, f.config, "")
		backend := openBusinessBackend(t, f, config, 3*time.Second)
		tx := sampleBusinessTx("TX001")
		created := mustBusinessCall(t, http.MethodPost, backend+"/api/transactions", tx, http.StatusCreated)
		if created.Status != "PENDING" {
			t.Fatal("create did not return PENDING")
		}
		signed := mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/TX001/sign", nil, http.StatusOK)
		if !signed.Verified || signed.Status != "VERIFIED" || signed.KeyID != f.config.KeyID || signed.SigningMode != "distributed" || signed.SessionID == "" || signed.Attempts < 1 || signed.DurationMS <= 0 {
			t.Fatal("invalid distributed business result")
		}
		if fmt.Sprint(signed.Participants) != "[0 2 4]" {
			t.Fatal("backend selected wrong participants")
		}
		message, err := clearing.SigningMessage(tx)
		if err != nil {
			t.Fatal(err)
		}
		var observed [3]int
		for id, proxy := range proxies {
			counts := proxy.roundCounts(cryptoengine.MessageHash(message))
			for i := range counts {
				observed[i] += counts[i]
			}
			if (id == 1 || id == 3) && counts != [3]int{} {
				t.Fatal("unselected node received a signing round")
			}
			proxy.assertNoRetries(t)
		}
		if observed != signed.RoundMessages || observed[0] < 3 || observed[1] < 3 || observed[2] < 3 {
			t.Fatal("business signature did not execute actual HTTP rounds")
		}
		fetched := mustBusinessCall(t, http.MethodGet, backend+"/api/transactions/TX001", nil, http.StatusOK)
		if fetched.Transaction.AmountFen != 100000000 || fetched.Signature != signed.Signature || fetched.KeyID != signed.KeyID || fetched.SessionID != signed.SessionID {
			t.Fatal("GET lost signed transaction metadata")
		}
		verifier, err := signer.NewVerifier(f.config.PublicConfig, f.config.KeyID)
		if err != nil {
			t.Fatal(err)
		}
		signature, err := base64.StdEncoding.DecodeString(signed.Signature)
		if err != nil {
			t.Fatal(err)
		}
		if !verifier.Verify(signed.KeyID, message, signature) {
			t.Fatal("system public key could not independently verify API signature")
		}
		tx.AmountFen = 200000000
		modified, err := clearing.SigningMessage(tx)
		if err != nil {
			t.Fatal(err)
		}
		if verifier.Verify(signed.KeyID, modified, signature) {
			t.Fatal("modified amount verified")
		}
		t.Logf("TX001 created; real nodes=[0,2,4]; rounds=%v; attempts=%d; same_key_id=true; independent_verify=true; modified_amount_verified=false; duration_ms=%.4f", observed, signed.Attempts, signed.DurationMS)
	})
	t.Run("duplicate_transaction_cannot_create_concurrent_sessions", func(t *testing.T) {
		config, proxies := observedConfig(t, f.config, "gate-round1")
		proxy := proxies[0]
		defer proxy.releaseGate()
		backend := openBusinessBackend(t, f, config, 3*time.Second)
		tx := sampleBusinessTx("DUP001")
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions", tx, http.StatusCreated)
		type outcome struct {
			status int
			view   businessView
			err    error
		}
		first := make(chan outcome, 1)
		go func() {
			status, view, err := businessCall(http.MethodPost, backend+"/api/transactions/DUP001/sign", nil)
			first <- outcome{status, view, err}
		}()
		select {
		case <-proxy.entered:
		case <-time.After(3 * time.Second):
			t.Fatal("network signing did not reach Round1")
		}
		pending := mustBusinessCall(t, http.MethodGet, backend+"/api/transactions/DUP001", nil, http.StatusOK)
		if pending.Status != "SIGNING" || pending.Verified {
			t.Fatal("transaction was not atomically marked SIGNING")
		}
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/DUP001/sign", nil, http.StatusConflict)
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions", tx, http.StatusConflict)
		proxy.releaseGate()
		result := <-first
		if result.err != nil || result.status != http.StatusOK || !result.view.Verified {
			t.Fatal("first request failed")
		}
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/DUP001/sign", nil, http.StatusConflict)
		message, _ := clearing.SigningMessage(tx)
		if len(proxy.sessionsFor(cryptoengine.MessageHash(message))) != 1 {
			t.Fatal("duplicate business submission created more than one network session")
		}
		proxy.assertNoRetries(t)
		t.Log("SIGNING is visible; concurrent/repeated requests return 409; exactly one network session created")
	})
	t.Run("network_timeout_records_failed_and_retry_uses_new_session", func(t *testing.T) {
		config, proxies := observedConfig(t, f.config, "timeout-round2")
		backend := openBusinessBackend(t, f, config, 300*time.Millisecond)
		tx := sampleBusinessTx("TIMEOUT001")
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions", tx, http.StatusCreated)
		failed := mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/TIMEOUT001/sign", nil, http.StatusGatewayTimeout)
		if failed.Status != "FAILED" || failed.Verified || failed.Signature != "" || failed.SessionID == "" || failed.AbortConfirmed == nil || !*failed.AbortConfirmed {
			t.Fatal("timed-out signature was not safely recorded FAILED")
		}
		fetched := mustBusinessCall(t, http.MethodGet, backend+"/api/transactions/TIMEOUT001", nil, http.StatusOK)
		if fetched.Status != "FAILED" || fetched.Verified {
			t.Fatal("GET incorrectly reported timeout success")
		}
		assertAttemptAborted(t, f, failed, tx)
		retried := mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/TIMEOUT001/sign", nil, http.StatusOK)
		if !retried.Verified || retried.SessionID == failed.SessionID {
			t.Fatal("business retry reused the timed-out session")
		}
		if !proxies[2].sawRound2State.Load() || proxies[2].abortCalls.Load() == 0 {
			t.Fatal("timeout did not query executed Round2 and Abort")
		}
		for _, proxy := range proxies {
			proxy.assertNoRetries(t)
		}
		t.Log("executed Round2 response timed out; HTTP 504/FAILED; old attempt ABORTED; business retry used a new session and verified")
	})
	t.Run("only_two_signer_processes_business_refuses_without_local_fallback", func(t *testing.T) {
		isolated := startBusinessCluster(t)
		config, proxies := observedConfig(t, isolated.config, "")
		backend := openBusinessBackend(t, isolated, config, 300*time.Millisecond)
		for _, id := range []int{1, 3, 4} {
			isolated.kill(id)
		}
		tx := sampleBusinessTx("FEW001")
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions", tx, http.StatusCreated)
		failed := mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/FEW001/sign", nil, http.StatusServiceUnavailable)
		if failed.Status != "FAILED" || failed.Verified || failed.Signature != "" || failed.SessionID != "" {
			t.Fatal("insufficient nodes were accepted")
		}
		message, _ := clearing.SigningMessage(tx)
		for _, proxy := range proxies {
			if proxy.roundCounts(cryptoengine.MessageHash(message)) != [3]int{} {
				t.Fatal("cryptographic session started despite unavailable selected node")
			}
		}
		t.Log("only P0/P2 alive; API returned 503/FAILED before Round1; no local fallback")
	})
	t.Run("unconfirmed_abort_blocks_further_business_sessions", func(t *testing.T) {
		config, proxies := observedConfig(t, f.config, "unconfirmed-abort")
		backend := openBusinessBackend(t, f, config, 300*time.Millisecond)
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions", sampleBusinessTx("UNCERTAIN001"), http.StatusCreated)
		failed := mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/UNCERTAIN001/sign", nil, http.StatusGatewayTimeout)
		if !failed.RequiresReinitialization || failed.AbortConfirmed == nil || *failed.AbortConfirmed || failed.Verified {
			t.Fatal("unconfirmed Abort was not surfaced")
		}
		before := proxies[2].totalRounds()
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/UNCERTAIN001/sign", nil, http.StatusConflict)
		mustBusinessCall(t, http.MethodPost, backend+"/api/transactions", sampleBusinessTx("UNCERTAIN002"), http.StatusCreated)
		blocked := mustBusinessCall(t, http.MethodPost, backend+"/api/transactions/UNCERTAIN002/sign", nil, http.StatusServiceUnavailable)
		if !blocked.RequiresReinitialization || proxies[2].totalRounds() != before {
			t.Fatal("new network session started after uncertain Abort")
		}
		t.Log("unconfirmed Abort => FAILED/requires_reinitialization; same transaction blocked with 409, new signing blocked with 503")
	})
}

func assertAttemptAborted(t *testing.T, f *clusterFixture, failed businessView, tx clearing.Transaction) {
	t.Helper()
	_, endpoints, err := f.config.Validate()
	if err != nil {
		t.Fatal(err)
	}
	client, err := network.NewClient(endpoints, f.token, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	message, _ := clearing.SigningMessage(tx)
	req := network.RoundRequest{WireContext: cryptoengine.WireContext{Version: cryptoengine.WireVersion, SessionID: failed.SessionID, Attempt: uint64(failed.Attempts - 1), KeyID: f.config.KeyID, Participants: []int{0, 2, 4}, MessageHash: cryptoengine.MessageHash(message)}}
	for _, id := range req.Participants {
		state, err := client.State(context.Background(), id, req)
		if err != nil || state.State != "ABORTED" {
			t.Fatal("old timeout attempt still active")
		}
		if _, err := client.Round(context.Background(), id, 1, req); err == nil {
			t.Fatal("old timeout state was reusable")
		}
	}
}

type observedRound struct {
	ctx   cryptoengine.WireContext
	round int
	count int
}
type businessProxy struct {
	server         *httptest.Server
	mu             sync.Mutex
	rounds         map[string]*observedRound
	entered        chan struct{}
	gate           chan struct{}
	enterOnce      sync.Once
	releaseOnce    sync.Once
	delayed        atomic.Bool
	sawRound2State atomic.Bool
	abortCalls     atomic.Int32
}

func (p *businessProxy) releaseGate() { p.releaseOnce.Do(func() { close(p.gate) }) }

func observedConfig(t *testing.T, original network.ClusterConfig, mode string) (network.ClusterConfig, map[int]*businessProxy) {
	t.Helper()
	config := original
	config.Nodes = append([]network.ClusterNode(nil), original.Nodes...)
	proxies := make(map[int]*businessProxy)
	for i, node := range config.Nodes {
		nodeMode := ""
		if mode == "gate-round1" && node.PartyID == 0 || mode != "gate-round1" && node.PartyID == 2 {
			nodeMode = mode
		}
		proxy := newBusinessProxy(t, node.Address, nodeMode)
		proxies[node.PartyID] = proxy
		address, err := url.Parse(proxy.server.URL)
		if err != nil {
			t.Fatal(err)
		}
		port, err := strconv.Atoi(address.Port())
		if err != nil {
			t.Fatal(err)
		}
		config.Nodes[i].Port = port
		config.Nodes[i].Address = proxy.server.URL
	}
	return config, proxies
}

func newBusinessProxy(t *testing.T, origin, mode string) *businessProxy {
	t.Helper()
	p := &businessProxy{rounds: make(map[string]*observedRound), entered: make(chan struct{}), gate: make(chan struct{})}
	upstream := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}
	p.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, network.MaxHTTPBodySize))
		if err != nil {
			w.WriteHeader(502)
			return
		}
		var req network.RoundRequest
		if r.Method == http.MethodPost && json.Unmarshal(body, &req) != nil {
			w.WriteHeader(502)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/internal/round") {
			round, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/internal/round"))
			if err != nil {
				w.WriteHeader(502)
				return
			}
			key := fmt.Sprintf("%s/%d/%d", req.SessionID, req.Attempt, round)
			p.mu.Lock()
			if p.rounds[key] == nil {
				p.rounds[key] = &observedRound{ctx: req.WireContext, round: round}
			}
			p.rounds[key].count++
			p.mu.Unlock()
			if mode == "gate-round1" && round == 1 {
				p.enterOnce.Do(func() { close(p.entered) })
				select {
				case <-p.gate:
				case <-r.Context().Done():
					return
				}
			}
		}
		if r.URL.Path == "/internal/abort" {
			p.abortCalls.Add(1)
		}
		if mode == "unconfirmed-abort" && p.delayed.Load() && (r.URL.Path == "/internal/status" || r.URL.Path == "/internal/abort") {
			w.WriteHeader(503)
			return
		}
		request, err := http.NewRequestWithContext(r.Context(), r.Method, origin+r.URL.Path, bytes.NewReader(body))
		if err != nil {
			w.WriteHeader(502)
			return
		}
		request.Header = r.Header.Clone()
		response, err := upstream.Do(request)
		if err != nil {
			w.WriteHeader(502)
			return
		}
		defer response.Body.Close()
		data, err := io.ReadAll(io.LimitReader(response.Body, network.MaxHTTPBodySize))
		if err != nil {
			w.WriteHeader(502)
			return
		}
		if r.URL.Path == "/internal/status" {
			var state network.StateResponse
			if json.Unmarshal(data, &state) == nil && state.State == "ROUND2_DONE" {
				p.sawRound2State.Store(true)
			}
		}
		if (mode == "timeout-round2" || mode == "unconfirmed-abort") && r.URL.Path == "/internal/round2" && response.StatusCode == 200 && p.delayed.CompareAndSwap(false, true) {
			time.Sleep(time.Second)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(data)
	}))
	t.Cleanup(func() { p.releaseGate(); p.server.Close(); upstream.CloseIdleConnections() })
	return p
}
func (p *businessProxy) roundCounts(hash string) (result [3]int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, round := range p.rounds {
		if round.ctx.MessageHash == hash {
			result[round.round-1] += round.count
		}
	}
	return
}
func (p *businessProxy) sessionsFor(hash string) map[string]bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make(map[string]bool)
	for _, round := range p.rounds {
		if round.ctx.MessageHash == hash {
			result[round.ctx.SessionID] = true
		}
	}
	return result
}
func (p *businessProxy) assertNoRetries(t *testing.T) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, round := range p.rounds {
		if round.count != 1 {
			t.Fatal("same cryptographic Round POST repeated")
		}
	}
}
func (p *businessProxy) totalRounds() int { p.mu.Lock(); defer p.mu.Unlock(); return len(p.rounds) }
