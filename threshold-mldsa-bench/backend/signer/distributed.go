package signer

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
)

type DistributedSigner struct {
	client  *network.Client
	config  *cryptoengine.PublicConfig
	blocked atomic.Bool
}

func NewDistributedSigner(cluster network.ClusterConfig, token string, nodeTimeout time.Duration) (*DistributedSigner, error) {
	config, endpoints, err := cluster.Validate()
	if err != nil {
		return nil, err
	}
	client, err := network.NewClient(endpoints, token, nodeTimeout)
	if err != nil {
		return nil, err
	}
	return &DistributedSigner{client: client, config: config}, nil
}
func (s *DistributedSigner) Close() { s.client.Close() }

func (s *DistributedSigner) Sign(ctx context.Context, message []byte, participants []int) (Result, error) {
	ctx, meter := network.MeasureTraffic(ctx)
	start := time.Now()
	result := Result{KeyID: s.config.KeyID()}
	fail := func(cause error, attempt *network.AttemptError) (Result, error) {
		traffic := meter.Snapshot()
		result.Traffic = &traffic
		result.DurationMs = float64(time.Since(start)) / float64(time.Millisecond)
		failure := &Failure{Result: result, Cause: cause, RequiresReinitialization: s.blocked.Load()}
		if attempt != nil {
			result.SessionID = attempt.SessionID
			result.Attempts = int(attempt.Attempt) + 1
			confirmed := attempt.AbortConfirmed
			failure.AbortConfirmed = &confirmed
			if !confirmed {
				s.blocked.Store(true)
				failure.RequiresReinitialization = true
			}
			failure.Result = result
		}
		return result, failure
	}
	if s.blocked.Load() {
		return fail(ErrReinitializationRequired, nil)
	}
	// Check the pinned selected nodes BEFORE starting any cryptographic session.
	// This never generates new keys or falls back to centralized signing.
	coordinator, err := network.Connect(ctx, s.client, participants, s.config.KeyID())
	if err != nil {
		return fail(fmt.Errorf("%w: %w", ErrUnavailable, err), nil)
	}
	if s.blocked.Load() {
		return fail(ErrReinitializationRequired, nil)
	}
	output, err := coordinator.Sign(ctx, message, participants)
	if err != nil {
		var attempt *network.AttemptError
		_ = errors.As(err, &attempt)
		return fail(err, attempt)
	}
	if s.blocked.Load() {
		return fail(ErrReinitializationRequired, nil)
	}
	traffic := meter.Snapshot()
	return Result{Traffic: &traffic, Signature: output.Signature, KeyID: output.KeyID, Attempts: int(output.Attempt) + 1,
		DurationMs: float64(time.Since(start)) / float64(time.Millisecond), SessionID: output.SessionID, RoundMessages: output.RoundMessages, RoundMessagesKnown: true}, nil
}
