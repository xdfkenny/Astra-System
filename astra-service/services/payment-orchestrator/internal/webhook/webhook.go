// Package webhook parses and verifies asynchronous notifications from the
// Verifone sidecar.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/astra-systems/astra-service/services/payment-orchestrator/internal/domain"
	"github.com/google/uuid"
)

// Notification is the async notification shape from the Verifone sidecar.
type Notification struct {
	PaymentID     string `json:"payment_id"`
	Status        string `json:"status"`
	DeclineReason string `json:"decline_reason,omitempty"`
}

// ParseNotification unmarshals and validates a webhook payload.
func ParseNotification(body []byte) (*Notification, error) {
	var n Notification
	if err := json.Unmarshal(body, &n); err != nil {
		return nil, fmt.Errorf("webhook: unmarshal: %w", err)
	}
	if n.PaymentID == "" {
		return nil, fmt.Errorf("webhook: missing payment_id")
	}
	if _, err := uuid.Parse(n.PaymentID); err != nil {
		return nil, fmt.Errorf("webhook: invalid payment_id: %w", err)
	}
	if n.Status == "" {
		return nil, fmt.Errorf("webhook: missing status")
	}
	return &n, nil
}

// VerifySignature checks the HMAC-SHA256 signature of a webhook body.
func VerifySignature(body []byte, signature string, secret []byte) error {
	if signature == "" {
		return fmt.Errorf("webhook: missing signature")
	}
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(body); err != nil {
		return fmt.Errorf("webhook: compute hmac: %w", err)
	}
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("webhook: signature mismatch")
	}
	return nil
}

// MapStatus converts a Verifone webhook status to the domain status machine.
func MapStatus(status string) (domain.PaymentStatus, error) {
	switch strings.ToUpper(status) {
	case "CAPTURED":
		return domain.StatusCaptured, nil
	case "SETTLED":
		return domain.StatusSettled, nil
	case "FAILED", "DECLINED":
		return domain.StatusFailed, nil
	default:
		return "", fmt.Errorf("webhook: unknown status %q", status)
	}
}
