// Package store provides the first-version in-memory transaction repository.
package store

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"threshold-mldsa/backend/clearing"
	"threshold-mldsa/network"
)

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusSigning  Status = "SIGNING"
	StatusVerified Status = "VERIFIED"
	StatusFailed   Status = "FAILED"
)

var (
	ErrNotFound                 = errors.New("transaction not found")
	ErrAlreadyExists            = errors.New("transaction already exists")
	ErrSigning                  = errors.New("transaction signing is already in progress")
	ErrAlreadySigned            = errors.New("transaction is already signed")
	ErrInvalidState             = errors.New("invalid transaction state")
	ErrReinitializationRequired = errors.New("transaction requires cluster reinitialization before retry")
)

// Record contains a transaction and the outcome of its threshold signature.
// Signature and Participants are copied at the Store boundary.
type Record struct {
	Transaction  clearing.Transaction
	Participants []int
	Status       Status
	Verified     bool
	Signature    []byte
	Duration     time.Duration
	Error        string
	Metrics      SigningMetrics
}

type SigningMetrics struct {
	Traffic                  *network.Traffic
	KeyID                    string
	Mode                     string
	SessionID                string
	Attempts                 int
	SigningDurationMS        float64
	VerificationDurationMS   float64
	RoundMessages            [3]int
	RoundMessagesKnown       bool
	AbortConfirmed           *bool
	RequiresReinitialization bool
}

type Store struct {
	mu      sync.RWMutex
	records map[string]Record
}

func New() *Store {
	return &Store{records: make(map[string]Record)}
}

func (s *Store) Create(tx clearing.Transaction) (Record, error) {
	if _, err := clearing.SigningMessage(tx); err != nil {
		return Record{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.records[tx.TransactionID]; exists {
		return Record{}, ErrAlreadyExists
	}
	record := Record{Transaction: tx, Status: StatusPending}
	s.records[tx.TransactionID] = record
	return cloneRecord(record), nil
}

func (s *Store) Get(transactionID string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.records[transactionID]
	if !exists {
		return Record{}, ErrNotFound
	}
	return cloneRecord(record), nil
}

// BeginSigning atomically prevents two HTTP requests from signing the same
// transaction concurrently. A failed transaction may be retried only if it
// does not require reinitialization of the controlled cluster.
func (s *Store) BeginSigning(transactionID string, participants []int, keyID, mode string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.records[transactionID]
	if !exists {
		return Record{}, ErrNotFound
	}
	switch record.Status {
	case StatusSigning:
		return Record{}, ErrSigning
	case StatusVerified:
		return Record{}, ErrAlreadySigned
	case StatusFailed:
		if record.Metrics.RequiresReinitialization {
			return Record{}, ErrReinitializationRequired
		}
	case StatusPending:
		// Allowed.
	default:
		return Record{}, fmt.Errorf("%w: %s", ErrInvalidState, record.Status)
	}

	record.Status = StatusSigning
	record.Error = ""
	record.Participants = append([]int(nil), participants...)
	record.Signature = nil
	record.Verified = false
	record.Duration = 0
	record.Metrics = SigningMetrics{KeyID: keyID, Mode: mode}
	s.records[transactionID] = record
	return cloneRecord(record), nil
}

func (s *Store) CompleteSigning(
	transactionID string,
	signature []byte,
	verified bool,
	duration time.Duration,
	metrics SigningMetrics,
) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.records[transactionID]
	if !exists {
		return Record{}, ErrNotFound
	}
	if record.Status != StatusSigning {
		return Record{}, fmt.Errorf("%w: cannot complete from %s", ErrInvalidState, record.Status)
	}

	record.Signature = append([]byte(nil), signature...)
	record.Verified = verified
	record.Duration = duration
	record.Metrics = cloneMetrics(metrics)
	if verified {
		record.Status = StatusVerified
	} else {
		record.Status = StatusFailed
		record.Signature = nil
		record.Error = "signature verification failed"
	}
	s.records[transactionID] = record
	return cloneRecord(record), nil
}

func (s *Store) FailSigning(transactionID string, cause error, duration time.Duration, metrics SigningMetrics) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.records[transactionID]
	if !exists {
		return Record{}, ErrNotFound
	}
	if record.Status != StatusSigning {
		return Record{}, fmt.Errorf("%w: cannot fail from %s", ErrInvalidState, record.Status)
	}

	record.Status = StatusFailed
	record.Verified = false
	record.Duration = duration
	record.Signature = nil
	record.Metrics = cloneMetrics(metrics)
	if cause != nil {
		record.Error = cause.Error()
	}
	s.records[transactionID] = record
	return cloneRecord(record), nil
}

func cloneRecord(record Record) Record {
	record.Participants = append([]int(nil), record.Participants...)
	record.Signature = append([]byte(nil), record.Signature...)
	record.Metrics = cloneMetrics(record.Metrics)
	return record
}

func cloneMetrics(metrics SigningMetrics) SigningMetrics {
	if metrics.Traffic != nil {
		value := *metrics.Traffic
		metrics.Traffic = &value
	}
	if metrics.AbortConfirmed != nil {
		value := *metrics.AbortConfirmed
		metrics.AbortConfirmed = &value
	}
	return metrics
}
