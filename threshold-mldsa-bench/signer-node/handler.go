// Package signernode hosts one Party and its process-local protocol sessions.
package signernode

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"sync"

	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
)

const maxSessionRecords = 4096 // Refuse new work rather than deleting replay tombstones.

type record struct {
	ctx       cryptoengine.WireContext
	state     string
	ownRound1 []byte
	ownRound2 []byte
}

type attemptKey struct {
	id      string
	attempt uint64
}

type Handler struct {
	mu       sync.Mutex
	config   *cryptoengine.PublicConfig
	party    cryptoengine.Party
	authHash [32]byte
	sessions map[attemptKey]*record
	latest   map[string]uint64
}

func NewHandler(config *cryptoengine.PublicConfig, party cryptoengine.Party, token string) (http.Handler, error) {
	if config == nil || party == nil || network.ValidateToken(token) != nil {
		return nil, errors.New("invalid node configuration")
	}
	h := &Handler{config: config, party: party, authHash: sha256.Sum256([]byte("Bearer " + token)),
		sessions: make(map[attemptKey]*record), latest: make(map[string]uint64)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, network.Health{PartyID: party.ID(), Status: "CRYPTO_READY", KeyLoaded: true,
			KeyID: config.KeyID(), Threshold: config.Threshold(), Total: config.TotalParties()})
	})
	mux.HandleFunc("GET /internal/config", h.authorized(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, config.PublicDescriptor())
	}))
	for _, operation := range []string{"round1", "round2", "round3", "abort", "status"} {
		mux.HandleFunc("POST /internal/"+operation, h.authorized(func(w http.ResponseWriter, r *http.Request) {
			h.handle(w, r, operation)
		}))
	}
	return mux, nil
}

