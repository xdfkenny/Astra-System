package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// httpTransport talks to the Verifone sidecar over HTTP/JSON.
type httpTransport struct {
	baseURL string
	client  *http.Client
}

func newHTTPTransport(baseURL string) *httpTransport {
	return &httpTransport{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type httpAuthorizeRequest struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	KioskID     string `json:"kiosk_id"`
	AmountCents int    `json:"amount_cents"`
	Currency    string `json:"currency"`
	Method      string `json:"method"`
}

type httpAuthorizeResponse struct {
	Status        string `json:"status"`
	VerifoneToken string `json:"verifone_token"`
	AuthCode      string `json:"auth_code"`
	DeclineReason string `json:"decline_reason"`
	CardBrand     string `json:"card_brand"`
	CardLastFour  string `json:"card_last_four"`
	ReceiptText   string `json:"receipt_text"`
}

func (t *httpTransport) Authorize(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error) {
	payload := httpAuthorizeRequest{
		PaymentID:   req.PaymentID,
		OrderID:     req.OrderID,
		KioskID:     req.KioskID,
		AmountCents: req.AmountCents,
		Currency:    req.Currency,
		Method:      req.Method,
	}
	resp, err := t.post(ctx, "/v1/authorize", payload)
	if err != nil {
		return nil, err
	}
	return &AuthorizeResponse{
		Status:        resp.Status,
		VerifoneToken: resp.VerifoneToken,
		AuthCode:      resp.AuthCode,
		DeclineReason: resp.DeclineReason,
		CardBrand:     resp.CardBrand,
		CardLastFour:  resp.CardLastFour,
		ReceiptText:   resp.ReceiptText,
	}, nil
}

func (t *httpTransport) Capture(ctx context.Context, paymentID, verifoneToken string) error {
	resp, err := t.post(ctx, "/v1/capture", map[string]string{
		"payment_id":     paymentID,
		"verifone_token": verifoneToken,
	})
	if err != nil {
		return err
	}
	if resp.Status != "APPROVED" && resp.Status != "CAPTURED" {
		return fmt.Errorf("verifone: capture declined: %s", resp.Status)
	}
	return nil
}

func (t *httpTransport) Settle(ctx context.Context, paymentID, verifoneToken string) error {
	resp, err := t.post(ctx, "/v1/settle", map[string]string{
		"payment_id":     paymentID,
		"verifone_token": verifoneToken,
	})
	if err != nil {
		return err
	}
	if resp.Status != "APPROVED" && resp.Status != "SETTLED" {
		return fmt.Errorf("verifone: settle declined: %s", resp.Status)
	}
	return nil
}

func (t *httpTransport) post(ctx context.Context, path string, payload any) (*httpAuthorizeResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("verifone http: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("verifone http: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verifone http: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("verifone http: read response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("verifone http: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var out httpAuthorizeResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("verifone http: unmarshal response: %w", err)
	}
	return &out, nil
}
