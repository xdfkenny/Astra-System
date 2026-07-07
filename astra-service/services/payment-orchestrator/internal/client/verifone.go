// Package client abstracts Verifone card-present operations. It prefers a gRPC
// connection to the Rust syncd verifone-ffi sidecar and falls back to HTTP when
// gRPC is unavailable or unconfigured.
package client

import (
	"context"
	"fmt"

	verifonepb "github.com/astra-systems/astra-service/services/payment-orchestrator/internal/client/verifonepb/verifone/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AuthorizeRequest describes a card-present authorization.
type AuthorizeRequest struct {
	PaymentID   string
	OrderID     string
	KioskID     string
	AmountCents int
	Currency    string
	Method      string
}

// AuthorizeResponse describes the result of a card-present authorization.
type AuthorizeResponse struct {
	Status        string
	VerifoneToken string
	AuthCode      string
	DeclineReason string
	CardBrand     string
	CardLastFour  string
	ReceiptText   string
}

// Gateway is the Verifone operations surface required by the orchestrator.
type Gateway interface {
	Authorize(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error)
	Capture(ctx context.Context, paymentID, verifoneToken string) error
	Settle(ctx context.Context, paymentID, verifoneToken string) error
}

// Client selects between gRPC and HTTP transports.
type Client struct {
	grpcAddr string
	httpURL  string
	gateway  Gateway
	closeFn  func() error
}

// New creates a Verifone client. If grpcAddr is non-empty, it dials the sidecar
// and uses gRPC; otherwise it falls back to the HTTP sidecar URL.
func New(grpcAddr, httpURL string) (*Client, error) {
	c := &Client{grpcAddr: grpcAddr, httpURL: httpURL}
	if grpcAddr != "" {
		conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("verifone: dial grpc: %w", err)
		}
		c.gateway = &grpcTransport{client: verifonepb.NewVerifoneFFIClient(conn)}
		c.closeFn = conn.Close
		return c, nil
	}
	c.gateway = newHTTPTransport(httpURL)
	c.closeFn = func() error { return nil }
	return c, nil
}

// Authorize delegates to the selected transport.
func (c *Client) Authorize(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error) {
	return c.gateway.Authorize(ctx, req)
}

// Capture delegates to the selected transport.
func (c *Client) Capture(ctx context.Context, paymentID, verifoneToken string) error {
	return c.gateway.Capture(ctx, paymentID, verifoneToken)
}

// Settle delegates to the selected transport.
func (c *Client) Settle(ctx context.Context, paymentID, verifoneToken string) error {
	return c.gateway.Settle(ctx, paymentID, verifoneToken)
}

// Close releases the underlying transport resources.
func (c *Client) Close() error {
	if c.closeFn != nil {
		return c.closeFn()
	}
	return nil
}
