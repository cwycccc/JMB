package clearing

import (
	"bytes"
	"testing"
)

func validTransaction() Transaction {
	return Transaction{
		Version:       "clearing-v1",
		TransactionID: "TX001",
		FromBank:      "BankA",
		ToBank:        "BankB",
		AmountFen:     100000000,
		Currency:      "CNY",
		Nonce:         "nonce-001",
	}
}

func TestSigningMessageIsDeterministic(t *testing.T) {
	tx := validTransaction()
	first, err := SigningMessage(tx)
	if err != nil {
		t.Fatalf("first SigningMessage: %v", err)
	}
	second, err := SigningMessage(tx)
	if err != nil {
		t.Fatalf("second SigningMessage: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("the same transaction produced different signing messages")
	}
	if !bytes.HasPrefix(first, []byte(signingDomain)) {
		t.Fatal("signing message is missing its domain separator")
	}
}

func TestSigningMessageRejectsInvalidTransaction(t *testing.T) {
	tx := validTransaction()
	tx.AmountFen = 0
	if _, err := SigningMessage(tx); err == nil {
		t.Fatal("zero amount was accepted")
	}
}
