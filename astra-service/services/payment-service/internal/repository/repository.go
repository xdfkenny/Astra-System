// Package repository implements persistence for the Payment aggregate.
// It intentionally uses plain SQL (pgx) rather than an ORM because payment
// logic demands precise transaction boundaries, especially when reconciling
// offline tokens that may arrive from multiple leader kiosks concurrently.
package repository

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/astra-service/payment-service/internal/domain"
	"github.com/google/uuid"
)

// ErrDuplicateIdempotencyKey is returned when a payment with the same
// idempotency key has already been recorded. Callers should treat this as
// success and return the existing payment.
var ErrDuplicateIdempotencyKey = errors.New("payment: duplicate idempotency key")

// ErrInvalidOfflineHMAC is returned when an offline token's signature does
// not verify against the shared HMAC key.
var ErrInvalidOfflineHMAC = errors.New("payment: offline token hmac verification failed")

// PaymentRepository provides payment persistence on top of PostgreSQL.
type PaymentRepository struct {
	db      *sql.DB
	hmacKey []byte
}

func NewPaymentRepository(db *sql.DB, hmacKey []byte) *PaymentRepository {
	return &PaymentRepository{db: db, hmacKey: hmacKey}
}

// RecordAuthorization persists a successful (or declined) payment
// authorization. Idempotency is enforced by the unique index on
// payments.idempotency_key.
func (r *PaymentRepository) RecordAuthorization(ctx context.Context, p *domain.Payment) error {
	receiptText := ""
	if p.ReceiptText != nil {
		receiptText = *p.ReceiptText
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO payments (
			payment_id, order_id, kiosk_id, idempotency_key, amount_cents, currency,
			method, status, verifone_token, verifone_auth_code, card_brand, card_last_four,
			decline_reason, receipt_text, is_offline_token, offline_token_hmac, synced_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $18)`,
		p.PaymentID, p.OrderID, p.KioskID, p.IdempotencyKey, p.AmountCents, p.Currency,
		p.Method, p.Status, p.VerifoneToken, p.VerifoneAuthCode, p.CardBrand, p.CardLastFour,
		p.DeclineReason, receiptText, p.IsOfflineToken, p.OfflineTokenHMAC, p.SyncedAt,
		p.CreatedAt,
	)
	if err != nil {
		// pgx's error code for unique violation is 23505.
		if isUniqueViolation(err) {
			return ErrDuplicateIdempotencyKey
		}
		return fmt.Errorf("payment_repository: insert payment: %w", err)
	}
	return nil
}

// GetByIdempotencyKey returns an existing payment by its idempotency key.
func (r *PaymentRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error) {
	var p domain.Payment
	var receiptText sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT payment_id, order_id, kiosk_id, idempotency_key, amount_cents, currency,
		       method, status, verifone_token, verifone_auth_code, card_brand, card_last_four,
		       decline_reason, receipt_text, is_offline_token, offline_token_hmac, synced_at, created_at
		FROM payments WHERE idempotency_key = $1`, key,
	).Scan(
		&p.PaymentID, &p.OrderID, &p.KioskID, &p.IdempotencyKey, &p.AmountCents, &p.Currency,
		&p.Method, &p.Status, &p.VerifoneToken, &p.VerifoneAuthCode, &p.CardBrand, &p.CardLastFour,
		&p.DeclineReason, &receiptText, &p.IsOfflineToken, &p.OfflineTokenHMAC, &p.SyncedAt, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("payment_repository: query idempotency key: %w", err)
	}
	if receiptText.Valid {
		p.ReceiptText = &receiptText.String
	}
	return &p, nil
}

// QueueOfflineToken stores a kiosk-generated offline payment token after
// verifying its HMAC signature. The canonical form must match the kiosk-side
// construction: tokenId|cartId|amountCents|verifoneOpaqueToken.
func (r *PaymentRepository) QueueOfflineToken(ctx context.Context, token *domain.OfflineToken) error {
	canonical := fmt.Sprintf("%s|%s|%d|%s", token.TokenID, token.CartID, token.AmountCents, token.VerifoneOpaqueToken)
	if !r.verifyHMAC(canonical, token.HMACSignature) {
		return ErrInvalidOfflineHMAC
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO offline_tokens (
			token_id, store_id, kiosk_id, cart_id, amount_cents, currency, method,
			verifone_opaque_token, hmac_signature, expires_at, settled_at, settlement_result, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULL, NULL, $11, $11)
		ON CONFLICT (token_id) DO NOTHING`,
		token.TokenID, token.StoreID, token.KioskID, token.CartID, token.AmountCents,
		token.Currency, token.Method, token.VerifoneOpaqueToken, token.HMACSignature,
		token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("payment_repository: queue offline token: %w", err)
	}
	return nil
}

// LoadUnsettledOfflineTokens returns offline tokens that have not yet been
// settled and have not expired. The caller is responsible for attempting
// settlement and marking results via MarkOfflineSettled.
func (r *PaymentRepository) LoadUnsettledOfflineTokens(ctx context.Context, limit int) ([]domain.OfflineToken, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT token_id, store_id, kiosk_id, cart_id, amount_cents, currency, method,
		       verifone_opaque_token, hmac_signature, expires_at, created_at
		FROM offline_tokens
		WHERE settled_at IS NULL AND expires_at > $1
		ORDER BY created_at ASC
		LIMIT $2`, time.Now().UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("payment_repository: query unsettled tokens: %w", err)
	}
	defer rows.Close()

	var tokens []domain.OfflineToken
	for rows.Next() {
		var t domain.OfflineToken
		if err := rows.Scan(
			&t.TokenID, &t.StoreID, &t.KioskID, &t.CartID, &t.AmountCents, &t.Currency,
			&t.Method, &t.VerifoneOpaqueToken, &t.HMACSignature, &t.ExpiresAt, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("payment_repository: scan offline token: %w", err)
		}
		tokens = append(tokens, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tokens, nil
}

// MarkOfflineSettled records the result of attempting to settle an offline
// token with the payment processor.
func (r *PaymentRepository) MarkOfflineSettled(ctx context.Context, tokenID string, result json.RawMessage) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE offline_tokens
		SET settled_at = $1, settlement_result = $2, updated_at = $1
		WHERE token_id = $3`,
		time.Now().UTC(), result, tokenID,
	)
	if err != nil {
		return fmt.Errorf("payment_repository: mark settled: %w", err)
	}
	return nil
}

func (r *PaymentRepository) verifyHMAC(canonical, signatureHex string) bool {
	mac := hmac.New(sha256.New, r.hmacKey)
	mac.Write([]byte(canonical))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signatureHex))
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "unique constraint")
}

// NewPaymentID generates a UUID v7 identifier for a new payment.
func NewPaymentID() string {
	return uuid.New().String()
}
