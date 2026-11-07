package cryptoengine

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha3"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/bits"
	"sort"

	"github.com/cloudflare/circl/sign/thmldsa/thmldsa44"
)

const (
	ShareEnvelopeVersion = 1
	MaxShareEnvelopeSize = 64 << 10
	// These describe THIS repository's threshold ML-DSA-44 Pack layout,
	// not an ordinary ML-DSA private key. Revisit when the library changes.
	thresholdShareHeaderSize = 1 + 32 + 32 + 64
	thresholdShareRecordSize = 1 + 96*(4+4)
)

var ErrInvalidShareEnvelope = errors.New("cryptoengine: invalid share envelope")

// ShareEnvelope is only for the launcher's private, per-child stdin pipe.
// ShareCommitments are public hashes binding each node's packed share to KeyID.
// The pipe plus a separately pinned expected KeyID is the trust boundary;
// this plaintext encoding is neither encrypted nor a general key-storage API.
type ShareEnvelope struct {
	Version          uint16   `json:"version"`
	KeyID            string   `json:"key_id"`
	PartyID          int      `json:"party_id"`
	Threshold        int      `json:"threshold"`
	Total            int      `json:"total"`
	PublicKey        []byte   `json:"public_key"`
	ShareCommitments [][]byte `json:"share_commitments"`
	SecretShare      []byte   `json:"secret_share"`
}

func (e ShareEnvelope) String() string   { return "ShareEnvelope{secret material redacted}" }
func (e ShareEnvelope) GoString() string { return e.String() }

// ExportPartyShare moves a share into a bounded wire envelope and invalidates
// the original handle. Callers must clear returned bytes after pipe delivery.
func ExportPartyShare(config *PublicConfig, share *PartyKeyShare) ([]byte, error) {
	if config == nil || config.publicKey == nil || config.keyID == "" {
		return nil, ErrInvalidShareEnvelope
	}
	id, key, err := share.take(config)
	if err != nil {
		return nil, err
	}
	defer func() { *key = thmldsa44.PrivateKey{} }()
	packed := packThresholdShare(key, config.threshold, config.parties)
	defer clear(packed)
	if !bytes.Equal(shareCommitment(id, packed), config.commitments[id]) {
		return nil, ErrInvalidShareEnvelope
	}
	return json.Marshal(ShareEnvelope{
		Version:          ShareEnvelopeVersion,
		KeyID:            config.keyID,
		PartyID:          id,
		Threshold:        config.threshold,
		Total:            config.parties,
		PublicKey:        config.publicKey.Bytes(),
		ShareCommitments: config.commitments,
		SecretShare:      packed,
	})
}

// ImportPartyShare validates and reconstructs a Party from trusted pipe data.
// For a process with an expected setup, use ImportPartyShareForKey instead.
// Import does not provide global replay protection against copying the input.
func ImportPartyShare(data []byte, expectedPartyID int) (*PublicConfig, Party, error) {
	return importPartyShare(data, expectedPartyID, "")
}

// ImportPartyShareForKey additionally pins the expected initialization batch.
func ImportPartyShareForKey(data []byte, expectedPartyID int, expectedKeyID string) (*PublicConfig, Party, error) {
	if len(expectedKeyID) != sha256.Size*2 {
		return nil, nil, ErrInvalidShareEnvelope
	}
	return importPartyShare(data, expectedPartyID, expectedKeyID)
}

