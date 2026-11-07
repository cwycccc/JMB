package cryptoengine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"slices"
)

const WireVersion = 1
const MaxWireBatchSize = 4 << 20
const MaxSigningMessageSize = 64 << 10

var ErrInvalidWireMessage = errors.New("cryptoengine: invalid public protocol message")

// WireContext binds public messages to one key, business message and attempt.
// Participants MUST already be sorted, distinct, zero-based Party IDs.
type WireContext struct {
	Version      uint16 `json:"version"`
	SessionID    string `json:"session_id"`
	Attempt      uint64 `json:"attempt"`
	KeyID        string `json:"key_id"`
	Participants []int  `json:"participants"`
	MessageHash  string `json:"message_hash"`
}

// PublicRoundMessage contains only a public commitment/hash/response. It cannot
// encode StRound1, StRound2, a private key, or commitment randomness.
type PublicRoundMessage struct {
	WireContext
	PartyID int    `json:"party_id"`
	Round   int    `json:"round"`
	Payload []byte `json:"payload"`
}

func MessageHash(message []byte) string {
	h := sha256.Sum256(message)
	return hex.EncodeToString(h[:])
}

func (c WireContext) Session() SigningSession {
	return SigningSession{ID: c.SessionID, Attempt: c.Attempt, Participants: append([]int(nil), c.Participants...)}
}

func (c WireContext) Equal(other WireContext) bool {
	return c.Version == other.Version && c.SessionID == other.SessionID && c.Attempt == other.Attempt &&
		c.KeyID == other.KeyID && c.MessageHash == other.MessageHash && slices.Equal(c.Participants, other.Participants)
}

func (c WireContext) Validate(config *PublicConfig) error {
	if config == nil || c.Version != WireVersion || c.KeyID != config.keyID || len(c.SessionID) == 0 || len(c.SessionID) > 128 ||
		len(c.MessageHash) != 64 || len(c.Participants) != config.threshold {
		return ErrInvalidWireMessage
	}
	for _, ch := range c.SessionID {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_') {
			return ErrInvalidWireMessage
		}
	}
	hash, err := hex.DecodeString(c.MessageHash)
	if err != nil || hex.EncodeToString(hash) != c.MessageHash {
		return ErrInvalidWireMessage
	}
	for i, id := range c.Participants {
		if id < 0 || id >= config.parties || (i > 0 && id <= c.Participants[i-1]) {
			return ErrInvalidWireMessage
		}
	}
	return nil
}

func ValidatePublicRoundMessage(config *PublicConfig, ctx WireContext, round int, m PublicRoundMessage) error {
	if err := ctx.Validate(config); err != nil {
		return err
	}
	if !ctx.Equal(m.WireContext) || m.Round != round || !slices.Contains(ctx.Participants, m.PartyID) {
		return ErrInvalidWireMessage
	}
	want := 0
	switch round {
	case 1:
		want = 32
	case 2:
		want = config.params.CommitmentSize()
	case 3:
		want = config.params.ResponseSize()
	default:
		return ErrInvalidWireMessage
	}
	if len(m.Payload) != want {
		return ErrInvalidWireMessage
	}
	return nil
}

func EncodeRoundMessages(config *PublicConfig, ctx WireContext, round int, messages []PublicRoundMessage) ([]byte, error) {
	if err := validateBatch(config, ctx, round, messages); err != nil {
		return nil, err
	}
	data, err := json.Marshal(messages)
	if err != nil || len(data) > MaxWireBatchSize {
		return nil, ErrInvalidWireMessage
	}
	return data, nil
}

func DecodeRoundMessages(config *PublicConfig, ctx WireContext, round int, data []byte) ([][]byte, error) {
	if len(data) == 0 || len(data) > MaxWireBatchSize {
		return nil, ErrInvalidWireMessage
	}
	var messages []PublicRoundMessage
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&messages) != nil || d.Decode(new(any)) != io.EOF {
		return nil, ErrInvalidWireMessage
	}
	if err := validateBatch(config, ctx, round, messages); err != nil {
		return nil, err
	}
	payloads := make([][]byte, len(messages))
	for i := range messages {
		payloads[i] = messages[i].Payload
	}
	return payloads, nil
}

func validateBatch(config *PublicConfig, ctx WireContext, round int, messages []PublicRoundMessage) error {
	if err := ctx.Validate(config); err != nil {
		return err
	}
	if len(messages) != len(ctx.Participants) {
		return ErrInvalidWireMessage
	}
	for i, message := range messages {
		if message.PartyID != ctx.Participants[i] {
			return ErrInvalidWireMessage
		}
		if err := ValidatePublicRoundMessage(config, ctx, round, message); err != nil {
			return err
		}
	}
	return nil
}
