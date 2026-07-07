package client

import (
	"context"
	"fmt"

	verifonepb "github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client/verifonepb/verifone/v1"
)

// grpcTransport talks to the Rust syncd verifone-ffi sidecar over gRPC.
type grpcTransport struct {
	client verifonepb.VerifoneFFIClient
}

func (t *grpcTransport) Authorize(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error) {
	resp, err := t.client.Authorize(ctx, &verifonepb.AuthorizeRequest{
		PaymentId:   req.PaymentID,
		OrderId:     req.OrderID,
		KioskId:     req.KioskID,
		AmountCents: int64(req.AmountCents),
		Currency:    req.Currency,
		Method:      req.Method,
	})
	if err != nil {
		return nil, fmt.Errorf("verifone grpc: authorize: %w", err)
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

func (t *grpcTransport) Capture(ctx context.Context, paymentID, verifoneToken string) error {
	resp, err := t.client.Capture(ctx, &verifonepb.CaptureRequest{
		PaymentId:     paymentID,
		VerifoneToken: verifoneToken,
	})
	if err != nil {
		return fmt.Errorf("verifone grpc: capture: %w", err)
	}
	if resp.Status != "APPROVED" && resp.Status != "CAPTURED" {
		return fmt.Errorf("verifone grpc: capture declined: %s", resp.Status)
	}
	return nil
}

func (t *grpcTransport) Settle(ctx context.Context, paymentID, verifoneToken string) error {
	resp, err := t.client.Settle(ctx, &verifonepb.SettleRequest{
		PaymentId:     paymentID,
		VerifoneToken: verifoneToken,
	})
	if err != nil {
		return fmt.Errorf("verifone grpc: settle: %w", err)
	}
	if resp.Status != "APPROVED" && resp.Status != "SETTLED" {
		return fmt.Errorf("verifone grpc: settle declined: %s", resp.Status)
	}
	return nil
}
