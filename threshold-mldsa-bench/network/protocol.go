// Package network implements the localhost public-message transport and
// coordinator. It has no private shares and never calls centralized SignMessage.
package network

import (
	"encoding/hex"
	"errors"

	"threshold-mldsa/cryptoengine"
)

const MaxHTTPBodySize = 8 << 20
const CoordinatorTokenEnv = "JMB_COORDINATOR_TOKEN"

type RoundRequest struct {
	cryptoengine.WireContext
	Payload []byte `json:"payload,omitempty"`
	Message []byte `json:"message,omitempty"`
}

type RoundResponse = cryptoengine.PublicRoundMessage

type Health struct {
	PartyID   int    `json:"party_id"`
	Status    string `json:"status"`
	KeyLoaded bool   `json:"key_loaded"`
	KeyID     string `json:"key_id,omitempty"`
	Threshold int    `json:"threshold,omitempty"`
	Total     int    `json:"total_parties,omitempty"`
}

type StateResponse struct {
	cryptoengine.WireContext
	PartyID int    `json:"party_id"`
	State   string `json:"state"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidateToken requires a caller-provided 256-bit hexadecimal credential.
func ValidateToken(token string) error {
	data, err := hex.DecodeString(token)
	if err != nil || len(data) != 32 {
		return errors.New("coordinator token must be 64 hexadecimal characters")
	}
	return nil
}
