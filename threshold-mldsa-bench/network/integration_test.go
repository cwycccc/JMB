package network_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
)

const message = "TX001|BankA|BankB|1000000.00|CNY"
const alteredMessage = "TX001|BankA|BankB|2000000.00|CNY"

type cluster struct {
	KeyID  string `json:"key_id"`
	Status string `json:"status"`
	Nodes  []struct {
		PartyID int `json:"party_id"`
		PID     int `json:"pid"`
		Port    int `json:"port"`
	} `json:"nodes"`
	token     string
	endpoints map[int]string
}

func startCluster(t *testing.T) *cluster {
	t.Helper()
	launcher, signer := os.Getenv("C23_LAUNCHER_BINARY"), os.Getenv("C23_SIGNER_BINARY")
	if launcher == "" || signer == "" {
		t.Skip("set C23_LAUNCHER_BINARY and C23_SIGNER_BINARY to run the real five-process integration test")
	}
	base := freeFivePorts(t)
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		t.Fatal(err)
	}
	token := hex.EncodeToString(secret[:])
	clear(secret[:])
	command := exec.Command(launcher, "-signer="+signer, fmt.Sprintf("-base-port=%d", base))
	command.Env = append(os.Environ(), network.CoordinatorTokenEnv+"="+token)
	output, err := command.Output()
	if err != nil {
		t.Fatal("launcher failed; check executable paths and free localhost ports")
	}
	var result cluster
	if json.Unmarshal(output, &result) != nil {
		t.Fatal("invalid launch summary")
	}
	t.Cleanup(func() {
		for _, node := range result.Nodes {
			if p, err := os.FindProcess(node.PID); err == nil {
				_ = p.Kill()
				_ = p.Release()
			}
		}
	})
	if result.Status != "CRYPTO_READY" || len(result.Nodes) != 5 {
		t.Fatal("five processes were not initialized")
	}
	result.token = token
	result.endpoints = make(map[int]string)
	pids := make(map[int]bool)
	for _, node := range result.Nodes {
		if pids[node.PID] || node.PID == command.Process.Pid {
			t.Fatal("nodes did not use independent processes")
		}
		pids[node.PID] = true
		result.endpoints[node.PartyID] = fmt.Sprintf("http://127.0.0.1:%d", node.Port)
	}
	return &result
}

func freeFivePorts(t *testing.T) int {
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
	t.Fatal("could not find five available consecutive localhost ports")
	return 0
}

