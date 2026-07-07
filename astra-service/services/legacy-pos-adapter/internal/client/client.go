// Package client provides an HTTP client for the legacy POS system.
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

// LegacyPOSRequest is the payload forwarded to the legacy POS.
type LegacyPOSRequest struct {
	OrderID   string      `json:"order_id"`
	CartID    string      `json:"cart_id"`
	StoreID   string      `json:"store_id"`
	KioskID   string      `json:"kiosk_id"`
	Total     int64       `json:"total_cents"`
	Currency  string      `json:"currency"`
	Items     []POSItem   `json:"items"`
	Timestamp time.Time   `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// POSItem mirrors an order line for the legacy POS.
type POSItem struct {
	ItemID     string `json:"item_id"`
	Name       string `json:"name"`
	Quantity   int32  `json:"quantity"`
	UnitPrice  int64  `json:"unit_price_cents"`
	LineTotal  int64  `json:"line_total_cents"`
}

// LegacyPOSResponse captures the legacy POS reply.
type LegacyPOSResponse struct {
	StatusCode int
	Body       []byte
	POSOrderID string
	Accepted   bool
	Error      string
}

// Client forwards completed orders to a legacy POS endpoint.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New returns a Client bound to the supplied legacy POS URL and credentials.
func New(baseURL, apiKey string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// BaseURL returns the configured legacy POS endpoint.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Submit sends a completed order to the legacy POS and returns the response.
func (c *Client) Submit(ctx context.Context, req LegacyPOSRequest) (*LegacyPOSResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("client: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/orders", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("client: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("client: submit request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("client: read response: %w", err)
	}

	result := &LegacyPOSResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Accepted:   resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	var parsed struct {
		POSOrderID string `json:"pos_order_id"`
		Accepted   bool   `json:"accepted"`
		Error      string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &parsed); err == nil {
		result.POSOrderID = parsed.POSOrderID
		if parsed.Accepted {
			result.Accepted = true
		}
		if parsed.Error != "" {
			result.Error = parsed.Error
		}
	}

	return result, nil
}

// compile-time interface assertion used by tests.
type submitter interface {
	Submit(ctx context.Context, req LegacyPOSRequest) (*LegacyPOSResponse, error)
}

var _ submitter = (*Client)(nil)
