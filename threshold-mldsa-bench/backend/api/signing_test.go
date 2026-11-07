package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"threshold-mldsa/backend/clearing"
	"threshold-mldsa/backend/signer"
	transactionstore "threshold-mldsa/backend/store"
)

type serviceFunc func(context.Context, []byte, []int) (signer.Result, error)

func (f serviceFunc) Sign(ctx context.Context, message []byte, participants []int) (signer.Result, error) {
	return f(ctx, message, participants)
}

func localFixture(t *testing.T) (*signer.LocalSigner, *signer.PublicVerifier) {
	t.Helper()
	local, err := signer.NewLocalSigner()
	if err != nil {
		t.Fatal(err)
	}
	public := local.PublicDescriptor()
	verifier, err := signer.NewVerifier(public, public.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	return local, verifier
}

func createFixtureTransaction(t *testing.T, handler http.Handler, id string) {
	t.Helper()
	data, err := json.Marshal(clearing.Transaction{Version: "clearing-v1", TransactionID: id, FromBank: "BankA", ToBank: "BankB", AmountFen: 100000000, Currency: "CNY", Nonce: id + "-nonce"})
	if err != nil {
		t.Fatal(err)
	}
	response := request(handler, http.MethodPost, "/api/transactions", data)
	if response.Code != http.StatusCreated {
		t.Fatalf("create transaction status %d", response.Code)
	}
}

func TestDuplicateSignRequestsUseOneServiceCall(t *testing.T) {
	local, verifier := localFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var calls atomic.Int32
	service := serviceFunc(func(ctx context.Context, message []byte, participants []int) (signer.Result, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
		case <-ctx.Done():
			return signer.Result{}, ctx.Err()
		}
		return local.Sign(ctx, message, participants)
	})
	store := transactionstore.New()
	handler, err := NewHandler(store, service, verifier, Options{Mode: "local"})
	if err != nil {
		t.Fatal(err)
	}
	createFixtureTransaction(t, handler, "DUP001")
	first := make(chan *httptest.ResponseRecorder, 1)
	go func() { first <- request(handler, http.MethodPost, "/api/transactions/DUP001/sign", nil) }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("first sign did not start")
	}
	if response := request(handler, http.MethodPost, "/api/transactions/DUP001/sign", nil); response.Code != http.StatusConflict {
		t.Fatal("concurrent duplicate sign was not rejected")
	}
	record, err := store.Get("DUP001")
	if err != nil || record.Status != transactionstore.StatusSigning {
		t.Fatal("transaction did not stay in SIGNING")
	}
	releaseOnce.Do(func() { close(release) })
	if response := <-first; response.Code != http.StatusOK {
		t.Fatal("first sign failed")
	}
	if response := request(handler, http.MethodPost, "/api/transactions/DUP001/sign", nil); response.Code != http.StatusConflict {
		t.Fatal("verified transaction was signed again")
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate submission created multiple signing service calls")
	}
}

func TestBackendIndependentlyRejectsBadSignatureOrKeyID(t *testing.T) {
	for _, mode := range []string{"invalid-signature", "wrong-key-id"} {
		t.Run(mode, func(t *testing.T) {
			local, verifier := localFixture(t)
			service := serviceFunc(func(ctx context.Context, message []byte, participants []int) (signer.Result, error) {
				result, err := local.Sign(ctx, message, participants)
				if err != nil {
					return result, err
				}
				if mode == "invalid-signature" {
					result.Signature[0] ^= 1
				} else {
					result.KeyID = strings.Repeat("0", 64)
				}
				return result, nil // A service success must not imply business verification.
			})
			store := transactionstore.New()
			handler, err := NewHandler(store, service, verifier, Options{Mode: "local"})
			if err != nil {
				t.Fatal(err)
			}
			createFixtureTransaction(t, handler, "BAD001")
			response := request(handler, http.MethodPost, "/api/transactions/BAD001/sign", nil)
			if response.Code != http.StatusBadGateway {
				t.Fatal("backend accepted an invalid signing result")
			}
			record, err := store.Get("BAD001")
			if err != nil || record.Status != transactionstore.StatusFailed || record.Verified || len(record.Signature) != 0 {
				t.Fatal("invalid result was not recorded as FAILED")
			}
		})
	}
}
