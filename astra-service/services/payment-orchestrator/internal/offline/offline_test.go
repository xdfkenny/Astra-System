package offline

import (
	"context"
	"testing"
	"time"

	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
	"github.com/google/uuid"
)

func TestVerifier_Verify(t *testing.T) {
	secret := []byte("offline-secret")
	v := NewVerifier(secret)

	token := &domain.OfflineToken{
		TokenID:             uuid.Must(uuid.NewV7()),
		StoreID:             uuid.Must(uuid.NewV7()),
		KioskID:             uuid.Must(uuid.NewV7()),
		CartID:              uuid.Must(uuid.NewV7()),
		AmountCents:         1234,
		Currency:            "USD",
		Method:              domain.MethodCreditDebit,
		VerifoneOpaqueToken: "opaque-token",
		ExpiresAt:           time.Now().UTC().Add(time.Hour),
	}
	token.HMACSignature = Sign(token, secret)

	if err := v.Verify(token); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifier_Verify_Tampered(t *testing.T) {
	secret := []byte("offline-secret")
	v := NewVerifier(secret)

	token := &domain.OfflineToken{
		TokenID:             uuid.Must(uuid.NewV7()),
		StoreID:             uuid.Must(uuid.NewV7()),
		KioskID:             uuid.Must(uuid.NewV7()),
		CartID:              uuid.Must(uuid.NewV7()),
		AmountCents:         1234,
		Currency:            "USD",
		Method:              domain.MethodCreditDebit,
		VerifoneOpaqueToken: "opaque-token",
		ExpiresAt:           time.Now().UTC().Add(time.Hour),
	}
	token.HMACSignature = Sign(token, secret)
	token.AmountCents = 9999

	if err := v.Verify(token); err == nil {
		t.Fatal("expected verification failure for tampered token")
	}
}

func TestVerifier_Verify_MissingSignature(t *testing.T) {
	v := NewVerifier([]byte("secret"))
	token := &domain.OfflineToken{TokenID: uuid.Must(uuid.NewV7())}
	if err := v.Verify(token); err == nil {
		t.Fatal("expected error for missing signature")
	}
}

func TestSettler_SettleBatch(t *testing.T) {
	secret := []byte("offline-secret")
	verifier := NewVerifier(secret)
	gateway := &mockGateway{}
	settler := NewSettler(verifier, gateway)

	token := &domain.OfflineToken{
		TokenID:             uuid.Must(uuid.NewV7()),
		StoreID:             uuid.Must(uuid.NewV7()),
		KioskID:             uuid.Must(uuid.NewV7()),
		CartID:              uuid.Must(uuid.NewV7()),
		AmountCents:         1000,
		Currency:            "USD",
		Method:              domain.MethodCreditDebit,
		VerifoneOpaqueToken: "tok-1",
		ExpiresAt:           time.Now().UTC().Add(time.Hour),
	}
	token.HMACSignature = Sign(token, secret)

	results, err := settler.SettleBatch(t.Context(), []*domain.OfflineToken{token})
	if err != nil {
		t.Fatalf("settle batch: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != domain.StatusSettled {
		t.Fatalf("expected SETTLED, got %s", results[0].Status)
	}
	if gateway.settleCalls != 1 {
		t.Fatalf("expected 1 settle call, got %d", gateway.settleCalls)
	}
}

func TestSettler_SettleBatch_Expired(t *testing.T) {
	secret := []byte("offline-secret")
	verifier := NewVerifier(secret)
	settler := NewSettler(verifier, &mockGateway{})

	token := &domain.OfflineToken{
		TokenID:             uuid.Must(uuid.NewV7()),
		StoreID:             uuid.Must(uuid.NewV7()),
		KioskID:             uuid.Must(uuid.NewV7()),
		CartID:              uuid.Must(uuid.NewV7()),
		AmountCents:         1000,
		Currency:            "USD",
		Method:              domain.MethodCreditDebit,
		VerifoneOpaqueToken: "tok-1",
		ExpiresAt:           time.Now().UTC().Add(-time.Hour),
	}
	token.HMACSignature = Sign(token, secret)

	results, err := settler.SettleBatch(t.Context(), []*domain.OfflineToken{token})
	if err != nil {
		t.Fatalf("settle batch: %v", err)
	}
	if results[0].Status != domain.StatusFailed {
		t.Fatalf("expected FAILED, got %s", results[0].Status)
	}
}

type mockGateway struct {
	settleCalls int
}

func (m *mockGateway) Authorize(_ context.Context, _ *client.AuthorizeRequest) (*client.AuthorizeResponse, error) {
	return nil, nil
}

func (m *mockGateway) Capture(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockGateway) Settle(_ context.Context, _, _ string) error {
	m.settleCalls++
	return nil
}
