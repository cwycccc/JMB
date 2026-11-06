// Package cryptoengine provides a message-oriented wrapper around the
// threshold ML-DSA-44 signing protocol used by the benchmark.
package cryptoengine

import (
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/cloudflare/circl/sign/thmldsa/thmldsa44"
)

const (
	defaultThreshold = 3
	defaultParties   = 5
)

// Engine owns one threshold ML-DSA-44 public key and all of its private-key
// shares. Participant IDs are zero-based and range from 0 to parties-1.
type Engine struct {
	threshold int
	parties   int
	params    *thmldsa44.ThresholdParams
	publicKey *thmldsa44.PublicKey
	shares    []thmldsa44.PrivateKey
}

// NewEngine creates a fresh threshold ML-DSA-44 key set using crypto/rand.
func NewEngine(threshold, parties int) (*Engine, error) {
	params, err := thresholdParams(threshold, parties)
	if err != nil {
		return nil, err
	}

	publicKey, shares, err := thmldsa44.GenerateThresholdKey(rand.Reader, params)
	if err != nil {
		return nil, fmt.Errorf("generate threshold ML-DSA-44 keys: %w", err)
	}

	return newEngine(threshold, parties, params, publicKey, shares), nil
}

// NewEngineFromSeed deterministically derives a threshold ML-DSA-44 key set.
// It is useful for reproducible tests. Production callers should normally use
// NewEngine so that key generation uses crypto/rand.
func NewEngineFromSeed(threshold, parties int, seed [thmldsa44.SeedSize]byte) (*Engine, error) {
	params, err := thresholdParams(threshold, parties)
	if err != nil {
		return nil, err
	}

	publicKey, shares := thmldsa44.NewThresholdKeysFromSeed(&seed, params)
	return newEngine(threshold, parties, params, publicKey, shares), nil
}

func thresholdParams(threshold, parties int) (*thmldsa44.ThresholdParams, error) {
	if threshold < 2 {
		return nil, fmt.Errorf("threshold must be at least 2: got %d", threshold)
	}
	if threshold > parties {
		return nil, fmt.Errorf("threshold %d exceeds party count %d", threshold, parties)
	}
	if parties > 6 {
		return nil, fmt.Errorf("party count must not exceed 6: got %d", parties)
	}

	params, err := thmldsa44.GetThresholdParams(uint8(threshold), uint8(parties))
	if err != nil {
		return nil, fmt.Errorf("get threshold ML-DSA-44 parameters: %w", err)
	}
	return params, nil
}

func newEngine(
	threshold, parties int,
	params *thmldsa44.ThresholdParams,
	publicKey *thmldsa44.PublicKey,
	shares []thmldsa44.PrivateKey,
) *Engine {
	return &Engine{
		threshold: threshold,
		parties:   parties,
		params:    params,
		publicKey: publicKey,
		shares:    shares,
	}
}

var defaultState struct {
	sync.Once
	engine *Engine
	err    error
}

// getDefaultEngine returns a process-local 3-of-5 engine used by the
// package-level SignMessage and VerifyMessage helpers.
func getDefaultEngine() (*Engine, error) {
	defaultState.Do(func() {
		defaultState.engine, defaultState.err = NewEngine(defaultThreshold, defaultParties)
	})
	return defaultState.engine, defaultState.err
}
