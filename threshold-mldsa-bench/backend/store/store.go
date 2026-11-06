// Package store provides the first-version in-memory transaction repository.
package store

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"threshold-mldsa/backend/clearing"
)

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusSigning  Status = "SIGNING"
	StatusVerified Status = "VERIFIED"
	StatusFailed   Status = "FAILED"
)

var (
	ErrNotFound      = errors.New("transaction not found")
	ErrAlreadyExists = errors.New("transaction already exists")
	ErrSigning       = errors.New("transaction signing is already in progress")
	ErrAlreadySigned = errors.New("transaction is already signed")
	ErrInvalidState  = errors.New("invalid transaction state")
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
// transaction concurrently. A failed transaction may be retried.
func (s *Store) BeginSigning(transactionID string) (Record, error) {
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
	case StatusPending, StatusFailed:
		// Allowed.
	default:
		return Record{}, fmt.Errorf("%w: %s", ErrInvalidState, record.Status)
	}

	record.Status = StatusSigning
	record.Error = ""
	s.records[transactionID] = record
	return cloneRecord(record), nil
}

func (s *Store) CompleteSigning(
	transactionID string,
	participants []int,
	signature []byte,
	verified bool,
	duration time.Duration,
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

	record.Participants = append([]int(nil), participants...)
	record.Signature = append([]byte(nil), signature...)
	record.Verified = verified
	record.Duration = duration
	if verified {
		record.Status = StatusVerified
	} else {
		record.Status = StatusFailed
		record.Error = "signature verification failed"
	}
	s.records[transactionID] = record
	return cloneRecord(record), nil
}

func (s *Store) FailSigning(transactionID string, cause error, duration time.Duration) (Record, error) {
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
	if cause != nil {
		record.Error = cause.Error()
	}
	s.records[transactionID] = record
	return cloneRecord(record), nil
}

func cloneRecord(record Record) Record {
	record.Participants = append([]int(nil), record.Participants...)
	record.Signature = append([]byte(nil), record.Signature...)
	return record
}
