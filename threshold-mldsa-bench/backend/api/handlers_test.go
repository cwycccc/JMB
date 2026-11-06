package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"threshold-mldsa/backend/clearing"
	transactionstore "threshold-mldsa/backend/store"
	"threshold-mldsa/cryptoengine"
)

func TestTransactionHTTPWorkflow(t *testing.T) {
	store := transactionstore.New()
	handler := NewHandler(store)
	tx := clearing.Transaction{
		Version:       "clearing-v1",
		TransactionID: "TX001",
		FromBank:      "BankA",
		ToBank:        "BankB",
		AmountFen:     100000000,
		Currency:      "CNY",
		Nonce:         "nonce-001",
	}

	createBody, err := json.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}
	create := request(handler, http.MethodPost, "/api/transactions", createBody)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}

	sign := request(handler, http.MethodPost, "/api/transactions/TX001/sign", nil)
	if sign.Code != http.StatusOK {
		t.Fatalf("sign status = %d, body = %s", sign.Code, sign.Body.String())
	}
	var signed signatureResponse
	if err := json.Unmarshal(sign.Body.Bytes(), &signed); err != nil {
		t.Fatalf("decode sign response: %v", err)
	}
	if signed.Status != transactionstore.StatusVerified || !signed.Verified {
		t.Fatalf("unexpected sign result: %+v", signed)
	}
	if signed.DurationMS <= 0 {
		t.Fatalf("duration_ms was not measured: %f", signed.DurationMS)
	}
	if !equalInts(signed.Participants, []int{0, 2, 4}) {
		t.Fatalf("participants = %v, want [0 2 4]", signed.Participants)
	}

	get := request(handler, http.MethodGet, "/api/transactions/TX001", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", get.Code, get.Body.String())
	}
	var fetched transactionResponse
	if err := json.Unmarshal(get.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if fetched.Transaction.AmountFen != 100000000 {
		t.Fatalf("amount_fen = %d, want 100000000", fetched.Transaction.AmountFen)
	}
	if fetched.SignatureBase64 == "" || fetched.SignatureBase64 != signed.SignatureBase64 {
		t.Fatal("GET did not return the stored signature")
	}

	signature, err := base64.StdEncoding.DecodeString(signed.SignatureBase64)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	tampered := tx
	tampered.AmountFen = 200000000
	tamperedMessage, err := clearing.SigningMessage(tampered)
	if err != nil {
		t.Fatalf("tampered signing message: %v", err)
	}
	if cryptoengine.VerifyMessage(tamperedMessage, signature) {
		t.Fatal("original signature verified after transaction amount was modified")
	}
}

func request(handler http.Handler, method, target string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
