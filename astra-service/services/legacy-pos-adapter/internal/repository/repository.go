// Package repository persists the outcome of legacy POS submissions.
package repository

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrSubmissionNotFound is returned when a submission record cannot be located.
var ErrSubmissionNotFound = errors.New("submission not found")

// Submission records the result of forwarding a completed cart/order to the
// legacy POS system.
type Submission struct {
	SubmissionID   string
	OrderID        string
	CartID         string
	StoreID        string
	KioskID        string
	LegacyPOSURL   string
	RequestPayload []byte
	ResponseBody   []byte
	StatusCode     int
	Error          string
	SentAt         time.Time
	CreatedAt      time.Time
}

// Repository defines persistence for legacy POS submissions.
type Repository interface {
	SaveSubmission(ctx context.Context, s *Submission) error
	GetSubmission(ctx context.Context, submissionID string) (*Submission, error)
	ListSubmissionsByOrder(ctx context.Context, orderID string) ([]*Submission, error)
}

// MemoryRepository is an in-memory implementation used for local development
// and unit tests.
type MemoryRepository struct {
	mu   sync.RWMutex
	byID map[string]*Submission
	byOrder map[string][]*Submission
}

// NewMemoryRepository returns an empty in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:    make(map[string]*Submission),
		byOrder: make(map[string][]*Submission),
	}
}

// SaveSubmission stores a submission record.
func (r *MemoryRepository) SaveSubmission(_ context.Context, s *Submission) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s.CreatedAt = time.Now().UTC()
	r.byID[s.SubmissionID] = s
	if s.OrderID != "" {
		r.byOrder[s.OrderID] = append(r.byOrder[s.OrderID], s)
	}
	return nil
}

// GetSubmission returns a submission by ID.
func (r *MemoryRepository) GetSubmission(_ context.Context, submissionID string) (*Submission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.byID[submissionID]
	if !ok {
		return nil, ErrSubmissionNotFound
	}
	return s, nil
}

// ListSubmissionsByOrder returns all submissions for an order.
func (r *MemoryRepository) ListSubmissionsByOrder(_ context.Context, orderID string) ([]*Submission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]*Submission(nil), r.byOrder[orderID]...), nil
}

// compile-time interface assertion.
var _ Repository = (*MemoryRepository)(nil)
