package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"threshold-mldsa/backend/clearing"
	"threshold-mldsa/backend/signer"
	"threshold-mldsa/backend/store"
)

func TestDashboardPublicAPI(t *testing.T) {
	local, err := signer.NewLocalSigner()
	if err != nil {
		t.Fatal(err)
	}
	p := local.PublicDescriptor()
	v, err := signer.NewVerifier(p, p.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(store.New(), local, v, Options{Mode: "local"})
	if err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{"/": "跨银行清算签名监控台", "/app.js": "parseFen", "/style.css": "@media"} {
		r := request(h, "GET", path, nil)
		if r.Code != 200 || !strings.Contains(r.Body.String(), content) || r.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("static asset %s", path)
		}
	}
	for _, path := range []string{"/go.mod", "/cluster.public.json", "/internal/config", "/api/unknown"} {
		if r := request(h, "GET", path, nil); r.Code != 404 {
			t.Fatalf("exposed path %s", path)
		}
	}
	r := request(h, "GET", "/api/cluster", nil)
	var c signer.ClusterStatus
	if json.Unmarshal(r.Body.Bytes(), &c) != nil || c.Mode != "local" || len(c.Nodes) != 0 || c.KeyID != p.KeyID {
		t.Fatal("local baseline falsely reports nodes")
	}
	tx := clearing.Transaction{Version: "clearing-v1", TransactionID: "DASHBOARD", FromBank: "BankA", ToBank: "BankB", AmountFen: 12345, Currency: "CNY", Nonce: "dashboard-nonce"}
	data, _ := json.Marshal(tx)
	if request(h, "POST", "/api/transactions", data).Code != 201 {
		t.Fatal("create")
	}
	for _, body := range []string{`{"participants":[0,2]}`, `{"participants":[0,0,4]}`, `{"participants":[0,2,5]}`, `{"participants":null}`, `{"participants":[0,2,4],"token":"x"}`, `{"participants":[0,2,4]} {}`} {
		if r := request(h, "POST", "/api/transactions/DASHBOARD/sign", []byte(body)); r.Code != 400 {
			t.Fatalf("invalid participant body accepted: %s", body)
		}
	}
	signed := request(h, "POST", "/api/transactions/DASHBOARD/sign", []byte(`{"participants":[4,1,3]}`))
	if signed.Code != 200 {
		t.Fatal(signed.Body.String())
	}
	var result signatureResponse
	_ = json.Unmarshal(signed.Body.Bytes(), &result)
	if !equalInts(result.Participants, []int{1, 3, 4}) || !result.Verified || result.Traffic != nil {
		t.Fatal("wrong selection or fake traffic in local mode")
	}
	verify := func(want bool) {
		data, _ := json.Marshal(tx)
		r := request(h, "POST", "/api/transactions/DASHBOARD/verify", data)
		var got struct{ Verified bool }
		if r.Code != http.StatusOK || json.Unmarshal(r.Body.Bytes(), &got) != nil || got.Verified != want {
			t.Fatalf("verify: %s", r.Body.String())
		}
	}
	verify(true)
	tx.AmountFen++
	verify(false)
	get := request(h, "GET", "/api/transactions/DASHBOARD", nil)
	var original transactionResponse
	_ = json.Unmarshal(get.Body.Bytes(), &original)
	if original.Transaction.AmountFen != 12345 || !original.Verified {
		t.Fatal("candidate verification mutated original transaction")
	}
	if request(h, "POST", "/api/transactions/DASHBOARD/sign", nil).Code != 409 {
		t.Fatal("duplicate not rejected")
	}
}
