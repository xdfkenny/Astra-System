package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/repository"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type stubHealth struct{}

func (stubHealth) Check(ctx context.Context) error { return nil }

type mockStore struct {
	payments      map[uuid.UUID]*domain.Payment
	created       []*domain.Payment
	updateErr     error
	lastFrom      domain.PaymentStatus
	lastTo        domain.PaymentStatus
}

func newMockStore() *mockStore {
	return &mockStore{payments: make(map[uuid.UUID]*domain.Payment)}
}

func (m *mockStore) Create(ctx context.Context, p *domain.Payment) error {
	if _, exists := m.payments[p.IdempotencyKey]; exists {
		return repository.ErrDuplicateIdempotencyKey
	}
	m.payments[p.PaymentID] = p
	m.created = append(m.created, p)
	return nil
}

func (m *mockStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	return m.payments[id], nil
}

func (m *mockStore) UpdateStatus(ctx context.Context, id uuid.UUID, from, to domain.PaymentStatus, token, reason string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.lastFrom = from
	m.lastTo = to
	p := m.payments[id]
	if p == nil {
		return errors.New("not found")
	}
	p.Status = to
	return nil
}

type mockLocker struct{}

func (mockLocker) Lock(ctx context.Context, key uuid.UUID) (bool, error) { return true, nil }
func (mockLocker) Unlock(ctx context.Context, key uuid.UUID) error        { return nil }

type mockVerifone struct {
	authorizeResp *client.AuthorizeResponse
	authorizeErr  error
	captureErr    error
	settleErr     error
}

func (m *mockVerifone) Authorize(ctx context.Context, req *client.AuthorizeRequest) (*client.AuthorizeResponse, error) {
	if m.authorizeErr != nil {
		return nil, m.authorizeErr
	}
	return m.authorizeResp, nil
}

func (m *mockVerifone) Capture(ctx context.Context, paymentID, verifoneToken string) error { return m.captureErr }
func (m *mockVerifone) Settle(ctx context.Context, paymentID, verifoneToken string) error   { return m.settleErr }

func newTestApp(t *testing.T, store service.Store, verifone client.Gateway) (*fiber.App, *REST) {
	t.Helper()
	svc := service.NewPayment(store, mockLocker{}, verifone)
	h := NewREST(svc, verifone, []byte("test-secret"), nil)
	app := fiber.New()
	h.RegisterRoutes(app, stubHealth{})
	return app, h
}

func TestCreatePayment_Approved(t *testing.T) {
	store := newMockStore()
	vf := &mockVerifone{authorizeResp: &client.AuthorizeResponse{Status: "APPROVED", VerifoneToken: "tok-1", AuthCode: "auth-1"}}
	app, _ := newTestApp(t, store, vf)

	body := `{"order_id":"11111111-1111-1111-1111-111111111111","kiosk_id":"22222222-2222-2222-2222-222222222222","amount_cents":1000,"currency":"USD","method":"credit_debit"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(b))
	}
	if len(store.created) != 1 {
		t.Fatalf("expected 1 created payment, got %d", len(store.created))
	}
	if store.created[0].Status != domain.StatusAuthorizing {
		t.Fatalf("expected status AUTHORIZING, got %s", store.created[0].Status)
	}
}

func TestCreatePayment_Declined(t *testing.T) {
	store := newMockStore()
	vf := &mockVerifone{authorizeResp: &client.AuthorizeResponse{Status: "DECLINED", DeclineReason: "insufficient_funds"}}
	app, _ := newTestApp(t, store, vf)

	body := `{"order_id":"11111111-1111-1111-1111-111111111111","kiosk_id":"22222222-2222-2222-2222-222222222222","amount_cents":1000,"currency":"USD","method":"credit_debit"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.New().String())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(b))
	}
	if store.created[0].Status != domain.StatusFailed {
		t.Fatalf("expected status FAILED, got %s", store.created[0].Status)
	}
}

func TestCreatePayment_MissingIdempotencyKey(t *testing.T) {
	app, _ := newTestApp(t, newMockStore(), &mockVerifone{})
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCapturePayment_Success(t *testing.T) {
	store := newMockStore()
	p := &domain.Payment{
		PaymentID:     uuid.New(),
		Status:        domain.StatusAuthorizing,
		VerifoneToken: "tok-1",
		CreatedAt:     time.Now().UTC(),
	}
	store.payments[p.PaymentID] = p
	vf := &mockVerifone{}
	app, _ := newTestApp(t, store, vf)

	req := httptest.NewRequest(http.MethodPost, "/v1/payments/"+p.PaymentID.String()+"/capture", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}
	if store.lastFrom != domain.StatusAuthorizing || store.lastTo != domain.StatusCaptured {
		t.Fatalf("unexpected transition from %s to %s", store.lastFrom, store.lastTo)
	}
}

func TestSettlePayment_WrongStatus(t *testing.T) {
	store := newMockStore()
	p := &domain.Payment{PaymentID: uuid.New(), Status: domain.StatusAuthorizing, CreatedAt: time.Now().UTC()}
	store.payments[p.PaymentID] = p
	app, _ := newTestApp(t, store, &mockVerifone{})

	req := httptest.NewRequest(http.MethodPost, "/v1/payments/"+p.PaymentID.String()+"/settle", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestWebhook_ValidSignature(t *testing.T) {
	store := newMockStore()
	p := &domain.Payment{PaymentID: uuid.New(), Status: domain.StatusAuthorizing, CreatedAt: time.Now().UTC()}
	store.payments[p.PaymentID] = p
	app, _ := newTestApp(t, store, &mockVerifone{})

	body := []byte(`{"payment_id":"` + p.PaymentID.String() + `","status":"CAPTURED"}`)
	sig := hmacSig(body, []byte("test-secret"))
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/webhooks/verifone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Verifone-Signature", sig)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(b))
	}
	if store.lastTo != domain.StatusCaptured {
		t.Fatalf("expected status CAPTURED, got %s", store.lastTo)
	}
}

func TestWebhook_InvalidSignature(t *testing.T) {
	app, _ := newTestApp(t, newMockStore(), &mockVerifone{})
	body := []byte(`{"payment_id":"11111111-1111-1111-1111-111111111111","status":"CAPTURED"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/webhooks/verifone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Verifone-Signature", "bad-sig")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func hmacSig(body, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