func importPartyShare(data []byte, expectedPartyID int, expectedKeyID string) (*PublicConfig, Party, error) {
	if len(data) == 0 || len(data) > MaxShareEnvelopeSize {
		return nil, nil, ErrInvalidShareEnvelope
	}
	var envelope ShareEnvelope
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		clear(envelope.SecretShare)
		return nil, nil, ErrInvalidShareEnvelope
	}
	defer clear(envelope.SecretShare)
	if decoder.Decode(new(any)) != io.EOF {
		return nil, nil, ErrInvalidShareEnvelope
	}
	if envelope.Version != ShareEnvelopeVersion ||
		expectedPartyID < 0 || expectedPartyID >= envelope.Total ||
		envelope.PartyID != expectedPartyID ||
		len(envelope.PublicKey) != thmldsa44.PublicKeySize ||
		len(envelope.ShareCommitments) != envelope.Total {
		return nil, nil, ErrInvalidShareEnvelope
	}
	params, err := thresholdParams(envelope.Threshold, envelope.Total)
	if err != nil {
		return nil, nil, ErrInvalidShareEnvelope
	}
	for _, commitment := range envelope.ShareCommitments {
		if len(commitment) != sha256.Size {
			return nil, nil, ErrInvalidShareEnvelope
		}
	}
	keyID := configurationKeyID(envelope.Threshold, envelope.Total, envelope.PublicKey, envelope.ShareCommitments)
	if keyID != envelope.KeyID || (expectedKeyID != "" && keyID != expectedKeyID) {
		return nil, nil, ErrInvalidShareEnvelope
	}
	if !validPackedThresholdShare(envelope.SecretShare, expectedPartyID, envelope.Threshold, envelope.Total, envelope.PublicKey) ||
		!bytes.Equal(shareCommitment(expectedPartyID, envelope.SecretShare), envelope.ShareCommitments[expectedPartyID]) {
		return nil, nil, ErrInvalidShareEnvelope
	}

	publicKey := new(thmldsa44.PublicKey)
	if err := publicKey.UnmarshalBinary(envelope.PublicKey); err != nil {
		return nil, nil, ErrInvalidShareEnvelope
	}
	key := new(thmldsa44.PrivateKey)
	// Only reach the library's unchecked decoder after all bounds, masks and
	// coefficient ranges have been checked.
	key.Unpack(envelope.SecretShare)
	identity := &setupIdentity{marker: 1}
	config := &PublicConfig{
		threshold: envelope.Threshold, parties: envelope.Total,
		params: params, publicKey: publicKey, setup: identity,
		keyID: keyID, commitments: envelope.ShareCommitments,
	}
	party, err := NewParty(config, &PartyKeyShare{id: expectedPartyID, key: key, setup: identity})
	if err != nil {
		return nil, nil, err
	}
	return config, party, nil
}

func thresholdShareSize(threshold, total int) int {
	count := 0
	for mask := 1; mask < 1<<uint(total); mask++ {
		if mask&1 != 0 && bits.OnesCount(uint(mask)) == total-threshold+1 {
			count++
		}
	}
	return thresholdShareHeaderSize + count*thresholdShareRecordSize
}

func packThresholdShare(key *thmldsa44.PrivateKey, threshold, total int) []byte {
	packed := make([]byte, thresholdShareSize(threshold, total))
	key.Pack(packed)
	// Native Pack walks a Go map. Sort records to give commitments a stable
	// representation without modifying the upstream library.
	count := (len(packed) - thresholdShareHeaderSize) / thresholdShareRecordSize
	records := make([][]byte, count)
	for i := range records {
		start := thresholdShareHeaderSize + i*thresholdShareRecordSize
		records[i] = append([]byte(nil), packed[start:start+thresholdShareRecordSize]...)
	}
	sort.Slice(records, func(i, j int) bool { return records[i][0] < records[j][0] })
	for i, record := range records {
		copy(packed[thresholdShareHeaderSize+i*thresholdShareRecordSize:], record)
		clear(record)
	}
	return packed
}

func validPackedThresholdShare(data []byte, partyID, threshold, total int, publicKey []byte) bool {
	if len(data) != thresholdShareSize(threshold, total) || int(data[0]) != partyID {
		return false
	}
	if !bytes.Equal(data[1:33], publicKey[:32]) ||
		!bytes.Equal(data[65:129], sha3.SumSHAKE256(publicKey, 64)) {
		return false
	}
	previousMask := 0
	for offset := thresholdShareHeaderSize; offset < len(data); offset += thresholdShareRecordSize {
		mask := int(data[offset])
		if mask <= previousMask || mask >= 1<<uint(total) || mask&(1<<uint(partyID)) == 0 ||
			bits.OnesCount(uint(mask)) != total-threshold+1 {
			return false
		}
		previousMask = mask
		// Each 3-byte group encodes eight 3-bit coefficients in [0, 2*eta].
		for i := offset + 1; i < offset+thresholdShareRecordSize; i += 3 {
			word := uint32(data[i]) | uint32(data[i+1])<<8 | uint32(data[i+2])<<16
			for j := 0; j < 8; j++ {
				if word&7 > 4 {
					return false
				}
				word >>= 3
			}
		}
	}
	return true
}

func shareCommitment(partyID int, packed []byte) []byte {
	h := sha256.New()
	h.Write([]byte("JMB-THMLDSA44-SHARE-V1\x00"))
	h.Write([]byte{byte(partyID)})
	h.Write(packed)
	return h.Sum(nil)
}

func configurationKeyID(threshold, total int, publicKey []byte, commitments [][]byte) string {
	h := sha256.New()
	h.Write([]byte("JMB-THMLDSA44-KEY-V1\x00"))
	h.Write([]byte{ShareEnvelopeVersion, byte(threshold), byte(total)})
	h.Write(publicKey)
	for _, commitment := range commitments {
		h.Write(commitment)
	}
	return hex.EncodeToString(h.Sum(nil))
}
