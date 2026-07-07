// Package domain contains the payment aggregate and state-machine rules.
package domain

import (
	"errors"
	"time"

	paymentv1 "github.com/astra-systems/astra-service/proto/gen/go/payment"
	"github.com/google/uuid"
)

// PaymentStatus models the payment lifecycle.
type PaymentStatus string

const (
	StatusPending     PaymentStatus = "PENDING"
	StatusAuthorizing PaymentStatus = "AUTHORIZING"
	StatusCaptured    PaymentStatus = "CAPTURED"
	StatusSettled     PaymentStatus = "SETTLED"
	StatusFailed      PaymentStatus = "FAILED"
)

// PaymentMethod mirrors the database schema values.
type PaymentMethod string

const (
	MethodCreditDebit  PaymentMethod = "credit_debit"
	MethodNFCApplePay  PaymentMethod = "nfc_apple_pay"
	MethodNFCGooglePay PaymentMethod = "nfc_google_pay"
	MethodQRCode       PaymentMethod = "qr_code"
	MethodCashRecycler PaymentMethod = "cash_recycler"
)

// Payment represents a payment attempt.
type Payment struct {
	PaymentID        uuid.UUID
	OrderID          uuid.UUID
	KioskID          uuid.UUID
	StoreID          uuid.UUID
	IdempotencyKey   uuid.UUID
	AmountCents      int
	Currency         string
	Method           PaymentMethod
	Status           PaymentStatus
	VerifoneToken    string
	VerifoneAuthCode string
	CardBrand        string
	CardLastFour     string
	DeclineReason    string
	ReceiptText      string
	IsOfflineToken   bool
	OfflineTokenHMAC string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ErrInvalidTransition is returned when a status change violates the state machine.
var ErrInvalidTransition = errors.New("payment: invalid status transition")

// Transition validates and applies a status change.
func (p *Payment) Transition(to PaymentStatus) error {
	for _, s := range validTransitions(p.Status) {
		if s == to {
			p.Status = to
			p.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return ErrInvalidTransition
}

func validTransitions(from PaymentStatus) []PaymentStatus {
	switch from {
	case StatusPending:
		return []PaymentStatus{StatusAuthorizing, StatusFailed}
	case StatusAuthorizing:
		return []PaymentStatus{StatusCaptured, StatusFailed}
	case StatusCaptured:
		return []PaymentStatus{StatusSettled, StatusFailed}
	case StatusSettled, StatusFailed:
		return nil
	default:
		return nil
	}
}

// IsTerminal returns true if no further transitions are allowed.
func (p *Payment) IsTerminal() bool {
	return p.Status == StatusSettled || p.Status == StatusFailed
}

// PaymentStatusFromProto maps a proto PaymentStatus to the domain status.
func PaymentStatusFromProto(s paymentv1.PaymentStatus) PaymentStatus {
	switch s {
	case paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING:
		return StatusPending
	case paymentv1.PaymentStatus_PAYMENT_STATUS_AUTHORIZED:
		return StatusAuthorizing
	case paymentv1.PaymentStatus_PAYMENT_STATUS_CAPTURED:
		return StatusCaptured
	case paymentv1.PaymentStatus_PAYMENT_STATUS_DECLINED:
		return StatusFailed
	case paymentv1.PaymentStatus_PAYMENT_STATUS_VOIDED, paymentv1.PaymentStatus_PAYMENT_STATUS_REFUNDED:
		return StatusFailed
	default:
		return ""
	}
}

// OfflineToken represents a payment token queued for connectivity restoration.
type OfflineToken struct {
	TokenID             uuid.UUID
	StoreID             uuid.UUID
	KioskID             uuid.UUID
	CartID              uuid.UUID
	AmountCents         int
	Currency            string
	Method              PaymentMethod
	VerifoneOpaqueToken string
	HMACSignature       string
	ExpiresAt           time.Time
	SettledAt           *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
