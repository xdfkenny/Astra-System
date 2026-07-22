// Package service implements the PaymentOrchestrator gRPC service and the core
// payment orchestration logic shared with the REST handler.
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/astra-systems/astra-service/proto/gen/go/payment"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store is the persistence surface required by the payment service.
type Store interface {
	Create(ctx context.Context, p *domain.Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, from, to domain.PaymentStatus, verifoneToken, declineReason string) error
}

// Locker is the idempotency surface required by the payment service.
type Locker interface {
	Lock(ctx context.Context, key uuid.UUID) (bool, error)
	Unlock(ctx context.Context, key uuid.UUID) error
}

// Payment implements the PaymentOrchestrator gRPC service.
type Payment struct {
	payment.UnimplementedPaymentOrchestratorServer
	repo      Store
	idemStore Locker
	verifone  client.Gateway
}

// NewPayment creates a payment orchestration service.
func NewPayment(repo Store, idemStore Locker, verifone client.Gateway) *Payment {
	return &Payment{
		repo:      repo,
		idemStore: idemStore,
		verifone:  verifone,
	}
}

// InitiatePayment starts a payment, transitions to AUTHORIZING on approval, and
// stores the payment with an outbox event.
func (s *Payment) InitiatePayment(ctx context.Context, req *payment.PaymentIntent) (*payment.PaymentResult, error) {
	if req.PaymentId == "" || req.OrderId == "" || req.KioskId == "" || req.AmountCents <= 0 || req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "missing required field")
	}

	paymentID, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment_id")
	}
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	kioskID, err := uuid.Parse(req.KioskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid kiosk_id")
	}
	idemKey, err := uuid.Parse(req.IdempotencyKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid idempotency_key")
	}

	locked, err := s.idemStore.Lock(ctx, idemKey)
	if err != nil {
		return nil, status.Error(codes.Internal, "idempotency lock error")
	}
	if !locked {
		return nil, status.Error(codes.AlreadyExists, "idempotency key in progress")
	}
	defer s.idemStore.Unlock(context.WithoutCancel(ctx), idemKey)

	p := &domain.Payment{
		PaymentID:      paymentID,
		OrderID:        orderID,
		KioskID:        kioskID,
		IdempotencyKey: idemKey,
		AmountCents:    int(req.AmountCents),
		Currency:       req.Currency,
		Method:         mapMethod(req.Method),
		Status:         domain.StatusPending,
		IsOfflineToken: req.IsOfflineToken,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	authResp, err := s.verifone.Authorize(ctx, &client.AuthorizeRequest{
		PaymentID:   p.PaymentID.String(),
		OrderID:     p.OrderID.String(),
		KioskID:     p.KioskID.String(),
		AmountCents: p.AmountCents,
		Currency:    p.Currency,
		Method:      string(p.Method),
	})
	if err != nil {
		_ = p.Transition(domain.StatusFailed)
		p.DeclineReason = err.Error()
		if createErr := s.repo.Create(ctx, p); createErr != nil {
			return nil, status.Error(codes.Internal, createErr.Error())
		}
		return s.domainToProto(p), nil
	}

	switch strings.ToUpper(authResp.Status) {
	case "APPROVED", "AUTHORIZED":
		p.VerifoneToken = authResp.VerifoneToken
		p.VerifoneAuthCode = authResp.AuthCode
		p.CardBrand = authResp.CardBrand
		p.CardLastFour = authResp.CardLastFour
		p.ReceiptText = authResp.ReceiptText
		if err := p.Transition(domain.StatusAuthorizing); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	case "DECLINED":
		p.DeclineReason = authResp.DeclineReason
		if err := p.Transition(domain.StatusFailed); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	default:
		p.DeclineReason = authResp.DeclineReason
		if err := p.Transition(domain.StatusFailed); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	if err := s.repo.Create(ctx, p); err != nil {
		if errors.Is(err, repository.ErrDuplicateIdempotencyKey) {
			return nil, status.Error(codes.AlreadyExists, "duplicate idempotency key")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.domainToProto(p), nil
}

// CapturePayment captures a payment that is currently AUTHORIZING.
func (s *Payment) CapturePayment(ctx context.Context, req *payment.PaymentResult) (*payment.PaymentResult, error) {
	id, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment_id")
	}

	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if p == nil {
		return nil, status.Error(codes.NotFound, "payment not found")
	}
	if p.Status != domain.StatusAuthorizing {
		return nil, status.Error(codes.FailedPrecondition, "payment not in authorizing state")
	}

	if err := s.verifone.Capture(ctx, p.PaymentID.String(), p.VerifoneToken); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	if err := s.repo.UpdateStatus(ctx, id, domain.StatusAuthorizing, domain.StatusCaptured, p.VerifoneToken, ""); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	p.Status = domain.StatusCaptured
	return s.domainToProto(p), nil
}

// RefundPayment refunds a previously captured or settled payment.
func (s *Payment) RefundPayment(ctx context.Context, req *payment.RefundRequest) (*payment.PaymentResult, error) {
	paymentID, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment_id")
	}

	p, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if p == nil {
		return nil, status.Error(codes.NotFound, "payment not found")
	}
	if p.Status != domain.StatusCaptured && p.Status != domain.StatusSettled {
		return nil, status.Error(codes.FailedPrecondition, "payment cannot be refunded")
	}

	// Refund is executed through the Verifone gateway as a reversal/settle request.
	if err := s.verifone.Settle(ctx, p.PaymentID.String(), p.VerifoneToken); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	if err := s.repo.UpdateStatus(ctx, paymentID, p.Status, domain.StatusFailed, p.VerifoneToken, "refunded"); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	p.Status = domain.StatusFailed
	return s.domainToProto(p), nil
}

// SettleOfflineToken verifies and settles an offline token batch containing one token.
func (s *Payment) SettleOfflineToken(ctx context.Context, req *payment.OfflineTokenSettlement) (*payment.PaymentResult, error) {
	return nil, status.Error(codes.Unimplemented, "use REST /v1/offline-tokens/settle for batch settlement")
}

// GetPaymentStatus returns the current status of a payment.
func (s *Payment) GetPaymentStatus(ctx context.Context, req *payment.GetPaymentStatusRequest) (*payment.PaymentResult, error) {
	id, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment_id")
	}
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if p == nil {
		return nil, status.Error(codes.NotFound, "payment not found")
	}
	return s.domainToProto(p), nil
}

