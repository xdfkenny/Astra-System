// Package handler adapts NATS JetStream messages into payment domain
// operations. It consumes:
//   - astra.payment.record_authorization: record an authorized payment
//   - astra.payment.offline_token_received: queue a kiosk offline token
// and runs a background settlement loop for unsettled offline tokens.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/payment-service/internal/domain"
	"github.com/astra-service/payment-service/internal/repository"
	"github.com/nats-io/nats.go/jetstream"
)

type PaymentHandler struct {
	repo      *repository.PaymentRepository
	bus       *eventbus.Bus
	settlementEnabled bool
	settlementInterval time.Duration
}

func NewPaymentHandler(repo *repository.PaymentRepository, bus *eventbus.Bus, settlementEnabled bool, settlementInterval time.Duration) *PaymentHandler {
	return &PaymentHandler{
		repo:               repo,
		bus:                bus,
		settlementEnabled:  settlementEnabled,
		settlementInterval: settlementInterval,
	}
}

// RecordAuthorizationCommand is published by the API gateway after the
// Verifone sidecar returns an authorization result.
type RecordAuthorizationCommand struct {
	PaymentID      string  `json:"paymentId"`
	OrderID        string  `json:"orderId"`
	KioskID        string  `json:"kioskId"`
	IdempotencyKey string  `json:"idempotencyKey"`
	AmountCents    int     `json:"amountCents"`
	Currency       string  `json:"currency"`
	Method         string  `json:"method"`
	Status         string  `json:"status"`
	VerifoneToken  string  `json:"verifoneToken"`
	AuthCode       string  `json:"authCode,omitempty"`
	CardBrand      string  `json:"cardBrand,omitempty"`
	CardLastFour   string  `json:"cardLastFour,omitempty"`
	DeclineReason  string  `json:"declineReason,omitempty"`
	ReceiptText    *string `json:"receiptText,omitempty"`
	IsOfflineToken bool    `json:"isOfflineToken"`
	OfflineHMAC    string  `json:"offlineHmac,omitempty"`
}

func (h *PaymentHandler) HandleRecordAuthorization(ctx context.Context, msg jetstream.Msg) error {
	var cmd RecordAuthorizationCommand
	if err := json.Unmarshal(msg.Data(), &cmd); err != nil {
		return fmt.Errorf("payment_handler: unmarshal record-auth command: %w", err)
	}

	// Idempotency check: if the idempotency key already exists, return success
	// without mutating state. This makes the command safely retryable.
	existing, err := h.repo.GetByIdempotencyKey(ctx, cmd.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("payment_handler: idempotency lookup: %w", err)
	}
	if existing != nil {
		return nil
	}

	p := &domain.Payment{
		PaymentID:      cmd.PaymentID,
		OrderID:        cmd.OrderID,
		KioskID:        cmd.KioskID,
		IdempotencyKey: cmd.IdempotencyKey,
		AmountCents:    cmd.AmountCents,
		Currency:       cmd.Currency,
		Method:         cmd.Method,
		Status:         cmd.Status,
		VerifoneToken:  cmd.VerifoneToken,
		VerifoneAuthCode: cmd.AuthCode,
		CardBrand:      cmd.CardBrand,
		CardLastFour:   cmd.CardLastFour,
		DeclineReason:  cmd.DeclineReason,
		ReceiptText:    cmd.ReceiptText,
		IsOfflineToken: cmd.IsOfflineToken,
		OfflineTokenHMAC: cmd.OfflineHMAC,
		CreatedAt:      time.Now().UTC(),
	}

	if err := h.repo.RecordAuthorization(ctx, p); err != nil {
		if err == repository.ErrDuplicateIdempotencyKey {
			return nil
		}
		return fmt.Errorf("payment_handler: record authorization: %w", err)
	}

	// Publish a domain event for downstream consumers (order fulfillment,
	// analytics, receipt dispatch).
	event := map[string]interface{}{
		"paymentId": cmd.PaymentID,
		"orderId":   cmd.OrderID,
		"status":    cmd.Status,
		"amountCents": cmd.AmountCents,
		"method":    cmd.Method,
		"recordedAt": time.Now().UTC().Format(time.RFC3339Nano),
	}
	eventBytes, _ := json.Marshal(event)
	if err := h.bus.Publish(ctx, "astra.payment.recorded.v1", eventBytes); err != nil {
		// Log but do not fail: the payment is already persisted; a missing
		// downstream event will be reconciled by the outbox/audit pipeline.
		log.Printf("payment_handler: failed to publish recorded event: %v", err)
	}

	return nil
}

