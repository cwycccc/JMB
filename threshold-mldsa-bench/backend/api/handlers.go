// Package api exposes the simulated clearing service over net/http.
package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"threshold-mldsa/backend/clearing"
	"threshold-mldsa/backend/signer"
	transactionstore "threshold-mldsa/backend/store"
	"threshold-mldsa/backend/web"
	"threshold-mldsa/network"
)

var signingParticipants = []int{0, 2, 4}

type Handler struct {
	store    *transactionstore.Store
	signer   signer.Service
	verifier signer.Verifier
	options  Options
}

type Options struct {
	Mode        string
	SignTimeout time.Duration
}

func NewHandler(store *transactionstore.Store, service signer.Service, verifier signer.Verifier, options Options) (http.Handler, error) {
	if store == nil || service == nil || verifier == nil || (options.Mode != "local" && options.Mode != "distributed") {
		return nil, errors.New("invalid signing handler configuration")
	}
	if options.SignTimeout == 0 {
		options.SignTimeout = 20 * time.Second
	}
	if options.SignTimeout < 0 {
		return nil, errors.New("invalid signing timeout")
	}
	handler := &Handler{store: store, signer: service, verifier: verifier, options: options}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/transactions", handler.createTransaction)
	mux.HandleFunc("POST /api/transactions/{id}/sign", handler.signTransaction)
	mux.HandleFunc("GET /api/transactions/{id}", handler.getTransaction)
	mux.HandleFunc("POST /api/transactions/{id}/verify", handler.verifyTransaction)
	mux.HandleFunc("GET /api/cluster", handler.clusterStatus)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { writeError(w, 404, errors.New("API not found")) })
	mux.Handle("/", web.Handler())
	return mux, nil
}

type createResponse struct {
	TransactionID string                  `json:"transaction_id"`
	Status        transactionstore.Status `json:"status"`
}

type signatureResponse struct {
	Traffic                  *network.Traffic        `json:"traffic,omitempty"`
	TransactionID            string                  `json:"transaction_id"`
	Participants             []int                   `json:"participants"`
	Status                   transactionstore.Status `json:"status"`
	Verified                 bool                    `json:"verified"`
	SignatureBase64          string                  `json:"signature_base64"`
	DurationMS               float64                 `json:"duration_ms"`
	KeyID                    string                  `json:"key_id"`
	SigningMode              string                  `json:"signing_mode"`
	SessionID                string                  `json:"session_id,omitempty"`
	Attempts                 int                     `json:"attempts"`
	SigningDurationMS        float64                 `json:"signing_duration_ms"`
	VerificationDurationMS   float64                 `json:"verification_duration_ms"`
	RoundMessages            *[3]int                 `json:"round_message_counts,omitempty"`
	AbortConfirmed           *bool                   `json:"abort_confirmed,omitempty"`
	RequiresReinitialization bool                    `json:"requires_reinitialization,omitempty"`
}

