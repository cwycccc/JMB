package cryptoengine

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/cloudflare/circl/sign/thmldsa/thmldsa44"
)

// PublicConfig is the non-secret system configuration shared by Parties and
// the Coordinator. It deliberately exposes no private-key material.
type PublicConfig struct {
	threshold   int
	parties     int
	params      *thmldsa44.ThresholdParams
	publicKey   *thmldsa44.PublicKey
	setup       *setupIdentity
	keyID       string
	commitments [][]byte
}

// PartyKeyShare is an in-memory hand-off object produced by TrustedKeySetup.
// NewParty consumes it, so the same secret share cannot initialize two Party
// objects accidentally. ExportPartyShare consumes this handle as well.
type PartyKeyShare struct {
	mu    sync.Mutex
	id    int
	key   *thmldsa44.PrivateKey
	setup *setupIdentity
}

// setupIdentity must have non-zero size: Go permits distinct pointers to
// zero-size values to compare equal, which would weaken setup matching.
type setupIdentity struct {
	marker byte
}

// TrustedKeySetup creates one public configuration and one private share for
// each Party. The trusted initializer must hand each PartyKeyShare to its
// matching node and then discard the returned slice.
func TrustedKeySetup(threshold, parties int) (*PublicConfig, []*PartyKeyShare, error) {
	params, err := thresholdParams(threshold, parties)
	if err != nil {
		return nil, nil, err
	}

	publicKey, privateKeys, err := thmldsa44.GenerateThresholdKey(rand.Reader, params)
	if err != nil {
		return nil, nil, fmt.Errorf("trusted threshold ML-DSA-44 key setup: %w", err)
	}

	identity := &setupIdentity{marker: 1}
	config := &PublicConfig{
		threshold: threshold,
		parties:   parties,
		params:    params,
		publicKey: publicKey,
		setup:     identity,
	}

	shares := make([]*PartyKeyShare, parties)
	for id := range privateKeys {
		key := privateKeys[id]
		privateKeys[id] = thmldsa44.PrivateKey{}
		shares[id] = &PartyKeyShare{
			id:    id,
			key:   &key,
			setup: identity,
		}
	}
	config.commitments = make([][]byte, parties)
	for id, share := range shares {
		packed := packThresholdShare(share.key, threshold, parties)
		config.commitments[id] = shareCommitment(id, packed)
		clear(packed)
	}
	config.keyID = configurationKeyID(threshold, parties, publicKey.Bytes(), config.commitments)
	return config, shares, nil
}

// KeyID identifies the public key, parameters and per-Party share commitments.
// It is public and may be supplied to a child process as an expected value.
func (c *PublicConfig) KeyID() string     { return c.keyID }
func (c *PublicConfig) Threshold() int    { return c.threshold }
func (c *PublicConfig) TotalParties() int { return c.parties }

// Destroy releases an undistributed share handle. It cannot revoke a share
// already imported by another process or guarantee erasure of Go heap copies.
func (s *PartyKeyShare) Destroy() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.key != nil {
		*s.key = thmldsa44.PrivateKey{}
		s.key = nil
	}
}

// PartyID returns the zero-based node ID this share belongs to.
func (s *PartyKeyShare) PartyID() int {
	if s == nil {
		return -1
	}
	return s.id
}

func (s *PartyKeyShare) take(config *PublicConfig) (int, *thmldsa44.PrivateKey, error) {
	if s == nil {
		return 0, nil, errors.New("cryptoengine: nil party key share")
	}
	if config == nil || config.setup == nil {
		return 0, nil, errors.New("cryptoengine: nil public configuration")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.setup != config.setup {
		return 0, nil, errors.New("cryptoengine: key share and public configuration come from different setups")
	}
	if s.key == nil {
		return 0, nil, errors.New("cryptoengine: party key share has already been consumed")
	}

	key := s.key
	s.key = nil
	return s.id, key, nil
}