// OfflineTokenCommand is published by the Raft leader when it uploads a batch
// of offline payment tokens from the kiosk mesh.
type OfflineTokenCommand struct {
	TokenID             string    `json:"tokenId"`
	StoreID             string    `json:"storeId"`
	KioskID             string    `json:"kioskId"`
	CartID              string    `json:"cartId"`
	AmountCents         int       `json:"amountCents"`
	Currency            string    `json:"currency"`
	Method              string    `json:"method"`
	VerifoneOpaqueToken string    `json:"verifoneOpaqueToken"`
	HMACSignature       string    `json:"hmacSignature"`
	ExpiresAt           time.Time `json:"expiresAt"`
	CreatedAt           time.Time `json:"createdAt"`
}

func (h *PaymentHandler) HandleOfflineToken(ctx context.Context, msg jetstream.Msg) error {
	var cmd OfflineTokenCommand
	if err := json.Unmarshal(msg.Data(), &cmd); err != nil {
		return fmt.Errorf("payment_handler: unmarshal offline-token command: %w", err)
	}

	token := &domain.OfflineToken{
		TokenID:             cmd.TokenID,
		StoreID:             cmd.StoreID,
		KioskID:             cmd.KioskID,
		CartID:              cmd.CartID,
		AmountCents:         cmd.AmountCents,
		Currency:            cmd.Currency,
		Method:              cmd.Method,
		VerifoneOpaqueToken: cmd.VerifoneOpaqueToken,
		HMACSignature:       cmd.HMACSignature,
		ExpiresAt:           cmd.ExpiresAt,
		CreatedAt:           cmd.CreatedAt,
	}

	if err := h.repo.QueueOfflineToken(ctx, token); err != nil {
		return fmt.Errorf("payment_handler: queue offline token: %w", err)
	}
	return nil
}

// RunSettlementLoop polls for unsettled offline tokens and attempts to settle
// them with the payment processor. In production this integrates with the
// Verifone cloud API; the scaffold simulates settlement success and records
// the result.
func (h *PaymentHandler) RunSettlementLoop(ctx context.Context) {
	if !h.settlementEnabled {
		log.Println("payment_handler: offline settlement loop disabled")
		return
	}

	ticker := time.NewTicker(h.settlementInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := h.settleBatch(ctx); err != nil {
				log.Printf("payment_handler: settlement batch error: %v", err)
			}
		}
	}
}

func (h *PaymentHandler) settleBatch(ctx context.Context) error {
	tokens, err := h.repo.LoadUnsettledOfflineTokens(ctx, 50)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}

	for _, token := range tokens {
		// Production integration: call Verifone cloud settlement endpoint
		// using token.VerifoneOpaqueToken. Here we simulate success.
		result := domain.SettlementResult{
			SettledAt:   time.Now().UTC(),
			Success:     true,
			ProcessorID: "simulated-settlement",
		}
		resultBytes, _ := json.Marshal(result)
		if err := h.repo.MarkOfflineSettled(ctx, token.TokenID, resultBytes); err != nil {
			log.Printf("payment_handler: mark settled %s: %v", token.TokenID, err)
			continue
		}
		log.Printf("payment_handler: settled offline token %s", token.TokenID)
	}
	return nil
}
