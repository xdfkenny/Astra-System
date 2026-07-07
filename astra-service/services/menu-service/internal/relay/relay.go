package relay

import (
	"database/sql"
	"fmt"

	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/outbox"
	"github.com/astra-systems/astra-service/services/menu-service/internal/publisher"
)

// SubjectResolver maps menu outbox event types to NATS JetStream subjects.
func SubjectResolver(eventType string) string {
	switch eventType {
	case publisher.EventTypeMenuUpdated:
		return publisher.SubjectMenuUpdated
	case publisher.EventTypeItemPriceChanged:
		return publisher.SubjectItemPriceChanged
	default:
		return fmt.Sprintf("astra.menu.%s", eventType)
	}
}

// New creates an outbox relay for the menu service.
func New(db *sql.DB, bus *eventbus.Bus) *outbox.Relay {
	return outbox.NewRelay(db, bus, SubjectResolver)
}