// SettlePayment settles a previously captured payment.
func (s *Payment) SettlePayment(ctx context.Context, paymentID uuid.UUID) (*payment.PaymentResult, error) {
	p, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if p == nil {
		return nil, status.Error(codes.NotFound, "payment not found")
	}
	if p.Status != domain.StatusCaptured {
		return nil, status.Error(codes.FailedPrecondition, "payment not in captured state")
	}

	if err := s.verifone.Settle(ctx, p.PaymentID.String(), p.VerifoneToken); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	if err := s.repo.UpdateStatus(ctx, paymentID, domain.StatusCaptured, domain.StatusSettled, p.VerifoneToken, ""); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	p.Status = domain.StatusSettled
	return s.domainToProto(p), nil
}

// UpdateStatus applies a webhook-driven status transition.
func (s *Payment) UpdateStatus(ctx context.Context, paymentID uuid.UUID, to domain.PaymentStatus, declineReason string) (*payment.PaymentResult, error) {
	p, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if p == nil {
		return nil, status.Error(codes.NotFound, "payment not found")
	}
	if p.IsTerminal() {
		return s.domainToProto(p), nil
	}
	if err := s.repo.UpdateStatus(ctx, paymentID, p.Status, to, p.VerifoneToken, declineReason); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	p.Status = to
	p.DeclineReason = declineReason
	return s.domainToProto(p), nil
}

func (s *Payment) domainToProto(p *domain.Payment) *payment.PaymentResult {
	return &payment.PaymentResult{
		PaymentId:     p.PaymentID.String(),
		OrderId:       p.OrderID.String(),
		Status:        mapStatus(p.Status),
		AuthCode:      p.VerifoneAuthCode,
		VerifoneToken: p.VerifoneToken,
		CardBrand:     p.CardBrand,
		CardLastFour:  p.CardLastFour,
		DeclineReason: p.DeclineReason,
		ReceiptText:   p.ReceiptText,
		ProcessedAt:   p.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func mapMethod(m payment.PaymentMethod) domain.PaymentMethod {
	switch m {
	case payment.PaymentMethod_PAYMENT_METHOD_CREDIT_DEBIT:
		return domain.MethodCreditDebit
	case payment.PaymentMethod_PAYMENT_METHOD_NFC_APPLE_PAY:
		return domain.MethodNFCApplePay
	case payment.PaymentMethod_PAYMENT_METHOD_NFC_GOOGLE_PAY:
		return domain.MethodNFCGooglePay
	case payment.PaymentMethod_PAYMENT_METHOD_QR_CODE:
		return domain.MethodQRCode
	case payment.PaymentMethod_PAYMENT_METHOD_CASH_RECYCLER:
		return domain.MethodCashRecycler
	default:
		return ""
	}
}

func mapStatus(s domain.PaymentStatus) payment.PaymentStatus {
	switch s {
	case domain.StatusPending:
		return payment.PaymentStatus_PAYMENT_STATUS_PENDING
	case domain.StatusAuthorizing:
		return payment.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
	case domain.StatusCaptured:
		return payment.PaymentStatus_PAYMENT_STATUS_CAPTURED
	case domain.StatusSettled:
		return payment.PaymentStatus_PAYMENT_STATUS_CAPTURED
	case domain.StatusFailed:
		return payment.PaymentStatus_PAYMENT_STATUS_DECLINED
	default:
		return payment.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}