func clientFor(t *testing.T, c *cluster, endpoints map[int]string, timeout time.Duration) *network.Client {
	t.Helper()
	client, err := network.NewClient(endpoints, c.token, timeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	return client
}

func TestNetworkFiveProcesses(t *testing.T) {
	c := startCluster(t)
	participants := []int{0, 2, 4}
	client := clientFor(t, c, c.endpoints, 3*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	for id := 0; id < 5; id++ {
		health, err := client.Health(ctx, id)
		if err != nil || !health.KeyLoaded || health.Status != "CRYPTO_READY" || health.KeyID != c.KeyID || health.PartyID != id {
			t.Fatalf("party %d not crypto-ready", id)
		}
	}
	t.Run("three_network_rounds_combine_verify_and_tamper", func(t *testing.T) {
		coordinator, err := network.Connect(ctx, client, participants, c.KeyID)
		if err != nil {
			t.Fatal(err)
		}
		result, err := coordinator.Sign(ctx, []byte(message), participants)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Verified || !coordinator.Verify([]byte(message), result.Signature) || coordinator.Verify([]byte(alteredMessage), result.Signature) {
			t.Fatal("network signature acceptance check failed")
		}
		for round := range result.Transcript {
			if len(result.Transcript[round]) != 3 {
				t.Fatal("round transcript missing HTTP responses")
			}
		}
		// Independently combine the final actual HTTP responses with public-only
		// configuration. There is no access to any child's private share here.
		config, err := client.PublicConfig(ctx, 0, c.KeyID)
		if err != nil {
			t.Fatal(err)
		}
		combiner, err := cryptoengine.NewCoordinator(config)
		if err != nil {
			t.Fatal(err)
		}
		commitments, responses := make([][]byte, 3), make([][]byte, 3)
		for i := 0; i < 3; i++ {
			commitments[i] = result.Transcript[1][i].Payload
			responses[i] = result.Transcript[2][i].Payload
		}
		signature, err := combiner.CombineAndVerify([]byte(message), commitments, responses)
		if err != nil || !bytes.Equal(signature, result.Signature) {
			t.Fatal("signature did not come from HTTP transcript")
		}
		t.Logf("five processes ready; participants=[0,2,4]; received=%v; verified=true; modified_amount_verified=false; duration_ms=%.4f", result.RoundMessages, result.DurationMS)
	})
	t.Run("internal_endpoints_require_coordinator_auth", func(t *testing.T) {
		response, err := http.Get(c.endpoints[0] + "/internal/config")
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatal("unauthenticated internal request accepted")
		}
	})
	t.Run("combine_rejection_aborts_and_uses_fresh_attempt", func(t *testing.T) {
		endpoints := cloneEndpoints(c.endpoints)
		var proxies []*faultProxy
		for _, id := range participants {
			proxy := newFaultProxy(t, c.endpoints[id], "corrupt-first-response")
			endpoints[id] = proxy.server.URL
			proxies = append(proxies, proxy)
		}
		faultClient := clientFor(t, c, endpoints, 3*time.Second)
		coordinator, err := network.Connect(ctx, faultClient, participants, c.KeyID)
		if err != nil {
			t.Fatal(err)
		}
		result, err := coordinator.Sign(ctx, []byte(message), participants)
		if err != nil {
			t.Fatal(err)
		}
		if result.Attempt < 1 || !coordinator.Verify([]byte(message), result.Signature) {
			t.Fatal("combine rejection was not followed by a fresh successful attempt")
		}
		req := network.RoundRequest{WireContext: result.Transcript[0][0].WireContext}
		req.Attempt = 0
		for _, id := range participants {
			state, err := client.State(ctx, id, req)
			if err != nil || state.State != "ABORTED" {
				t.Fatal("rejected attempt was not aborted")
			}
			if _, err := client.Round(ctx, id, 1, req); err == nil {
				t.Fatal("rejected old Round1 state was reused")
			}
		}
		for _, proxy := range proxies {
			proxy.assertNoRoundRetries(t)
		}
		t.Logf("fault-injected attempt=0 rejected and aborted; verified new attempt=%d", result.Attempt)
	})
	t.Run("lost_round2_response_queries_then_aborts_without_retry", func(t *testing.T) {
		endpoints := cloneEndpoints(c.endpoints)
		proxy := newFaultProxy(t, c.endpoints[2], "lose-round2-response")
		endpoints[2] = proxy.server.URL
		faultClient := clientFor(t, c, endpoints, 300*time.Millisecond)
		coordinator, err := network.Connect(ctx, faultClient, participants, c.KeyID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = coordinator.Sign(ctx, []byte(message), participants)
		var failure *network.AttemptError
		if !errors.As(err, &failure) || !failure.StateQueried || !failure.AbortConfirmed {
			t.Fatalf("uncertain request was not safely terminated: %v", err)
		}
		if !proxy.sawRound2State.Load() || proxy.statusCalls.Load() == 0 || proxy.abortCalls.Load() == 0 {
			t.Fatal("coordinator did not query executed Round2 before abort")
		}
		req := network.RoundRequest{WireContext: cryptoengine.WireContext{Version: cryptoengine.WireVersion, SessionID: failure.SessionID, Attempt: failure.Attempt, KeyID: c.KeyID, Participants: participants, MessageHash: cryptoengine.MessageHash([]byte(message))}}
		for _, id := range participants {
			state, err := client.State(ctx, id, req)
			if err != nil || state.State != "ABORTED" {
				t.Fatal("timed-out attempt was not aborted everywhere")
			}
			if _, err := client.Round(ctx, id, 1, req); err == nil {
				t.Fatal("timed-out attempt could be reused")
			}
		}
		result, err := coordinator.Sign(ctx, []byte(message), participants)
		if err != nil || result.SessionID == failure.SessionID || !coordinator.Verify([]byte(message), result.Signature) {
			t.Fatal("fresh session after timeout did not sign")
		}
		proxy.assertNoRoundRetries(t)
		t.Log("executed Round2 response lost; status=ROUND2_DONE observed; abort confirmed; no Round POST retry; new session verified")
	})
	t.Run("unconfirmed_abort_stops_without_new_attempt", func(t *testing.T) {
		endpoints := cloneEndpoints(c.endpoints)
		proxy := newFaultProxy(t, c.endpoints[2], "unconfirmed-abort")
		endpoints[2] = proxy.server.URL
		faultClient := clientFor(t, c, endpoints, 300*time.Millisecond)
		coordinator, err := network.Connect(ctx, faultClient, participants, c.KeyID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = coordinator.Sign(ctx, []byte(message), participants)
		var failure *network.AttemptError
		if !errors.As(err, &failure) || !failure.StateQueried || failure.AbortConfirmed {
			t.Fatalf("unconfirmed abort was not surfaced: %v", err)
		}
		proxy.assertNoRoundRetries(t)
		proxy.mu.Lock()
		defer proxy.mu.Unlock()
		if len(proxy.posts) != 2 {
			t.Fatal("coordinator started extra rounds or attempts after unconfirmed abort")
		}
		t.Log("status and abort unavailable; abort_confirmed=false; coordinator stopped without starting a new attempt")
	})
}

func cloneEndpoints(original map[int]string) map[int]string {
	result := make(map[int]string, len(original))
	for id, endpoint := range original {
		result[id] = endpoint
	}
	return result
}

// Fault injection forwards requests to REAL independent signer processes.
// Only public response bytes are corrupted or delayed; no private state is read.
type faultProxy struct {
	server         *httptest.Server
	mu             sync.Mutex
	posts          map[string]int
	delayed        atomic.Bool
	statusCalls    atomic.Int32
	abortCalls     atomic.Int32
	sawRound2State atomic.Bool
}

func newFaultProxy(t *testing.T, origin, mode string) *faultProxy {
	t.Helper()
	p := &faultProxy{posts: make(map[string]int)}
	upstream := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true}}
	p.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, network.MaxHTTPBodySize))
		if err != nil {
			w.WriteHeader(502)
			return
		}
		var roundReq network.RoundRequest
		if r.Method == http.MethodPost && json.Unmarshal(body, &roundReq) != nil {
			w.WriteHeader(502)
			return
		}
		if r.URL.Path == "/internal/status" {
			p.statusCalls.Add(1)
		}
		if r.URL.Path == "/internal/abort" {
			p.abortCalls.Add(1)
		}
		if mode == "unconfirmed-abort" && p.delayed.Load() && (r.URL.Path == "/internal/status" || r.URL.Path == "/internal/abort") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/internal/round1" || r.URL.Path == "/internal/round2" || r.URL.Path == "/internal/round3" {
			key := r.URL.Path + "/" + roundReq.SessionID + "/" + strconv.FormatUint(roundReq.Attempt, 10)
			p.mu.Lock()
			p.posts[key]++
			p.mu.Unlock()
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
		if mode == "corrupt-first-response" && r.URL.Path == "/internal/round3" && roundReq.Attempt == 0 && response.StatusCode == 200 {
			var result network.RoundResponse
			if json.Unmarshal(data, &result) != nil {
				w.WriteHeader(502)
				return
			}
			clear(result.Payload)
			data, err = json.Marshal(result)
			if err != nil {
				w.WriteHeader(502)
				return
			}
		}
		if (mode == "lose-round2-response" || mode == "unconfirmed-abort") && r.URL.Path == "/internal/round2" && response.StatusCode == 200 && p.delayed.CompareAndSwap(false, true) {
			// The origin has already executed Round2. Lose only its response.
			time.Sleep(time.Second)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(data)
	}))
	t.Cleanup(func() { p.server.Close(); upstream.CloseIdleConnections() })
	return p
}

func (p *faultProxy) assertNoRoundRetries(t *testing.T) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, count := range p.posts {
		if count != 1 {
			t.Fatal("cryptographic Round HTTP POST was repeated")
		}
	}
}
