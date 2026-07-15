// Package repository persists payment state and emits outbox events atomically
// within the same transaction.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/astra-service/go-common/outbox"
	eventsv1 "github.com/astra-systems/astra-service/proto/gen/go/events"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/events"
	"github.com/google/uuid"
)

// PaymentRepository provides SQL persistence for payments.
type PaymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository constructs a repository backed by db.
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// ErrDuplicateIdempotencyKey indicates the idempotency key already exists.
var ErrDuplicateIdempotencyKey = errors.New("payment: duplicate idempotency key")

// Create inserts a payment and its PaymentInitiated outbox event atomically.
func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("payment_repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var existing uuid.UUID
	err = tx.QueryRowContext(ctx, `
		SELECT payment_id FROM payments WHERE idempotency_key = $1`, p.IdempotencyKey).Scan(&existing)
	if err == nil {
		return ErrDuplicateIdempotencyKey
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("payment_repository: idempotency check: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (
			payment_id, order_id, kiosk_id, store_id, idempotency_key, amount_cents, currency,
			method, status, verifone_token, verifone_auth_code, card_brand, card_last_four,
			decline_reason, receipt_text, is_offline_token, offline_token_hmac, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $18)`,
		p.PaymentID, p.OrderID, p.KioskID, p.StoreID, p.IdempotencyKey, p.AmountCents, p.Currency,
		string(p.Method), string(p.Status), p.VerifoneToken, p.VerifoneAuthCode, p.CardBrand, p.CardLastFour,
		p.DeclineReason, p.ReceiptText, p.IsOfflineToken, p.OfflineTokenHMAC, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("payment_repository: insert payment: %w", err)
	}

	metadata := map[string]string{"kiosk_id": p.KioskID.String(), "store_id": p.StoreID.String()}
	payload := &eventsv1.PaymentInitiated{
		PaymentId:   p.PaymentID.String(),
		OrderId:     p.OrderID.String(),
		KioskId:     p.KioskID.String(),
		AmountCents: int64(p.AmountCents),
		Currency:    p.Currency,
		Method:      string(p.Method),
	}
	env, err := events.Envelope(events.EventTypePaymentInitiated, p.PaymentID.String(), 1, metadata, payload)
	if err != nil {
		return fmt.Errorf("payment_repository: build envelope: %w", err)
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("payment_repository: marshal envelope: %w", err)
	}

	if err := outbox.InsertWithinTx(ctx, tx, outbox.Entry{
		EventID:       env.EventId,
		AggregateType: "payment",
		AggregateID:   p.PaymentID.String(),
		EventType:     events.EventTypePaymentInitiated,
		Payload:       data,
		OccurredAtMs:  p.CreatedAt.UnixMilli(),
	}); err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID returns a payment by id or nil if not found.
func (r *PaymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	var p domain.Payment
	var token, authCode, cardBrand, cardLastFour, declineReason, receiptText, offlineHMAC sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT payment_id, order_id, kiosk_id, store_id, idempotency_key, amount_cents, currency,
		       method, status, verifone_token, verifone_auth_code, card_brand, card_last_four,
		       decline_reason, receipt_text, is_offline_token, offline_token_hmac, created_at, updated_at
		FROM payments WHERE payment_id = $1`, id).Scan(
		&p.PaymentID, &p.OrderID, &p.KioskID, &p.StoreID, &p.IdempotencyKey, &p.AmountCents, &p.Currency,
		&p.Method, &p.Status, &token, &authCode, &cardBrand, &cardLastFour,
		&declineReason, &receiptText, &p.IsOfflineToken, &offlineHMAC, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("payment_repository: get by id: %w", err)
	}
	p.VerifoneToken = token.String
	p.VerifoneAuthCode = authCode.String
	p.CardBrand = cardBrand.String
	p.CardLastFour = cardLastFour.String
	p.DeclineReason = declineReason.String
	p.ReceiptText = receiptText.String
	p.OfflineTokenHMAC = offlineHMAC.String
	return &p, nil
}

// UpdateStatus applies an optimistic status transition and writes an outbox
// event for the transition.
func (r *PaymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, from, to domain.PaymentStatus, verifoneToken, declineReason string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("payment_repository: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.ExecContext(ctx, `
		UPDATE payments
		SET status = $1, verifone_token = $2, decline_reason = $3, updated_at = $4
		WHERE payment_id = $5 AND status = $6`,
		string(to), verifoneToken, declineReason, time.Now().UTC(), id, string(from),
	)
	if err != nil {
		return fmt.Errorf("payment_repository: update status: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("payment_repository: rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrInvalidTransition
	}

	env, err := r.statusEnvelope(ctx, tx, id, to, declineReason)
	if err != nil {
		return err
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("payment_repository: marshal envelope: %w", err)
	}

	if err := outbox.InsertWithinTx(ctx, tx, outbox.Entry{
		EventID:       env.EventId,
		AggregateType: "payment",
		AggregateID:   id.String(),
		EventType:     envTypeForStatus(to),
		Payload:       data,
		OccurredAtMs:  time.Now().UnixMilli(),
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PaymentRepository) statusEnvelope(ctx context.Context, tx *sql.Tx, id uuid.UUID, to domain.PaymentStatus, declineReason string) (*eventsv1.EventEnvelope, error) {
	var orderID, kioskID string
	if err := tx.QueryRowContext(ctx, `SELECT order_id, kiosk_id FROM payments WHERE payment_id = $1`, id).Scan(&orderID, &kioskID); err != nil {
		return nil, fmt.Errorf("payment_repository: load payment ids: %w", err)
	}
	metadata := map[string]string{"kiosk_id": kioskID}

	switch to {
	case domain.StatusCaptured, domain.StatusSettled:
		payload := &eventsv1.PaymentConfirmed{
			PaymentId: id.String(),
			OrderId:   orderID,
			Status:    string(to),
		}
		return events.Envelope(events.EventTypePaymentConfirmed, id.String(), 2, metadata, payload)
	case domain.StatusFailed:
		payload := &events.PaymentFailed{
			PaymentID:     id.String(),
			OrderID:       orderID,
			KioskID:       kioskID,
			Status:        string(to),
			DeclineReason: declineReason,
		}
		return events.Envelope(events.EventTypePaymentFailed, id.String(), 2, metadata, payload)
	default:
		return nil, fmt.Errorf("payment_repository: no envelope for status %s", to)
	}
}

func envTypeForStatus(s domain.PaymentStatus) string {
	switch s {
	case domain.StatusCaptured, domain.StatusSettled:
		return events.EventTypePaymentConfirmed
	case domain.StatusFailed:
		return events.EventTypePaymentFailed
	default:
		return ""
	}
}

// RawJSON is a helper for handlers that need to pass JSON payloads.
func RawJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