type transactionResponse struct {
	Transaction clearing.Transaction `json:"transaction"`
	signatureResponse
	Error string `json:"error,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var tx clearing.Transaction
	if err := decodeJSON(w, r, &tx); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	record, err := h.store.Create(tx)
	if err != nil {
		if errors.Is(err, transactionstore.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, createResponse{
		TransactionID: record.Transaction.TransactionID,
		Status:        record.Status,
	})
}

func (h *Handler) signTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID := r.PathValue("id")
	participants := append([]int(nil), signingParticipants...)
	if r.Body != nil && r.Body != http.NoBody {
		// Empty bodies keep the original API behavior; explicit JSON selects a set.
		var request struct {
			Participants []int `json:"participants"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if err := d.Decode(&request); err != io.EOF {
			if err != nil || d.Decode(new(any)) != io.EOF {
				writeError(w, 400, errors.New("invalid participant request"))
				return
			}
			participants = request.Participants
		}
	}
	slices.Sort(participants)
	if len(participants) != 3 {
		writeError(w, 400, errors.New("select exactly 3 participants for 3-of-5 signing"))
		return
	}
	for i, id := range participants {
		if id < 0 || id >= 5 || i > 0 && participants[i-1] == id {
			writeError(w, 400, errors.New("invalid or duplicate participant"))
			return
		}
	}
	record, err := h.store.BeginSigning(transactionID, participants, h.verifier.KeyID(), h.options.Mode)
	if err != nil {
		switch {
		case errors.Is(err, transactionstore.ErrNotFound):
			writeError(w, http.StatusNotFound, err)
		case errors.Is(err, transactionstore.ErrSigning), errors.Is(err, transactionstore.ErrAlreadySigned), errors.Is(err, transactionstore.ErrReinitializationRequired):
			writeError(w, http.StatusConflict, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	message, err := clearing.SigningMessage(record.Transaction)
	if err != nil {
		_, _ = h.store.FailSigning(transactionID, err, 0, record.Metrics)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), h.options.SignTimeout)
	defer cancel()
	result, err := h.signer.Sign(ctx, append([]byte(nil), message...), append([]int(nil), participants...))
	metrics := record.Metrics
	metrics.SessionID = result.SessionID
	metrics.Attempts = result.Attempts
	metrics.SigningDurationMS = result.DurationMs
	metrics.RoundMessages = result.RoundMessages
	metrics.RoundMessagesKnown = result.RoundMessagesKnown
	metrics.Traffic = result.Traffic
	if err != nil {
		var failure *signer.Failure
		if errors.As(err, &failure) {
			metrics.AbortConfirmed = failure.AbortConfirmed
			metrics.RequiresReinitialization = failure.RequiresReinitialization
		}
		record, storeErr := h.store.FailSigning(transactionID, err, time.Since(start), metrics)
		if storeErr != nil {
			writeError(w, http.StatusInternalServerError, storeErr)
			return
		}
		status := http.StatusBadGateway
		if signer.IsTimeout(err) {
			status = http.StatusGatewayTimeout
		} else if errors.Is(err, signer.ErrUnavailable) || errors.Is(err, signer.ErrReinitializationRequired) {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, transactionResponse{Transaction: record.Transaction, signatureResponse: signatureResult(record), Error: record.Error})
		return
	}

	verifyStart := time.Now()
	verified := h.verifier.Verify(result.KeyID, message, result.Signature)
	metrics.VerificationDurationMS = durationMilliseconds(time.Since(verifyStart))
	if ctx.Err() != nil {
		record, storeErr := h.store.FailSigning(transactionID, ctx.Err(), time.Since(start), metrics)
		if storeErr != nil {
			writeError(w, http.StatusInternalServerError, storeErr)
			return
		}
		writeJSON(w, http.StatusGatewayTimeout, transactionResponse{Transaction: record.Transaction, signatureResponse: signatureResult(record), Error: record.Error})
		return
	}
	record, err = h.store.CompleteSigning(
		transactionID,
		result.Signature,
		verified,
		time.Since(start),
		metrics,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !verified {
		writeJSON(w, http.StatusBadGateway, transactionResponse{Transaction: record.Transaction, signatureResponse: signatureResult(record), Error: record.Error})
		return
	}

	writeJSON(w, http.StatusOK, signatureResult(record))
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	record, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, transactionstore.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, transactionResponse{
		Transaction:       record.Transaction,
		signatureResponse: signatureResult(record),
		Error:             record.Error,
	})
}

func signatureResult(record transactionstore.Record) signatureResponse {
	var roundMessages *[3]int
	if record.Metrics.RoundMessagesKnown {
		value := record.Metrics.RoundMessages
		roundMessages = &value
	}
	return signatureResponse{
		Traffic:         record.Metrics.Traffic,
		TransactionID:   record.Transaction.TransactionID,
		Participants:    append([]int(nil), record.Participants...),
		Status:          record.Status,
		Verified:        record.Verified,
		SignatureBase64: base64.StdEncoding.EncodeToString(record.Signature),
		DurationMS:      durationMilliseconds(record.Duration),
		KeyID:           record.Metrics.KeyID, SigningMode: record.Metrics.Mode, SessionID: record.Metrics.SessionID,
		Attempts: record.Metrics.Attempts, SigningDurationMS: record.Metrics.SigningDurationMS,
		VerificationDurationMS: record.Metrics.VerificationDurationMS, RoundMessages: roundMessages,
		AbortConfirmed: record.Metrics.AbortConfirmed, RequiresReinitialization: record.Metrics.RequiresReinitialization,
	}
}

func (h *Handler) clusterStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if monitor, ok := h.signer.(signer.Monitor); ok {
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		writeJSON(w, 200, monitor.ClusterStatus(ctx))
		return
	}
	// Local mode must never pretend to monitor five independent processes.
	writeJSON(w, 200, signer.ClusterStatus{Mode: h.options.Mode, KeyID: h.verifier.KeyID(), Threshold: 3, Total: 5, Nodes: []signer.NodeStatus{}})
}

// Verify a candidate copy against the stored signature; never mutate the
// signed transaction or accept a caller-supplied public key/signature.
func (h *Handler) verifyTransaction(w http.ResponseWriter, r *http.Request) {
	record, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		writeError(w, 404, err)
		return
	}
	if record.Status != transactionstore.StatusVerified {
		writeError(w, 409, errors.New("transaction has no verified signature"))
		return
	}
	var tx clearing.Transaction
	if err := decodeJSON(w, r, &tx); err != nil {
		writeError(w, 400, err)
		return
	}
	message, err := clearing.SigningMessage(tx)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"verified": h.verifier.Verify(record.Metrics.KeyID, message, record.Signature), "key_id": record.Metrics.KeyID, "transaction_id": record.Transaction.TransactionID, "stored_transaction_unchanged": true})
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration.Nanoseconds()) / float64(time.Millisecond)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResponse{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
