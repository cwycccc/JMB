package cryptoengine

import "testing"

const (
	messageA = "TX001|BankA|BankB|1000000.00|CNY"
	messageB = "TX002|BankA|BankB|2000000.00|CNY"
)

// Keep the requested package-level API from changing accidentally.
var _ func([]byte, []int) ([]byte, error) = SignMessage

func TestSignAndVerifyTwoDifferentMessages(t *testing.T) {
	participants := []int{0, 1, 2}

	signatureA, err := SignMessage([]byte(messageA), participants)
	if err != nil {
		t.Fatalf("sign message A: %v", err)
	}
	signatureB, err := SignMessage([]byte(messageB), participants)
	if err != nil {
		t.Fatalf("sign message B: %v", err)
	}

	if !VerifyMessage([]byte(messageA), signatureA) {
		t.Fatal("message A signature did not verify for message A")
	}
	if !VerifyMessage([]byte(messageB), signatureB) {
		t.Fatal("message B signature did not verify for message B")
	}
	if VerifyMessage([]byte(messageB), signatureA) {
		t.Fatal("message A signature incorrectly verified for message B")
	}
}

func TestEngineSupportsNonContiguousParticipants(t *testing.T) {
	var seed [32]byte
	copy(seed[:], "cryptoengine test seed")

	engine, err := NewEngineFromSeed(3, 5, seed)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}

	signature, err := engine.SignMessage([]byte(messageA), []int{4, 0, 2})
	if err != nil {
		t.Fatalf("sign with participants 0, 2 and 4: %v", err)
	}
	if !engine.VerifyMessage([]byte(messageA), signature) {
		t.Fatal("signature from participants 0, 2 and 4 did not verify")
	}
}
