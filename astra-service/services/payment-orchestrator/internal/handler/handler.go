// Package handler exposes the payment orchestrator REST API and webhook
// endpoints. It delegates domain logic to the service layer.
package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/astra-service/go-common/observability"
	"github.com/astra-systems/astra-service/proto/gen/go/payment"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/offline"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/service"
	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/webhook"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/google/uuid"
)

// REST exposes payment REST endpoints.
type REST struct {
	svc            *service.Payment
	verifone       client.Gateway
	webhookSecret  []byte
	offlineSettler *offline.Settler
}

// NewREST builds a REST handler.
func NewREST(svc *service.Payment, verifone client.Gateway, webhookSecret []byte, offlineSettler *offline.Settler) *REST {
	return &REST{
		svc:            svc,
		verifone:       verifone,
		webhookSecret:  webhookSecret,
		offlineSettler: offlineSettler,
	}
}

// RegisterRoutes wires payment endpoints.
func (h *REST) RegisterRoutes(app *fiber.App, health observability.Checkable) {
	v1 := app.Group("/v1/payments")
	v1.Post("/", h.CreatePayment)
	v1.Post("/:id/capture", h.CapturePayment)
	v1.Post("/:id/settle", h.SettlePayment)
	v1.Post("/:id/refund", h.RefundPayment)

	v1.Post("/webhooks/verifone", h.Webhook)
	app.Post("/v1/offline-tokens/settle", h.SettleOfflineTokens)

	app.Get("/health", adaptor.HTTPHandler(http.HandlerFunc(observability.HealthHandler)))
	app.Get("/live", adaptor.HTTPHandler(http.HandlerFunc(observability.LiveHandler)))
	app.Get("/ready", adaptor.HTTPHandler(observability.ReadyHandler(health)))
	app.Get("/metrics", adaptor.HTTPHandler(observability.MetricsHandler()))
}

// CreatePaymentRequest starts a new payment attempt.
type CreatePaymentRequest struct {
	OrderID        string `json:"order_id"`
	KioskID        string `json:"kiosk_id"`
	AmountCents    int    `json:"amount_cents"`
	Currency       string `json:"currency"`
	Method         string `json:"method"`
	IsOfflineToken bool   `json:"is_offline_token"`
}

func (h *REST) CreatePayment(c fiber.Ctx) error {
	idempotencyKey := c.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "missing_idempotency_key"})
	}

	var req CreatePaymentRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_body"})
	}
	if req.OrderID == "" || req.KioskID == "" || req.AmountCents <= 0 || req.Currency == "" || req.Method == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "missing_required_field"})
	}

	protoMethod, ok := parseMethod(req.Method)
	if !ok {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_method"})
	}

	paymentID := uuid.Must(uuid.NewV7()).String()
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	result, err := h.svc.InitiatePayment(ctx, &payment.PaymentIntent{
		PaymentId:      paymentID,
		OrderId:        req.OrderID,
		KioskId:        req.KioskID,
		IdempotencyKey: idempotencyKey,
		AmountCents:    int64(req.AmountCents),
		Currency:       req.Currency,
		Method:         protoMethod,
		IsOfflineToken: req.IsOfflineToken,
	})
	if err != nil {
		st, _ := grpcStatus(err)
		return c.Status(st).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(protoToMap(result))
}

func (h *REST) CapturePayment(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_payment_id"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	result, err := h.svc.CapturePayment(ctx, &payment.PaymentResult{PaymentId: id.String()})
	if err != nil {
		st, _ := grpcStatus(err)
		return c.Status(st).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(protoToMap(result))
}

func (h *REST) SettlePayment(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_payment_id"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	result, err := h.svc.SettlePayment(ctx, id)
	if err != nil {
		st, _ := grpcStatus(err)
		return c.Status(st).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(protoToMap(result))
}

func (h *REST) RefundPayment(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_payment_id"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	result, err := h.svc.RefundPayment(ctx, &payment.RefundRequest{PaymentId: id.String()})
	if err != nil {
		st, _ := grpcStatus(err)
		return c.Status(st).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(protoToMap(result))
}

func (h *REST) Webhook(c fiber.Ctx) error {
	body := c.Body()
	sig := c.Get("X-Verifone-Signature")
	if err := webhook.VerifySignature(body, sig, h.webhookSecret); err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "invalid_signature"})
	}

	n, err := webhook.ParseNotification(body)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	target, err := webhook.MapStatus(n.Status)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	p, err := h.svc.GetPaymentStatus(ctx, &payment.GetPaymentStatusRequest{PaymentId: n.PaymentID})
	if err != nil {
		st, _ := grpcStatus(err)
		return c.Status(st).JSON(fiber.Map{"error": err.Error()})
	}
	status := domain.PaymentStatusFromProto(p.Status)
	if status == domain.StatusSettled || status == domain.StatusFailed {
		return c.JSON(fiber.Map{"payment_id": p.PaymentId, "status": p.Status})
	}

	result, err := h.svc.UpdateStatus(ctx, uuid.Must(uuid.Parse(n.PaymentID)), target, n.DeclineReason)
	if err != nil {
		st, _ := grpcStatus(err)
		return c.Status(st).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(protoToMap(result))
}

// OfflineTokenRequest is a single offline token in a batch.
type OfflineTokenRequest struct {
	TokenID             string `json:"token_id"`
	StoreID             string `json:"store_id"`
	KioskID             string `json:"kiosk_id"`
	CartID              string `json:"cart_id"`
	AmountCents         int    `json:"amount_cents"`
	Currency            string `json:"currency"`
	Method              string `json:"method"`
	VerifoneOpaqueToken string `json:"verifone_opaque_token"`
	HMACSignature       string `json:"hmac_signature"`
	ExpiresAt           string `json:"expires_at"`
}

// SettleOfflineTokensRequest is a batch settlement request.
type SettleOfflineTokensRequest struct {
	Tokens []OfflineTokenRequest `json:"tokens"`
}

func (h *REST) SettleOfflineTokens(c fiber.Ctx) error {
	var req SettleOfflineTokensRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid_body"})
	}
	if len(req.Tokens) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "empty_batch"})
	}

	tokens := make([]*domain.OfflineToken, 0, len(req.Tokens))
	for _, t := range req.Tokens {
		token, err := parseOfflineToken(t)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		tokens = append(tokens, token)
	}

	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	results, err := h.offlineSettler.SettleBatch(ctx, tokens)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	out := make([]fiber.Map, 0, len(results))
	for _, r := range results {
		out = append(out, fiber.Map{
			"token_id":       r.TokenID.String(),
			"status":         r.Status,
			"decline_reason": r.DeclineReason,
			"settled_at":     r.SettledAt,
		})
	}
	return c.JSON(fiber.Map{"results": out})
}

