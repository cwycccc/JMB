package cryptoengine

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/cloudflare/circl/sign/thmldsa/thmldsa44"
)

var (
	// ErrInvalidSigningSet means that the participant list does not describe
	// exactly one valid threshold signing set.
	ErrInvalidSigningSet = errors.New("cryptoengine: invalid signing set")

	// ErrSessionAlreadyUsed is returned for replayed, completed, failed or
	// aborted signing attempts. A retry must use a new Attempt value.
	ErrSessionAlreadyUsed = errors.New("cryptoengine: signing session attempt already used")

	// ErrWrongRound means a protocol message arrived out of order.
	ErrWrongRound = errors.New("cryptoengine: wrong protocol round")

	// ErrCombineRejected means that the probabilistic threshold attempt did
	// not produce a combinable signature. All Parties must restart at Round1
	// with a new Attempt value.
	ErrCombineRejected = errors.New("cryptoengine: signature combine rejected")
)

// SigningSession identifies one and only one threshold signing attempt.
// Participants contains zero-based Party IDs. Protocol message slices must be
// transmitted in the ascending order returned by OrderedParticipants.
type SigningSession struct {
	ID           string
	Attempt      uint64
	Participants []int
}

// OrderedParticipants returns a sorted copy of the participant IDs.
func (s SigningSession) OrderedParticipants() []int {
	ordered := append([]int(nil), s.Participants...)
	sort.Ints(ordered)
	return ordered
}

// Party performs the secret operations for exactly one threshold key share.
// Implementations retain one-time round state internally and never return it
// to the Coordinator or transport layer.
type Party interface {
	ID() int

	Round1(session SigningSession) ([]byte, error)

	Round2(
		session SigningSession,
		message []byte,
		round1Messages [][]byte,
	) ([]byte, error)

	Round3(
		session SigningSession,
		round2Messages [][]byte,
	) ([]byte, error)

	Abort(session SigningSession) error
}

type sessionKey struct {
	id      string
	attempt uint64
}

type sessionPhase uint8

const (
	phaseRound1 sessionPhase = iota + 1
	phaseRound2
	phaseCompleted
	phaseFailed
	phaseAborted
)

type partySession struct {
	participants []int
	active       uint8
	phase        sessionPhase
	round1       thmldsa44.StRound1
	round2       thmldsa44.StRound2
}

type localParty struct {
	mu       sync.Mutex
	id       int
	config   *PublicConfig
	keyShare *thmldsa44.PrivateKey
	sessions map[sessionKey]*partySession
}

// NewParty consumes one PartyKeyShare and creates an isolated Party that owns
// no other node's private material.
func NewParty(config *PublicConfig, share *PartyKeyShare) (Party, error) {
	id, privateKey, err := share.take(config)
	if err != nil {
		return nil, err
	}
	if id < 0 || id >= config.parties {
		return nil, fmt.Errorf("cryptoengine: party ID %d is outside [0,%d)", id, config.parties)
	}

	return &localParty{
		id:       id,
		config:   config,
		keyShare: privateKey,
		sessions: make(map[sessionKey]*partySession),
	}, nil
}

func (p *localParty) ID() int {
	return p.id
}

func (p *localParty) Round1(session SigningSession) ([]byte, error) {
	key, ordered, active, err := p.validateSession(session)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.sessions[key]; exists {
		return nil, ErrSessionAlreadyUsed
	}

	message, state, err := thmldsa44.Round1(p.keyShare, p.config.params)
	if err != nil {
		p.sessions[key] = terminalSession(ordered, active, phaseFailed)
		return nil, fmt.Errorf("party %d round 1: %w", p.id, err)
	}
	p.sessions[key] = &partySession{
		participants: ordered,
		active:       active,
		phase:        phaseRound1,
		round1:       state,
	}
	return message, nil
}

func (p *localParty) Round2(
	session SigningSession,
	message []byte,
	round1Messages [][]byte,
) ([]byte, error) {
	key, ordered, active, err := p.validateSession(session)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state, err := p.stateForRound(key, ordered, phaseRound1)
	if err != nil {
		return nil, err
	}
	if len(round1Messages) != len(ordered) {
		p.makeTerminal(state, phaseFailed)
		return nil, fmt.Errorf("party %d round 2: got %d round-1 messages, want %d", p.id, len(round1Messages), len(ordered))
	}

	round2Message, round2State, err := thmldsa44.Round2(
		p.keyShare,
		active,
		message,
		nil,
		round1Messages,
		&state.round1,
		p.config.params,
	)
	if err != nil {
		p.makeTerminal(state, phaseFailed)
		return nil, fmt.Errorf("party %d round 2: %w", p.id, err)
	}

	state.phase = phaseRound2
	state.round1 = thmldsa44.StRound1{}
	state.round2 = round2State
	return round2Message, nil
}

func (p *localParty) Round3(
	session SigningSession,
	round2Messages [][]byte,
) ([]byte, error) {
	key, ordered, _, err := p.validateSession(session)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state, err := p.stateForRound(key, ordered, phaseRound2)
	if err != nil {
		return nil, err
	}
	if len(round2Messages) != len(ordered) {
		p.makeTerminal(state, phaseFailed)
		return nil, fmt.Errorf("party %d round 3: got %d round-2 messages, want %d", p.id, len(round2Messages), len(ordered))
	}

	response, err := thmldsa44.Round3(
		p.keyShare,
		round2Messages,
		&state.round2,
		p.config.params,
	)
	if err != nil {
		p.makeTerminal(state, phaseFailed)
		return nil, fmt.Errorf("party %d round 3: %w", p.id, err)
	}

	p.makeTerminal(state, phaseCompleted)
	return response, nil
}

