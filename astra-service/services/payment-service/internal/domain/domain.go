// Package domain contains the payment aggregate and offline-token models.
package domain

import "time"

// Payment represents a single payment attempt/authorization. It stores only
// PCI-safe metadata: no PAN, no track data, no CVV. Card-present data is
// confined to the Verifone terminal and its opaque token.
type Payment struct {
	PaymentID        string
	OrderID          string
	KioskID          string
	IdempotencyKey   string
	AmountCents      int
	Currency         string
	Method           string
	Status           string
	VerifoneToken    string
	VerifoneAuthCode string
	CardBrand        string
	CardLastFour     string
	DeclineReason    string
	ReceiptText      *string
	IsOfflineToken   bool
	OfflineTokenHMAC string
	SyncedAt         *time.Time
	CreatedAt        time.Time
}

// OfflineToken is a kiosk-signed record of a payment authorized locally while
// the store was offline. The cloud payment orchestrator verifies the HMAC and
// replays the authorization to the processor for settlement.
type OfflineToken struct {
	TokenID             string
	StoreID             string
	KioskID             string
	CartID              string
	AmountCents         int
	Currency            string
	Method              string
	VerifoneOpaqueToken string
	HMACSignature       string
	ExpiresAt           time.Time
	CreatedAt           time.Time
}

// SettlementResult records the outcome of replaying an offline token to the
// payment processor.
type SettlementResult struct {
	SettledAt     time.Time `json:"settledAt"`
	Success       bool      `json:"success"`
	ProcessorID   string    `json:"processorId,omitempty"`
	DeclineReason string    `json:"declineReason,omitempty"`
}
