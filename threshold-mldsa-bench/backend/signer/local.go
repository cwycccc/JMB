package signer

import (
	"context"
	"time"

	"threshold-mldsa/cryptoengine"
)

// LocalSigner is an explicit opt-in regression baseline, never a fallback for
// DistributedSigner. Only this implementation creates local private material.
type LocalSigner struct {
	engine *cryptoengine.Engine
	public *cryptoengine.PublicConfig
}

func NewLocalSigner() (*LocalSigner, error) {
	engine, err := cryptoengine.NewEngine(3, 5)
	if err != nil {
		return nil, err
	}
	return &LocalSigner{engine: engine, public: engine.PublicConfig()}, nil
}
func (s *LocalSigner) PublicDescriptor() cryptoengine.PublicDescriptor {
	return s.public.PublicDescriptor()
}
func (s *LocalSigner) Sign(ctx context.Context, message []byte, participants []int) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	start := time.Now()
	signature, err := s.engine.SignMessage(message, participants)
	if err == nil {
		err = ctx.Err()
	}
	result := Result{KeyID: s.public.KeyID(), DurationMs: float64(time.Since(start)) / float64(time.Millisecond)}
	if err != nil {
		return result, &Failure{Result: result, Cause: err}
	}
	result.Signature = signature
	return result, nil
}