// Abort makes an attempt permanently unusable in this process. It is
// idempotent and also records an abort that arrives before Round1, preventing
// a delayed Round1 request from resurrecting the attempt.
func (p *localParty) Abort(session SigningSession) error {
	key, ordered, active, err := p.validateSession(session)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state, exists := p.sessions[key]
	if !exists {
		p.sessions[key] = terminalSession(ordered, active, phaseAborted)
		return nil
	}
	if !sameParticipants(state.participants, ordered) {
		return fmt.Errorf("%w: participant set changed", ErrSessionAlreadyUsed)
	}
	if state.phase == phaseCompleted || state.phase == phaseFailed {
		return nil
	}
	p.makeTerminal(state, phaseAborted)
	return nil
}

func (p *localParty) validateSession(session SigningSession) (sessionKey, []int, uint8, error) {
	if session.ID == "" {
		return sessionKey{}, nil, 0, fmt.Errorf("%w: empty session ID", ErrInvalidSigningSet)
	}
	if len(session.Participants) != p.config.threshold {
		return sessionKey{}, nil, 0, fmt.Errorf(
			"%w: need exactly %d participants, got %d",
			ErrInvalidSigningSet,
			p.config.threshold,
			len(session.Participants),
		)
	}

	ordered := session.OrderedParticipants()
	var active uint8
	containsSelf := false
	for i, participant := range ordered {
		if participant < 0 || participant >= p.config.parties {
			return sessionKey{}, nil, 0, fmt.Errorf(
				"%w: participant %d is outside [0,%d)",
				ErrInvalidSigningSet,
				participant,
				p.config.parties,
			)
		}
		if i > 0 && participant == ordered[i-1] {
			return sessionKey{}, nil, 0, fmt.Errorf("%w: duplicate participant %d", ErrInvalidSigningSet, participant)
		}
		containsSelf = containsSelf || participant == p.id
		active |= uint8(1 << uint(participant))
	}
	if !containsSelf {
		return sessionKey{}, nil, 0, fmt.Errorf("%w: party %d is not a participant", ErrInvalidSigningSet, p.id)
	}

	return sessionKey{id: session.ID, attempt: session.Attempt}, ordered, active, nil
}

func (p *localParty) stateForRound(
	key sessionKey,
	participants []int,
	want sessionPhase,
) (*partySession, error) {
	state, exists := p.sessions[key]
	if !exists {
		return nil, fmt.Errorf("%w: session was not started", ErrWrongRound)
	}
	if !sameParticipants(state.participants, participants) {
		return nil, fmt.Errorf("%w: participant set changed", ErrSessionAlreadyUsed)
	}
	if state.phase == phaseCompleted || state.phase == phaseFailed || state.phase == phaseAborted {
		return nil, ErrSessionAlreadyUsed
	}
	if state.phase != want {
		return nil, fmt.Errorf("%w: party %d is in phase %d, want %d", ErrWrongRound, p.id, state.phase, want)
	}
	return state, nil
}

func (p *localParty) makeTerminal(state *partySession, phase sessionPhase) {
	state.phase = phase
	state.round1 = thmldsa44.StRound1{}
	state.round2 = thmldsa44.StRound2{}
}

func terminalSession(participants []int, active uint8, phase sessionPhase) *partySession {
	return &partySession{
		participants: append([]int(nil), participants...),
		active:       active,
		phase:        phase,
	}
}

func sameParticipants(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Coordinator owns only public material and combines the public protocol
// messages returned by the Parties.
type Coordinator struct {
	config *PublicConfig
}

func NewCoordinator(config *PublicConfig) (*Coordinator, error) {
	if config == nil || config.publicKey == nil || config.params == nil {
		return nil, errors.New("cryptoengine: invalid public configuration")
	}
	return &Coordinator{config: config}, nil
}

// CombineAndVerify combines one threshold attempt and verifies the resulting
// signature before returning it. Rejection requires a new attempt on every
// Party; this method never asks a Party to reuse old round state.
func (c *Coordinator) CombineAndVerify(
	message []byte,
	round2Messages [][]byte,
	round3Messages [][]byte,
) ([]byte, error) {
	if c == nil || c.config == nil {
		return nil, errors.New("cryptoengine: nil coordinator")
	}
	if len(round2Messages) != c.config.threshold || len(round3Messages) != c.config.threshold {
		return nil, fmt.Errorf(
			"%w: need %d round messages, got %d commitments and %d responses",
			ErrInvalidSigningSet,
			c.config.threshold,
			len(round2Messages),
			len(round3Messages),
		)
	}

	signature := make([]byte, thmldsa44.SignatureSize)
	if !thmldsa44.Combine(
		c.config.publicKey,
		message,
		nil,
		round2Messages,
		round3Messages,
		signature,
		c.config.params,
	) {
		return nil, ErrCombineRejected
	}
	if !c.VerifyMessage(message, signature) {
		return nil, errors.New("cryptoengine: combined signature failed verification")
	}
	return signature, nil
}

// VerifyMessage verifies a signature using public material only.
func (c *Coordinator) VerifyMessage(message, signature []byte) bool {
	if c == nil || c.config == nil || len(signature) != thmldsa44.SignatureSize {
		return false
	}
	return thmldsa44.Verify(c.config.publicKey, message, nil, signature)
}
