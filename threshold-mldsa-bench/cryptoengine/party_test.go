package cryptoengine

import (
	"errors"
	"fmt"
	"testing"
)

type testSystem struct {
	parties     []Party
	coordinator *Coordinator
}

func newTestSystem(t *testing.T) *testSystem {
	t.Helper()

	config, shares, err := TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatalf("trusted key setup: %v", err)
	}

	coordinator, err := NewCoordinator(config)
	if err != nil {
		t.Fatalf("new coordinator: %v", err)
	}

	parties := make([]Party, len(shares))
	for i, share := range shares {
		parties[i], err = NewParty(config, share)
		if err != nil {
			t.Fatalf("new party %d: %v", i, err)
		}
	}
	return &testSystem{parties: parties, coordinator: coordinator}
}

func (s *testSystem) sign(
	message []byte,
	participants []int,
	sessionID string,
) ([]byte, error) {
	return s.signFrom(message, participants, sessionID, 0)
}

func (s *testSystem) signFrom(
	message []byte,
	participants []int,
	sessionID string,
	firstAttempt uint64,
) ([]byte, error) {
	for offset := uint64(0); offset < maxSigningAttempts; offset++ {
		attempt := firstAttempt + offset
		session := SigningSession{
			ID:           sessionID,
			Attempt:      attempt,
			Participants: participants,
		}
		ordered := session.OrderedParticipants()

		round1Messages := make([][]byte, len(ordered))
		for i, id := range ordered {
			var err error
			round1Messages[i], err = s.parties[id].Round1(session)
			if err != nil {
				return nil, err
			}
		}

		round2Messages := make([][]byte, len(ordered))
		for i, id := range ordered {
			var err error
			round2Messages[i], err = s.parties[id].Round2(session, message, round1Messages)
			if err != nil {
				return nil, err
			}
		}

		round3Messages := make([][]byte, len(ordered))
		for i, id := range ordered {
			var err error
			round3Messages[i], err = s.parties[id].Round3(session, round2Messages)
			if err != nil {
				return nil, err
			}
		}

		signature, err := s.coordinator.CombineAndVerify(message, round2Messages, round3Messages)
		if errors.Is(err, ErrCombineRejected) {
			continue
		}
		return signature, err
	}
	return nil, fmt.Errorf("signature was not produced after %d attempts", maxSigningAttempts)
}

func TestIndependentPartiesCanSign(t *testing.T) {
	tests := []struct {
		name         string
		participants []int
	}{
		{name: "parties 0 1 2", participants: []int{0, 1, 2}},
		{name: "parties 0 2 4", participants: []int{0, 2, 4}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system := newTestSystem(t)
			signature, err := system.sign([]byte(messageA), test.participants, t.Name())
			if err != nil {
				t.Fatalf("distributed sign: %v", err)
			}
			if !system.coordinator.VerifyMessage([]byte(messageA), signature) {
				t.Fatal("coordinator rejected the distributed signature")
			}
		})
	}
}

func TestTwoPartiesCannotStartThresholdThreeSession(t *testing.T) {
	system := newTestSystem(t)
	session := SigningSession{
		ID:           t.Name(),
		Attempt:      0,
		Participants: []int{0, 1},
	}

	if _, err := system.parties[0].Round1(session); !errors.Is(err, ErrInvalidSigningSet) {
		t.Fatalf("two-party Round1 error = %v, want ErrInvalidSigningSet", err)
	}
	if _, err := system.coordinator.CombineAndVerify([]byte(messageA), make([][]byte, 2), make([][]byte, 2)); !errors.Is(err, ErrInvalidSigningSet) {
		t.Fatalf("two-party combine error = %v, want ErrInvalidSigningSet", err)
	}
}

func TestModifiedMessageFailsVerification(t *testing.T) {
	system := newTestSystem(t)
	signature, err := system.sign([]byte(messageA), []int{0, 2, 4}, t.Name())
	if err != nil {
		t.Fatalf("distributed sign: %v", err)
	}
	if system.coordinator.VerifyMessage([]byte(messageB), signature) {
		t.Fatal("message A signature incorrectly verified for message B")
	}
}

func TestAbortDoesNotReuseOneTimeState(t *testing.T) {
	system := newTestSystem(t)
	participants := []int{0, 2, 4}
	aborted := SigningSession{
		ID:           t.Name(),
		Attempt:      0,
		Participants: participants,
	}

	for _, id := range aborted.OrderedParticipants() {
		if _, err := system.parties[id].Round1(aborted); err != nil {
			t.Fatalf("party %d round 1: %v", id, err)
		}
		if err := system.parties[id].Abort(aborted); err != nil {
			t.Fatalf("party %d abort: %v", id, err)
		}
		if _, err := system.parties[id].Round1(aborted); !errors.Is(err, ErrSessionAlreadyUsed) {
			t.Fatalf("party %d reused aborted state: got %v, want ErrSessionAlreadyUsed", id, err)
		}
	}

	// The same business session may retry only under a new attempt number.
	signature, err := system.signFrom([]byte(messageA), participants, t.Name(), 1)
	if err != nil {
		t.Fatalf("fresh attempt after abort: %v", err)
	}
	if !system.coordinator.VerifyMessage([]byte(messageA), signature) {
		t.Fatal("fresh attempt after abort did not verify")
	}
}

func TestPartyKeyShareCanOnlyInitializeOneParty(t *testing.T) {
	config, shares, err := TrustedKeySetup(3, 5)
	if err != nil {
		t.Fatalf("trusted key setup: %v", err)
	}
	if _, err := NewParty(config, shares[0]); err != nil {
		t.Fatalf("first NewParty: %v", err)
	}
	if _, err := NewParty(config, shares[0]); err == nil {
		t.Fatal("the same key share initialized two Party objects")
	}
}
