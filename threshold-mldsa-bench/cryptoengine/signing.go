package cryptoengine

import (
	"errors"
	"fmt"
	"sort"

	"github.com/cloudflare/circl/sign/thmldsa/thmldsa44"
)

const maxSigningAttempts = 570

// SignMessage signs message with the selected participants using a
// process-local 3-of-5 ML-DSA-44 engine. Participant IDs are zero-based.
func SignMessage(message []byte, participants []int) ([]byte, error) {
	engine, err := getDefaultEngine()
	if err != nil {
		return nil, err
	}
	return engine.SignMessage(message, participants)
}

// VerifyMessage verifies a signature produced by the package-level
// SignMessage function.
func VerifyMessage(message, signature []byte) bool {
	engine, err := getDefaultEngine()
	if err != nil {
		return false
	}
	return engine.VerifyMessage(message, signature)
}

// SignMessage executes Round1, Round2, Round3 and Combine in that order.
// A rejected Combine starts a completely new attempt with fresh Round1 state.
func (e *Engine) SignMessage(message []byte, participants []int) ([]byte, error) {
	if e == nil {
		return nil, errors.New("cryptoengine: nil engine")
	}

	ordered, active, err := e.signingSet(participants)
	if err != nil {
		return nil, err
	}

	for attempt := 1; attempt <= maxSigningAttempts; attempt++ {
		round1Messages := make([][]byte, len(ordered))
		round1States := make([]thmldsa44.StRound1, len(ordered))
		for i, participant := range ordered {
			round1Messages[i], round1States[i], err = thmldsa44.Round1(
				&e.shares[participant],
				e.params,
			)
			if err != nil {
				return nil, fmt.Errorf("round 1 for participant %d: %w", participant, err)
			}
		}

		round2Messages := make([][]byte, len(ordered))
		round2States := make([]thmldsa44.StRound2, len(ordered))
		for i, participant := range ordered {
			round2Messages[i], round2States[i], err = thmldsa44.Round2(
				&e.shares[participant],
				active,
				message,
				nil,
				round1Messages,
				&round1States[i],
				e.params,
			)
			if err != nil {
				return nil, fmt.Errorf("round 2 for participant %d: %w", participant, err)
			}
		}

		round3Messages := make([][]byte, len(ordered))
		for i, participant := range ordered {
			round3Messages[i], err = thmldsa44.Round3(
				&e.shares[participant],
				round2Messages,
				&round2States[i],
				e.params,
			)
			if err != nil {
				return nil, fmt.Errorf("round 3 for participant %d: %w", participant, err)
			}
		}

		signature := make([]byte, thmldsa44.SignatureSize)
		if !thmldsa44.Combine(
			e.publicKey,
			message,
			nil,
			round2Messages,
			round3Messages,
			signature,
			e.params,
		) {
			continue
		}

		if !e.VerifyMessage(message, signature) {
			return nil, errors.New("cryptoengine: combined signature failed verification")
		}
		return signature, nil
	}

	return nil, fmt.Errorf("cryptoengine: could not combine a signature after %d fresh attempts", maxSigningAttempts)
}

// VerifyMessage checks that signature is a valid ML-DSA-44 signature for
// message under this engine's public key.
func (e *Engine) VerifyMessage(message, signature []byte) bool {
	if e == nil || len(signature) != thmldsa44.SignatureSize {
		return false
	}
	return thmldsa44.Verify(e.publicKey, message, nil, signature)
}

func (e *Engine) signingSet(participants []int) ([]int, uint8, error) {
	if len(participants) != e.threshold {
		return nil, 0, fmt.Errorf(
			"cryptoengine: need exactly %d participants, got %d",
			e.threshold,
			len(participants),
		)
	}

	ordered := append([]int(nil), participants...)
	sort.Ints(ordered)

	var active uint8
	for i, participant := range ordered {
		if participant < 0 || participant >= e.parties {
			return nil, 0, fmt.Errorf(
				"cryptoengine: participant %d is outside [0,%d)",
				participant,
				e.parties,
			)
		}
		if i > 0 && participant == ordered[i-1] {
			return nil, 0, fmt.Errorf("cryptoengine: duplicate participant %d", participant)
		}
		active |= uint8(1 << uint(participant))
	}

	return ordered, active, nil
}
