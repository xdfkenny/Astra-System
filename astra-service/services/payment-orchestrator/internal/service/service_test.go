package service

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentv1 "github.com/astra-systems/astra-service/proto/gen/go/payment"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPayment_InitiatePayment_Approved(t *testing.T) {
	repo := newMockStore()
	locker := newMockLocker()
	vf := &mockVerifone{authorizeResp: &client.AuthorizeResponse{Status: "APPROVED", VerifoneToken: "tok-1", AuthCode: "auth-1"}}
	svc := NewPayment(repo, locker, vf)

	resp, err := svc.InitiatePayment(context.Background(), &paymentv1.PaymentIntent{
		PaymentId:      uuid.Must(uuid.NewV7()).String(),
		OrderId:        uuid.Must(uuid.NewV7()).String(),
		KioskId:        uuid.Must(uuid.NewV7()).String(),
		IdempotencyKey: uuid.Must(uuid.NewV7()).String(),
		AmountCents:    1000,
		Currency:       "USD",
		Method:         paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_DEBIT,
	})
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if resp.Status != paymentv1.PaymentStatus_PAYMENT_STATUS_AUTHORIZED {
		t.Fatalf("expected AUTHORIZED, got %v", resp.Status)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected 1 created payment, got %d", len(repo.created))
	}
	if repo.created[0].Status != domain.StatusAuthorizing {
		t.Fatalf("expected domain status AUTHORIZING, got %s", repo.created[0].Status)
	}
}

func TestPayment_InitiatePayment_Declined(t *testing.T) {
	repo := newMockStore()
	locker := newMockLocker()
	vf := &mockVerifone{authorizeResp: &client.AuthorizeResponse{Status: "DECLINED", DeclineReason: "insufficient_funds"}}
	svc := NewPayment(repo, locker, vf)

	resp, err := svc.InitiatePayment(context.Background(), &paymentv1.PaymentIntent{
		PaymentId:      uuid.Must(uuid.NewV7()).String(),
		OrderId:        uuid.Must(uuid.NewV7()).String(),
		KioskId:        uuid.Must(uuid.NewV7()).String(),
		IdempotencyKey: uuid.Must(uuid.NewV7()).String(),
		AmountCents:    1000,
		Currency:       "USD",
		Method:         paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_DEBIT,
	})
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}
	if resp.Status != paymentv1.PaymentStatus_PAYMENT_STATUS_DECLINED {
		t.Fatalf("expected DECLINED, got %v", resp.Status)
	}
}

func TestPayment_InitiatePayment_DuplicateIdempotencyKey(t *testing.T) {
	repo := newMockStore()
	locker := newMockLocker()
	vf := &mockVerifone{authorizeResp: &client.AuthorizeResponse{Status: "APPROVED", VerifoneToken: "tok-1"}}
	svc := NewPayment(repo, locker, vf)

	idemKey := uuid.Must(uuid.NewV7()).String()
	req := &paymentv1.PaymentIntent{
		PaymentId:      uuid.Must(uuid.NewV7()).String(),
		OrderId:        uuid.Must(uuid.NewV7()).String(),
		KioskId:        uuid.Must(uuid.NewV7()).String(),
		IdempotencyKey: idemKey,
		AmountCents:    1000,
		Currency:       "USD",
		Method:         paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_DEBIT,
	}
	if _, err := svc.InitiatePayment(context.Background(), req); err != nil {
		t.Fatalf("first initiate: %v", err)
	}
	repo.createErr = repository.ErrDuplicateIdempotencyKey
	_, err := svc.InitiatePayment(context.Background(), req)
	if err == nil {
		t.Fatal("expected duplicate idempotency key error")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", st.Code())
	}
}

func TestPayment_CapturePayment_WrongStatus(t *testing.T) {
	repo := newMockStore()
	paymentID := uuid.Must(uuid.NewV7())
	repo.payments[paymentID] = &domain.Payment{PaymentID: paymentID, Status: domain.StatusPending}
	svc := NewPayment(repo, newMockLocker(), &mockVerifone{})

	_, err := svc.CapturePayment(context.Background(), &paymentv1.PaymentResult{PaymentId: paymentID.String()})
	if err == nil {
		t.Fatal("expected error")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestPayment_CapturePayment_Success(t *testing.T) {
	repo := newMockStore()
	paymentID := uuid.Must(uuid.NewV7())
	repo.payments[paymentID] = &domain.Payment{PaymentID: paymentID, Status: domain.StatusAuthorizing, VerifoneToken: "tok-1"}
	svc := NewPayment(repo, newMockLocker(), &mockVerifone{})

	resp, err := svc.CapturePayment(context.Background(), &paymentv1.PaymentResult{PaymentId: paymentID.String()})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if resp.Status != paymentv1.PaymentStatus_PAYMENT_STATUS_CAPTURED {
		t.Fatalf("expected CAPTURED, got %v", resp.Status)
	}
}

func TestPayment_LockPreventsConcurrentIdempotencyKey(t *testing.T) {
	repo := newMockStore()
	locker := newMockLocker()
	locker.locked = false
	vf := &mockVerifone{authorizeResp: &client.AuthorizeResponse{Status: "APPROVED"}}
	svc := NewPayment(repo, locker, vf)

	_, err := svc.InitiatePayment(context.Background(), &paymentv1.PaymentIntent{
		PaymentId:      uuid.Must(uuid.NewV7()).String(),
		OrderId:        uuid.Must(uuid.NewV7()).String(),
		KioskId:        uuid.Must(uuid.NewV7()).String(),
		IdempotencyKey: uuid.Must(uuid.NewV7()).String(),
		AmountCents:    100,
		Currency:       "USD",
		Method:         paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_DEBIT,
	})
	if err == nil {
		t.Fatal("expected error when lock not acquired")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", st.Code())
	}
}

type mockStore struct {
	payments  map[uuid.UUID]*domain.Payment
	created   []*domain.Payment
	createErr error
}

func newMockStore() *mockStore {
	return &mockStore{payments: make(map[uuid.UUID]*domain.Payment)}
}

func (m *mockStore) Create(_ context.Context, p *domain.Payment) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.payments[p.PaymentID] = p
	m.created = append(m.created, p)
	return nil
}

func (m *mockStore) GetByID(_ context.Context, id uuid.UUID) (*domain.Payment, error) {
	return m.payments[id], nil
}

func (m *mockStore) UpdateStatus(_ context.Context, id uuid.UUID, from, to domain.PaymentStatus, token, reason string) error {
	p := m.payments[id]
	if p == nil {
		return errors.New("not found")
	}
	if p.Status != from {
		return domain.ErrInvalidTransition
	}
	p.Status = to
	p.VerifoneToken = token
	p.DeclineReason = reason
	p.UpdatedAt = time.Now().UTC()
	return nil
}

type mockLocker struct {
	locked bool
}

func newMockLocker() *mockLocker {
	return &mockLocker{locked: true}
}

func (m *mockLocker) Lock(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.locked, nil
}

func (m *mockLocker) Unlock(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockVerifone struct {
	authorizeResp *client.AuthorizeResponse
	authorizeErr  error
	captureErr    error
	settleErr     error
}

func (m *mockVerifone) Authorize(_ context.Context, _ *client.AuthorizeRequest) (*client.AuthorizeResponse, error) {
	if m.authorizeErr != nil {
		return nil, m.authorizeErr
	}
	return m.authorizeResp, nil
}

func (m *mockVerifone) Capture(_ context.Context, _, _ string) error { return m.captureErr }
func (m *mockVerifone) Settle(_ context.Context, _, _ string) error  { return m.settleErr }
