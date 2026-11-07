package cryptoengine

// PublicConfig exports ONLY the public configuration of a local Engine for
// independent verification by the regression baseline. Existing signing code
// and its single-use round-state handling are unchanged.
func (e *Engine) PublicConfig() *PublicConfig {
	commitments := make([][]byte, e.parties)
	for id := range e.shares {
		packed := packThresholdShare(&e.shares[id], e.threshold, e.parties)
		commitments[id] = shareCommitment(id, packed)
		clear(packed)
	}
	return &PublicConfig{threshold: e.threshold, parties: e.parties, params: e.params, publicKey: e.publicKey,
		commitments: commitments, keyID: configurationKeyID(e.threshold, e.parties, e.publicKey.Bytes(), commitments)}
}
