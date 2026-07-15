// Package service implements the legacy POS adapter use cases: forwarding
// completed carts/orders to a legacy POS while storing the submission result
// in Astra.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	orderv1 "github.com/astra-systems/astra-service/proto/gen/go/order"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/client"
	"github.com/astra-systems/astra-service/services/legacy-pos-adapter/internal/repository"
	"github.com/google/uuid"
)

// AdapterService forwards completed orders to a legacy POS and persists the
// outcome. When no legacy POS URL is configured the service records submissions
// as skipped.
type AdapterService struct {
	repo      repository.Repository
	posClient posClient
	legacyURL string
	enabled   bool
}

// posClient is the minimal surface of the legacy POS client required by the
// service, enabling tests to inject a fake.
type posClient interface {
	BaseURL() string
	Submit(ctx context.Context, req client.LegacyPOSRequest) (*client.LegacyPOSResponse, error)
}

// New returns an AdapterService backed by the supplied repository and client.
// The client may be nil when the integration is disabled.
func New(repo repository.Repository, posClient posClient, enabled bool) *AdapterService {
	var legacyURL string
	if posClient != nil {
		legacyURL = posClient.BaseURL()
	}
	return &AdapterService{
		repo:      repo,
		posClient: posClient,
		legacyURL: legacyURL,
		enabled:   enabled,
	}
}

// HandleOrderCreated proxies a finalized order to the legacy POS when enabled
// and records the submission in Astra.
func (s *AdapterService) HandleOrderCreated(ctx context.Context, order *orderv1.Order) (*repository.Submission, error) {
	if order == nil {
		return nil, fmt.Errorf("order is required")
	}
	if order.OrderId == "" {
		return nil, fmt.Errorf("order_id is required")
	}

	if !s.enabled || s.posClient == nil {
		return s.recordSkipped(ctx, order)
	}

	req := s.buildPOSRequest(order)
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("service: marshal pos request: %w", err)
	}

	sentAt := time.Now().UTC()
	resp, err := s.posClient.Submit(ctx, req)

	submission := &repository.Submission{
		SubmissionID:   uuid.New().String(),
		OrderID:        order.OrderId,
		CartID:         order.CartId,
		StoreID:        order.StoreId,
		KioskID:        order.KioskId,
		LegacyPOSURL:   s.legacyURL,
		RequestPayload: reqBody,
		SentAt:         sentAt,
	}
	if err != nil {
		submission.Error = err.Error()
	} else {
		submission.ResponseBody = resp.Body
		submission.StatusCode = resp.StatusCode
		if !resp.Accepted {
			submission.Error = fmt.Sprintf("legacy pos rejected order: %d", resp.StatusCode)
			if resp.Error != "" {
				submission.Error = resp.Error
			}
		}
	}

	if err := s.repo.SaveSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("service: save submission: %w", err)
	}
	return submission, nil
}

func (s *AdapterService) recordSkipped(ctx context.Context, order *orderv1.Order) (*repository.Submission, error) {
	submission := &repository.Submission{
		SubmissionID: uuid.New().String(),
		OrderID:      order.OrderId,
		CartID:       order.CartId,
		StoreID:      order.StoreId,
		KioskID:      order.KioskId,
		LegacyPOSURL: "",
		Error:        "legacy pos integration disabled",
		SentAt:       time.Now().UTC(),
	}
	if err := s.repo.SaveSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("service: save skipped submission: %w", err)
	}
	return submission, nil
}

func (s *AdapterService) buildPOSRequest(order *orderv1.Order) client.LegacyPOSRequest {
	items := make([]client.POSItem, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, client.POSItem{
			ItemID:    item.ItemId,
			Name:      item.NameSnapshot,
			Quantity:  item.Quantity,
			UnitPrice: item.PriceCentsSnapshot,
			LineTotal: item.LineTotalCents,
		})
	}
	return client.LegacyPOSRequest{
		OrderID:   order.OrderId,
		CartID:    order.CartId,
		StoreID:   order.StoreId,
		KioskID:   order.KioskId,
		Total:     order.TotalCents,
		Currency:  "USD",
		Items:     items,
		Timestamp: time.Now().UTC(),
		Metadata: map[string]string{
			"source": "astra-legacy-pos-adapter",
		},
	}
}

// compile-time assertion that the service can be built with a nil client.
var _ = New(nil, nil, false)