func (h *Handler) authorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		actual := sha256.Sum256([]byte(r.Header.Get("Authorization")))
		if err != nil || ip == nil || !ip.IsLoopback() || subtle.ConstantTimeCompare(actual[:], h.authHash[:]) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized coordinator")
			return
		}
		next(w, r)
	}
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request, operation string) {
	var req network.RoundRequest
	r.Body = http.MaxBytesReader(w, r.Body, network.MaxHTTPBodySize)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(&req) != nil || d.Decode(new(any)) != io.EOF || req.WireContext.Validate(h.config) != nil ||
		!slices.Contains(req.Participants, h.party.ID()) {
		writeError(w, http.StatusBadRequest, "invalid protocol request")
		return
	}
	if operation != "round2" && len(req.Message) != 0 ||
		(operation == "round1" || operation == "status" || operation == "abort") && len(req.Payload) != 0 {
		writeError(w, http.StatusBadRequest, "unexpected protocol payload")
		return
	}
	if operation == "round2" && (len(req.Message) > cryptoengine.MaxSigningMessageSize || cryptoengine.MessageHash(req.Message) != req.MessageHash) {
		writeError(w, http.StatusBadRequest, "message hash does not match session")
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	key := attemptKey{req.SessionID, req.Attempt}
	state := h.sessions[key]
	if state != nil && !state.ctx.Equal(req.WireContext) {
		writeError(w, http.StatusConflict, "session binding changed")
		return
	}
	if latest, exists := h.latest[req.SessionID]; exists {
		previous := h.sessions[attemptKey{req.SessionID, latest}]
		if previous.ctx.MessageHash != req.MessageHash || !slices.Equal(previous.ctx.Participants, req.Participants) {
			writeError(w, http.StatusConflict, "business session binding changed")
			return
		}
		isRound := operation == "round1" || operation == "round2" || operation == "round3"
		if isRound && req.Attempt < latest || req.Attempt > latest && operation != "status" && !terminal(previous.state) {
			writeError(w, http.StatusConflict, "previous attempt is not terminal")
			return
		}
	}
	if operation == "status" {
		phase := "UNKNOWN"
		if state != nil {
			phase = state.state
		}
		writeJSON(w, http.StatusOK, network.StateResponse{WireContext: req.WireContext, PartyID: h.party.ID(), State: phase})
		return
	}
	if operation == "abort" {
		if state == nil {
			if len(h.sessions) >= maxSessionRecords {
				writeError(w, http.StatusServiceUnavailable, "session capacity reached")
				return
			}
			state = &record{ctx: cloneContext(req.WireContext)}
			h.sessions[key] = state
			if latest, exists := h.latest[req.SessionID]; !exists || latest < req.Attempt {
				h.latest[req.SessionID] = req.Attempt
			}
		}
		if err := h.party.Abort(req.WireContext.Session()); err != nil {
			writeError(w, http.StatusConflict, "abort rejected")
			return
		}
		state.state = "ABORTED"
		state.ownRound1, state.ownRound2 = nil, nil
		writeJSON(w, http.StatusOK, network.StateResponse{WireContext: req.WireContext, PartyID: h.party.ID(), State: state.state})
		return
	}
	if operation == "round1" {
		if state != nil {
			writeError(w, http.StatusConflict, "attempt already started or spent")
			return
		}
		if len(h.sessions) >= maxSessionRecords {
			writeError(w, http.StatusServiceUnavailable, "session capacity reached")
			return
		}
		state = &record{ctx: cloneContext(req.WireContext), state: "STARTING"}
		h.sessions[key] = state
		h.latest[req.SessionID] = req.Attempt
	} else if state == nil || terminal(state.state) {
		writeError(w, http.StatusConflict, "attempt missing or spent")
		return
	}

	var payload []byte
	var err error
	round := 0
	switch operation {
	case "round1":
		round = 1
		payload, err = h.party.Round1(req.WireContext.Session())
		if err == nil {
			state.state = "ROUND1_DONE"
			state.ownRound1 = append([]byte(nil), payload...)
		}
	case "round2":
		round = 2
		if state.state != "ROUND1_DONE" {
			writeError(w, http.StatusConflict, "wrong protocol round")
			return
		}
		var messages [][]byte
		messages, err = cryptoengine.DecodeRoundMessages(h.config, req.WireContext, 1, req.Payload)
		if err == nil && !bytes.Equal(messages[slices.Index(req.Participants, h.party.ID())], state.ownRound1) {
			err = cryptoengine.ErrInvalidWireMessage
		}
		if err == nil {
			payload, err = h.party.Round2(req.WireContext.Session(), req.Message, messages)
		}
		if err == nil {
			state.state = "ROUND2_DONE"
			state.ownRound2 = append([]byte(nil), payload...)
		}
	case "round3":
		round = 3
		if state.state != "ROUND2_DONE" {
			writeError(w, http.StatusConflict, "wrong protocol round")
			return
		}
		var messages [][]byte
		messages, err = cryptoengine.DecodeRoundMessages(h.config, req.WireContext, 2, req.Payload)
		if err == nil && !bytes.Equal(messages[slices.Index(req.Participants, h.party.ID())], state.ownRound2) {
			err = cryptoengine.ErrInvalidWireMessage
		}
		if err == nil {
			payload, err = h.party.Round3(req.WireContext.Session(), messages)
		}
		if err == nil {
			state.state = "COMPLETED"
			state.ownRound1, state.ownRound2 = nil, nil
		}
	}
	if err != nil {
		_ = h.party.Abort(req.WireContext.Session())
		state.state = "FAILED"
		state.ownRound1, state.ownRound2 = nil, nil
		writeError(w, http.StatusConflict, "protocol attempt failed and was consumed")
		return
	}
	writeJSON(w, http.StatusOK, network.RoundResponse{WireContext: state.ctx, PartyID: h.party.ID(), Round: round, Payload: payload})
}

func terminal(state string) bool {
	return state == "COMPLETED" || state == "FAILED" || state == "ABORTED"
}
func cloneContext(ctx cryptoengine.WireContext) cryptoengine.WireContext {
	ctx.Participants = append([]int(nil), ctx.Participants...)
	return ctx
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, network.ErrorResponse{Error: message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
