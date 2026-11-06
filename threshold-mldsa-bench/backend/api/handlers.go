// Package api exposes the simulated clearing service over net/http.
package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"threshold-mldsa/backend/clearing"
	transactionstore "threshold-mldsa/backend/store"
	"threshold-mldsa/cryptoengine"
)

var signingParticipants = []int{0, 2, 4}

type Handler struct {
	store *transactionstore.Store
}

func NewHandler(store *transactionstore.Store) http.Handler {
	handler := &Handler{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/transactions", handler.createTransaction)
	mux.HandleFunc("POST /api/transactions/{id}/sign", handler.signTransaction)
	mux.HandleFunc("GET /api/transactions/{id}", handler.getTransaction)
	return mux
}

type createResponse struct {
	TransactionID string                  `json:"transaction_id"`
	Status        transactionstore.Status `json:"status"`
}

type signatureResponse struct {
	TransactionID   string                  `json:"transaction_id"`
	Participants    []int                   `json:"participants"`
	Status          transactionstore.Status `json:"status"`
	Verified        bool                    `json:"verified"`
	SignatureBase64 string                  `json:"signature_base64"`
	DurationMS      float64                 `json:"duration_ms"`
}

type transactionResponse struct {
	Transaction     clearing.Transaction    `json:"transaction"`
	TransactionID   string                  `json:"transaction_id"`
	Participants    []int                   `json:"participants"`
	Status          transactionstore.Status `json:"status"`
	Verified        bool                    `json:"verified"`
	SignatureBase64 string                  `json:"signature_base64"`
	DurationMS      float64                 `json:"duration_ms"`
	Error           string                  `json:"error,omitempty"`
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
	record, err := h.store.BeginSigning(transactionID)
	if err != nil {
		switch {
		case errors.Is(err, transactionstore.ErrNotFound):
			writeError(w, http.StatusNotFound, err)
		case errors.Is(err, transactionstore.ErrSigning), errors.Is(err, transactionstore.ErrAlreadySigned):
			writeError(w, http.StatusConflict, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	message, err := clearing.SigningMessage(record.Transaction)
	if err != nil {
		_, _ = h.store.FailSigning(transactionID, err, 0)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	start := time.Now()
	signature, err := cryptoengine.SignMessage(message, signingParticipants)
	duration := time.Since(start)
	if err != nil {
		_, _ = h.store.FailSigning(transactionID, err, duration)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	verified := cryptoengine.VerifyMessage(message, signature)
	record, err = h.store.CompleteSigning(
		transactionID,
		signingParticipants,
		signature,
		verified,
		duration,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !verified {
		writeError(w, http.StatusInternalServerError, errors.New("threshold signature verification failed"))
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
		Transaction:     record.Transaction,
		TransactionID:   record.Transaction.TransactionID,
		Participants:    append([]int(nil), record.Participants...),
		Status:          record.Status,
		Verified:        record.Verified,
		SignatureBase64: base64.StdEncoding.EncodeToString(record.Signature),
		DurationMS:      durationMilliseconds(record.Duration),
		Error:           record.Error,
	})
}

func signatureResult(record transactionstore.Record) signatureResponse {
	return signatureResponse{
		TransactionID:   record.Transaction.TransactionID,
		Participants:    append([]int(nil), record.Participants...),
		Status:          record.Status,
		Verified:        record.Verified,
		SignatureBase64: base64.StdEncoding.EncodeToString(record.Signature),
		DurationMS:      durationMilliseconds(record.Duration),
	}
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
