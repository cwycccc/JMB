package signernode

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
)

func TestNodeAuthenticationAndSessionBindings(t *testing.T) {
	config, shares, err := cryptoengine.TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, share := range shares {
			share.Destroy()
		}
	}()
	party, err := cryptoengine.NewParty(config, shares[0])
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("a", 64)
	handler, err := NewHandler(config, party, token)
	if err != nil {
		t.Fatal(err)
	}
	wire := cryptoengine.WireContext{Version: cryptoengine.WireVersion, SessionID: "node-bindings", KeyID: config.KeyID(), Participants: []int{0, 2, 4}, MessageHash: cryptoengine.MessageHash([]byte("message"))}
	call := func(operation string, req network.RoundRequest, credential string, remote string) *httptest.ResponseRecorder {
		t.Helper()
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodPost, "/internal/"+operation, bytes.NewReader(data))
		r.RemoteAddr = remote
		r.Header.Set("Authorization", "Bearer "+credential)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	req := network.RoundRequest{WireContext: wire}
	if w := call("round1", req, "wrong", "127.0.0.1:1"); w.Code != 401 {
		t.Fatal("missing credential accepted")
	}
	if w := call("round1", req, token, "192.0.2.1:1"); w.Code != 401 {
		t.Fatal("non-loopback coordinator accepted")
	}
	bad := req
	bad.KeyID = "wrong"
	if w := call("round1", bad, token, "127.0.0.1:1"); w.Code != 400 {
		t.Fatal("wrong KeyID accepted")
	}
	if w := call("round1", req, token, "127.0.0.1:1"); w.Code != 200 {
		t.Fatalf("round1 failed: %d", w.Code)
	}
	if w := call("round1", req, token, "127.0.0.1:1"); w.Code != 409 {
		t.Fatal("duplicate round1 accepted")
	}
	bad = req
	bad.Participants = []int{0, 1, 4}
	bad.Message = []byte("message")
	if w := call("round2", bad, token, "127.0.0.1:1"); w.Code != 409 {
		t.Fatal("changed participant set accepted")
	}
	bad = req
	bad.Attempt++
	if w := call("round1", bad, token, "127.0.0.1:1"); w.Code != 409 {
		t.Fatal("new attempt started while prior state active")
	}
	bad = req
	bad.Message = []byte("altered")
	if w := call("round2", bad, token, "127.0.0.1:1"); w.Code != 400 {
		t.Fatal("message hash mismatch accepted")
	}
	if w := call("abort", req, token, "127.0.0.1:1"); w.Code != 200 {
		t.Fatal("abort failed")
	}
	if w := call("round1", req, token, "127.0.0.1:1"); w.Code != 409 {
		t.Fatal("aborted attempt was revived")
	}
	if w := call("round3", req, token, "127.0.0.1:1"); w.Code != 409 {
		t.Fatal("aborted round state reused")
	}
	status := call("status", req, token, "127.0.0.1:1")
	var state network.StateResponse
	if json.Unmarshal(status.Body.Bytes(), &state) != nil || state.State != "ABORTED" {
		t.Fatal("abort state was not reported")
	}
	req.Attempt++
	if w := call("round1", req, token, "127.0.0.1:1"); w.Code != 200 {
		t.Fatal("fresh attempt was rejected")
	}
	// An early abort tombstones a delayed Round1 that has not executed yet.
	req.SessionID = "abort-before-round1"
	if w := call("abort", req, token, "127.0.0.1:1"); w.Code != 200 {
		t.Fatal("early abort failed")
	}
	if w := call("round1", req, token, "127.0.0.1:1"); w.Code != 409 {
		t.Fatal("delayed round1 revived an aborted attempt")
	}
}