func parseOfflineToken(t OfflineTokenRequest) (*domain.OfflineToken, error) {
	if t.TokenID == "" || t.StoreID == "" || t.KioskID == "" || t.CartID == "" || t.AmountCents <= 0 || t.Currency == "" || t.Method == "" || t.VerifoneOpaqueToken == "" || t.HMACSignature == "" || t.ExpiresAt == "" {
		return nil, fmt.Errorf("missing offline token field")
	}
	tokenID, err := uuid.Parse(t.TokenID)
	if err != nil {
		return nil, fmt.Errorf("invalid token_id")
	}
	storeID, err := uuid.Parse(t.StoreID)
	if err != nil {
		return nil, fmt.Errorf("invalid store_id")
	}
	kioskID, err := uuid.Parse(t.KioskID)
	if err != nil {
		return nil, fmt.Errorf("invalid kiosk_id")
	}
	cartID, err := uuid.Parse(t.CartID)
	if err != nil {
		return nil, fmt.Errorf("invalid cart_id")
	}
	expiresAt, err := time.Parse(time.RFC3339, t.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("invalid expires_at")
	}
	method, ok := parseDomainMethod(t.Method)
	if !ok {
		return nil, fmt.Errorf("invalid method")
	}
	return &domain.OfflineToken{
		TokenID:             tokenID,
		StoreID:             storeID,
		KioskID:             kioskID,
		CartID:              cartID,
		AmountCents:         t.AmountCents,
		Currency:            t.Currency,
		Method:              method,
		VerifoneOpaqueToken: t.VerifoneOpaqueToken,
		HMACSignature:       t.HMACSignature,
		ExpiresAt:           expiresAt,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}, nil
}

func parseMethod(m string) (payment.PaymentMethod, bool) {
	switch strings.ToLower(m) {
	case "credit_debit":
		return payment.PaymentMethod_PAYMENT_METHOD_CREDIT_DEBIT, true
	case "nfc_apple_pay":
		return payment.PaymentMethod_PAYMENT_METHOD_NFC_APPLE_PAY, true
	case "nfc_google_pay":
		return payment.PaymentMethod_PAYMENT_METHOD_NFC_GOOGLE_PAY, true
	case "qr_code":
		return payment.PaymentMethod_PAYMENT_METHOD_QR_CODE, true
	case "cash_recycler":
		return payment.PaymentMethod_PAYMENT_METHOD_CASH_RECYCLER, true
	default:
		return payment.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED, false
	}
}

func parseDomainMethod(m string) (domain.PaymentMethod, bool) {
	switch strings.ToLower(m) {
	case "credit_debit":
		return domain.MethodCreditDebit, true
	case "nfc_apple_pay":
		return domain.MethodNFCApplePay, true
	case "nfc_google_pay":
		return domain.MethodNFCGooglePay, true
	case "qr_code":
		return domain.MethodQRCode, true
	case "cash_recycler":
		return domain.MethodCashRecycler, true
	default:
		return "", false
	}
}

func protoToMap(r *payment.PaymentResult) fiber.Map {
	return fiber.Map{
		"payment_id":     r.PaymentId,
		"order_id":       r.OrderId,
		"status":         r.Status,
		"auth_code":      r.AuthCode,
		"verifone_token": r.VerifoneToken,
		"card_brand":     r.CardBrand,
		"card_last_four": r.CardLastFour,
		"decline_reason": r.DeclineReason,
		"receipt_text":   r.ReceiptText,
		"processed_at":   r.ProcessedAt,
	}
}

func grpcStatus(err error) (int, string) {
	if err == nil {
		return http.StatusOK, "OK"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "InvalidArgument"):
		return http.StatusBadRequest, "InvalidArgument"
	case strings.Contains(msg, "NotFound"):
		return http.StatusNotFound, "NotFound"
	case strings.Contains(msg, "AlreadyExists"):
		return http.StatusConflict, "AlreadyExists"
	case strings.Contains(msg, "FailedPrecondition"):
		return http.StatusConflict, "FailedPrecondition"
	default:
		return http.StatusInternalServerError, "Internal"
	}
}
