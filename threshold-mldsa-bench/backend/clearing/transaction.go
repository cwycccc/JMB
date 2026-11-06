// Package clearing defines the canonical business message signed by the
// threshold ML-DSA layer.
package clearing

import (
	"encoding/json"
	"errors"
)

const signingDomain = "JMB-CLEARING-V1\x00"

// Transaction is a simulated inter-bank clearing transaction. Monetary
// values are stored as integer fen to avoid floating-point rounding errors.
type Transaction struct {
	Version       string `json:"version"`
	TransactionID string `json:"transaction_id"`
	FromBank      string `json:"from_bank"`
	ToBank        string `json:"to_bank"`
	AmountFen     int64  `json:"amount_fen"`
	Currency      string `json:"currency"`
	Nonce         string `json:"nonce"`
}

// SigningMessage validates tx and returns its deterministic, domain-separated
// representation. Transaction contains no maps, so encoding/json emits its
// fields in declaration order.
func SigningMessage(tx Transaction) ([]byte, error) {
	if tx.Version != "clearing-v1" {
		return nil, errors.New("invalid transaction version")
	}
	if tx.TransactionID == "" ||
		tx.FromBank == "" ||
		tx.ToBank == "" ||
		tx.AmountFen <= 0 ||
		tx.Currency != "CNY" ||
		tx.Nonce == "" {
		return nil, errors.New("invalid transaction fields")
	}

	data, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}

	message := make([]byte, 0, len(signingDomain)+len(data))
	message = append(message, signingDomain...)
	message = append(message, data...)
	return message, nil
}
