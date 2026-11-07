package cryptoengine

import (
	"crypto/sha256"
	"errors"

	"github.com/cloudflare/circl/sign/thmldsa/thmldsa44"
)

// PublicDescriptor contains no private share and can be sent to a coordinator.
type PublicDescriptor struct {
	Version          uint16   `json:"version"`
	KeyID            string   `json:"key_id"`
	Threshold        int      `json:"threshold"`
	Total            int      `json:"total"`
	PublicKey        []byte   `json:"public_key"`
	ShareCommitments [][]byte `json:"share_commitments"`
}

func (c *PublicConfig) PublicDescriptor() PublicDescriptor {
	commitments := make([][]byte, len(c.commitments))
	for i := range commitments {
		commitments[i] = append([]byte(nil), c.commitments[i]...)
	}
	return PublicDescriptor{ShareEnvelopeVersion, c.keyID, c.threshold, c.parties, c.publicKey.Bytes(), commitments}
}

// ImportPublicConfig checks a separately pinned KeyID; it creates no secret keys.
func ImportPublicConfig(d PublicDescriptor, expectedKeyID string) (*PublicConfig, error) {
	invalid := errors.New("cryptoengine: invalid public configuration")
	if d.Version != ShareEnvelopeVersion || len(expectedKeyID) != 64 ||
		d.KeyID != expectedKeyID || len(d.PublicKey) != thmldsa44.PublicKeySize {
		return nil, invalid
	}
	params, err := thresholdParams(d.Threshold, d.Total)
	if err != nil || len(d.ShareCommitments) != d.Total {
		return nil, invalid
	}
	for _, commitment := range d.ShareCommitments {
		if len(commitment) != sha256.Size {
			return nil, invalid
		}
	}
	if configurationKeyID(d.Threshold, d.Total, d.PublicKey, d.ShareCommitments) != expectedKeyID {
		return nil, invalid
	}
	pk := new(thmldsa44.PublicKey)
	if err := pk.UnmarshalBinary(d.PublicKey); err != nil {
		return nil, invalid
	}
	commitments := make([][]byte, d.Total)
	for i := range commitments {
		commitments[i] = append([]byte(nil), d.ShareCommitments[i]...)
	}
	return &PublicConfig{threshold: d.Threshold, parties: d.Total, params: params,
		publicKey: pk, keyID: d.KeyID, commitments: commitments}, nil
}
