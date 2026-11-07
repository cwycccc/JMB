package network

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"threshold-mldsa/cryptoengine"
)

// Coordinator owns public material and HTTP clients only.
type Coordinator struct {
	client   *Client
	config   *cryptoengine.PublicConfig
	combiner *cryptoengine.Coordinator
}

type Result struct {
	SessionID     string             `json:"session_id"`
	Attempt       uint64             `json:"attempt"`
	KeyID         string             `json:"key_id"`
	Participants  []int              `json:"participants"`
	Signature     []byte             `json:"signature_base64"`
	Verified      bool               `json:"verified"`
	DurationMS    float64            `json:"duration_ms"`
	RoundMessages [3]int             `json:"round_message_counts"`
	Transcript    [3][]RoundResponse `json:"-"`
}

// AttemptError describes a failed attempt. No cryptographic POST is retried.
// AbortConfirmed=false requires stopping/reinitializing the controlled cluster.
type AttemptError struct {
	SessionID      string
	Attempt        uint64
	StateQueried   bool
	AbortConfirmed bool
	Cause          error
}

func (e *AttemptError) Error() string {
	return fmt.Sprintf("signing attempt %s/%d stopped (state queried=%t, abort confirmed=%t): %v", e.SessionID, e.Attempt, e.StateQueried, e.AbortConfirmed, e.Cause)
}
func (e *AttemptError) Unwrap() error { return e.Cause }

// Connect checks selected nodes and reconstructs ONLY the verified public config.
func Connect(ctx context.Context, client *Client, participants []int, expectedKeyID string) (*Coordinator, error) {
	if client == nil || len(participants) == 0 {
		return nil, errors.New("invalid coordinator setup")
	}
	var config *cryptoengine.PublicConfig
	for _, id := range participants {
		health, err := client.Health(ctx, id)
		if err != nil {
			return nil, err
		}
		if health.PartyID != id || health.KeyID != expectedKeyID || !health.KeyLoaded || health.Status != "CRYPTO_READY" {
			return nil, errors.New("node identity or readiness mismatch")
		}
		current, err := client.PublicConfig(ctx, id, expectedKeyID)
		if err != nil {
			return nil, err
		}
		if health.Threshold != current.Threshold() || health.Total != current.TotalParties() {
			return nil, errors.New("node parameter mismatch")
		}
		config = current
	}
	ordered := append([]int(nil), participants...)
	slices.Sort(ordered)
	probe := cryptoengine.WireContext{Version: cryptoengine.WireVersion, SessionID: "probe", KeyID: expectedKeyID, Participants: ordered, MessageHash: cryptoengine.MessageHash(nil)}
	if err := probe.Validate(config); err != nil {
		return nil, err
	}
	combiner, err := cryptoengine.NewCoordinator(config)
	if err != nil {
		return nil, err
	}
	return &Coordinator{client: client, config: config, combiner: combiner}, nil
}

func (c *Coordinator) Verify(message, signature []byte) bool {
	return c.combiner.VerifyMessage(message, signature)
}

func (c *Coordinator) Sign(ctx context.Context, message []byte, participants []int) (*Result, error) {
	if len(message) > cryptoengine.MaxSigningMessageSize {
		return nil, errors.New("message too large")
	}
	message = append([]byte(nil), message...)
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	ordered := append([]int(nil), participants...)
	slices.Sort(ordered)
	wire := cryptoengine.WireContext{Version: cryptoengine.WireVersion, SessionID: hex.EncodeToString(nonce[:]),
		KeyID: c.config.KeyID(), Participants: ordered, MessageHash: cryptoengine.MessageHash(message)}
	if err := wire.Validate(c.config); err != nil {
		return nil, err
	}
	start := time.Now()
	counts := [3]int{}
	for attempt := uint64(0); attempt < 570; attempt++ {
		wire.Attempt = attempt
		req := RoundRequest{WireContext: wire}
		var transcript [3][]RoundResponse
		for round := 1; round <= 3; round++ {
			if round > 1 {
				payload, err := cryptoengine.EncodeRoundMessages(c.config, wire, round-1, transcript[round-2])
				if err != nil {
					return nil, c.stopAttempt(req, err, true)
				}
				req.Payload = payload
			}
			req.Message = nil
			if round == 2 {
				req.Message = message
			}
			for _, id := range ordered {
				response, err := c.client.Round(ctx, id, round, req)
				if err == nil {
					err = cryptoengine.ValidatePublicRoundMessage(c.config, wire, round, response)
				}
				if err != nil {
					return nil, c.stopAttempt(req, err, true)
				}
				transcript[round-1] = append(transcript[round-1], response)
				counts[round-1]++
			}
		}
		commitments := make([][]byte, len(ordered))
		responses := make([][]byte, len(ordered))
		for i := range ordered {
			commitments[i] = transcript[1][i].Payload
			responses[i] = transcript[2][i].Payload
		}
		signature, err := c.combiner.CombineAndVerify(message, commitments, responses)
		if errors.Is(err, cryptoengine.ErrCombineRejected) {
			stopped := c.stopAttempt(req, err, false)
			if !stopped.AbortConfirmed {
				return nil, stopped
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue // Increment Attempt; every node executes a fresh Round1.
		}
		if err != nil {
			return nil, c.stopAttempt(req, err, false)
		}
		return &Result{SessionID: wire.SessionID, Attempt: attempt, KeyID: wire.KeyID, Participants: ordered,
			Signature: signature, Verified: true, DurationMS: float64(time.Since(start)) / float64(time.Millisecond),
			RoundMessages: counts, Transcript: transcript}, nil
	}
	return nil, errors.New("signing attempt limit reached; last attempt was aborted")
}

func (c *Coordinator) stopAttempt(req RoundRequest, cause error, query bool) *AttemptError {
	result := &AttemptError{SessionID: req.SessionID, Attempt: req.Attempt, Cause: cause, StateQueried: query, AbortConfirmed: true}
	// Cleanup uses a new context even if the signing request deadline expired.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if query {
		for _, id := range req.Participants {
			_, _ = c.client.State(ctx, id, req)
		}
	}
	for _, id := range req.Participants {
		if _, err := c.client.Abort(ctx, id, req); err != nil {
			result.AbortConfirmed = false
		}
	}
	return result
}
