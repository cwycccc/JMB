// Package signer adapts local and HTTP threshold signing for the business API.
package signer

import (
	"context"
	"errors"
	"net"

	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
)

type Result struct {
	Signature          []byte
	KeyID              string
	Attempts           int // Zero means the legacy local API does not report the count.
	DurationMs         float64
	SessionID          string
	RoundMessages      [3]int
	RoundMessagesKnown bool
	Traffic            *network.Traffic
}

type Service interface {
	Sign(ctx context.Context, message []byte, participants []int) (Result, error)
}

// Verifier is deliberately independent from Service and the network result's
// Verified flag. It owns only the pinned system public configuration.
type Verifier interface {
	KeyID() string
	Verify(keyID string, message, signature []byte) bool
}

type PublicVerifier struct {
	keyID    string
	verifier *cryptoengine.Coordinator
}

func NewVerifier(descriptor cryptoengine.PublicDescriptor, expectedKeyID string) (*PublicVerifier, error) {
	config, err := cryptoengine.ImportPublicConfig(descriptor, expectedKeyID)
	if err != nil {
		return nil, err
	}
	verifier, err := cryptoengine.NewCoordinator(config)
	if err != nil {
		return nil, err
	}
	return &PublicVerifier{keyID: config.KeyID(), verifier: verifier}, nil
}
func (v *PublicVerifier) KeyID() string { return v.keyID }
func (v *PublicVerifier) Verify(keyID string, message, signature []byte) bool {
	return keyID == v.keyID && v.verifier.VerifyMessage(message, signature)
}

var ErrUnavailable = errors.New("selected signing nodes unavailable or inconsistent")
var ErrReinitializationRequired = errors.New("cluster reinitialization required: previous abort could not be confirmed")

type Failure struct {
	Result                   Result
	Cause                    error
	AbortConfirmed           *bool
	RequiresReinitialization bool
}

func (e *Failure) Error() string { return e.Cause.Error() }
func (e *Failure) Unwrap() error { return e.Cause }

func IsTimeout(err error) bool {
	var timeout net.Error
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.As(err, &timeout) && timeout.Timeout()
}
