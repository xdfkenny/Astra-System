package publisher

// NATS JetStream subjects for menu domain events.
const (
	SubjectMenuUpdated       = "astra.menu.updated.v1"
	SubjectItemPriceChanged  = "astra.menu.item.price_changed.v1"
	StreamName               = "ASTRA_MENU"
)

// Event types written to the outbox_events table.
const (
	EventTypeMenuUpdated      = "MenuUpdated"
	EventTypeItemPriceChanged = "ItemPriceChanged"
)
