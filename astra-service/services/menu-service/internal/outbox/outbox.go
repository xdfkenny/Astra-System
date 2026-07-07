package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	commonoutbox "github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/menu-service/internal/publisher"
	"github.com/google/uuid"
)

// ItemPriceChangedPayload is the schema for item price change events.
type ItemPriceChangedPayload struct {
	ItemID          string `json:"item_id"`
	StoreID         string `json:"store_id"`
	PreviousPriceCents int64 `json:"previous_price_cents"`
	NewPriceCents   int64  `json:"new_price_cents"`
	OccurredAtMs    int64  `json:"occurred_at_ms"`
}

// MenuUpdatedPayload is the schema for menu update events.
type MenuUpdatedPayload struct {
	StoreID      string `json:"store_id"`
	EntityType   string `json:"entity_type"`
	EntityID     string `json:"entity_id"`
	OccurredAtMs int64  `json:"occurred_at_ms"`
}

// InsertItemPriceChanged writes an ItemPriceChanged event inside the supplied transaction.
func InsertItemPriceChanged(ctx context.Context, tx *sql.Tx, itemID, storeID uuid.UUID, previousPriceCents, newPriceCents int64) error {
	payload := ItemPriceChangedPayload{
		ItemID:             itemID.String(),
		StoreID:            storeID.String(),
		PreviousPriceCents: previousPriceCents,
		NewPriceCents:      newPriceCents,
		OccurredAtMs:       time.Now().UnixMilli(),
	}
	return insert(ctx, tx, publisher.EventTypeItemPriceChanged, "item", itemID, payload)
}

// InsertMenuUpdated writes a MenuUpdated event inside the supplied transaction.
func InsertMenuUpdated(ctx context.Context, tx *sql.Tx, entityType string, entityID, storeID uuid.UUID) error {
	payload := MenuUpdatedPayload{
		StoreID:      storeID.String(),
		EntityType:   entityType,
		EntityID:     entityID.String(),
		OccurredAtMs: time.Now().UnixMilli(),
	}
	return insert(ctx, tx, publisher.EventTypeMenuUpdated, entityType, entityID, payload)
}

func insert(ctx context.Context, tx *sql.Tx, eventType, aggregateType string, aggregateID uuid.UUID, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox: marshal %s payload: %w", eventType, err)
	}
	eventID, err := uuid.NewV7()
	if err != nil {
		eventID = uuid.New()
	}
	entry := commonoutbox.Entry{
		EventID:       eventID.String(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID.String(),
		EventType:     eventType,
		Payload:       data,
		OccurredAtMs:  time.Now().UnixMilli(),
	}
	if err := commonoutbox.InsertWithinTx(ctx, tx, entry); err != nil {
		return fmt.Errorf("outbox: insert %s: %w", eventType, err)
	}
	return nil
}
