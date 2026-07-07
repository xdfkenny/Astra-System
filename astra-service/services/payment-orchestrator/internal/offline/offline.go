// Package offline implements offline-token settlement: HMAC verification and
// batch submission to the Verifone sidecar.
package offline

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
	"github.com/google/uuid"
)

// SettlementResult records the outcome of settling one offline token.
type SettlementResult struct {
	TokenID       uuid.UUID
	Status        domain.PaymentStatus
	DeclineReason string
	SettledAt     time.Time
}

// Verifier checks offline token HMAC signatures.
type Verifier struct {
	secret []byte
}

// NewVerifier creates an HMAC verifier with the provided secret.
func NewVerifier(secret []byte) *Verifier {
	return &Verifier{secret: secret}
}

// Verify checks the HMAC-SHA256 signature of an offline token payload.
// The canonical form is: token_id|store_id|kiosk_id|cart_id|amount_cents|currency|method|verifone_opaque_token|expires_at(RFC3339)
func (v *Verifier) Verify(token *domain.OfflineToken) error {
	if token.HMACSignature == "" {
		return fmt.Errorf("offline: missing hmac signature")
	}
	canonical := canonicalForm(token)
	mac := hmac.New(sha256.New, v.secret)
	if _, err := mac.Write([]byte(canonical)); err != nil {
		return fmt.Errorf("offline: compute hmac: %w", err)
	}
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(token.HMACSignature)) {
		return fmt.Errorf("offline: signature mismatch")
	}
	return nil
}

func canonicalForm(t *domain.OfflineToken) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d|%s|%s|%s|%s",
		t.TokenID.String(),
		t.StoreID.String(),
		t.KioskID.String(),
		t.CartID.String(),
		t.AmountCents,
		t.Currency,
		t.Method,
		t.VerifoneOpaqueToken,
		t.ExpiresAt.Format(time.RFC3339),
	)
}

// Sign computes the HMAC-SHA256 signature for an offline token.
func Sign(token *domain.OfflineToken, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(canonicalForm(token)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Settler settles verified offline tokens through the Verifone gateway.
type Settler struct {
	verifier *Verifier
	gateway  client.Gateway
}

// NewSettler creates a batch offline-token settler.
func NewSettler(verifier *Verifier, gateway client.Gateway) *Settler {
	return &Settler{verifier: verifier, gateway: gateway}
}

// SettleBatch verifies and settles each token in the batch. Failures are
// recorded per-token; the function returns an error only if the entire batch
// cannot be processed.
func (s *Settler) SettleBatch(ctx context.Context, tokens []*domain.OfflineToken) ([]SettlementResult, error) {
	results := make([]SettlementResult, 0, len(tokens))
	for _, token := range tokens {
		if err := s.verifier.Verify(token); err != nil {
			results = append(results, SettlementResult{
				TokenID:       token.TokenID,
				Status:        domain.StatusFailed,
				DeclineReason: err.Error(),
			})
			continue
		}
		if token.ExpiresAt.Before(time.Now().UTC()) {
			results = append(results, SettlementResult{
				TokenID:       token.TokenID,
				Status:        domain.StatusFailed,
				DeclineReason: "offline: token expired",
			})
			continue
		}

		if err := s.gateway.Settle(ctx, token.TokenID.String(), token.VerifoneOpaqueToken); err != nil {
			results = append(results, SettlementResult{
				TokenID:       token.TokenID,
				Status:        domain.StatusFailed,
				DeclineReason: err.Error(),
			})
			continue
		}

		now := time.Now().UTC()
		results = append(results, SettlementResult{
			TokenID:   token.TokenID,
			Status:    domain.StatusSettled,
			SettledAt: now,
		})
	}
	return results, nil
}
