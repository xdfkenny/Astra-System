// Package service implements the cloud-side SyncService gRPC handlers.
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/astra-systems/astra-service/proto/gen/go/sync"
	"github.com/astra-systems/astra-service/services/sync-service/internal/auth"
	"github.com/astra-systems/astra-service/services/sync-service/internal/eventbus"
	"github.com/astra-systems/astra-service/services/sync-service/internal/model"
	"github.com/astra-systems/astra-service/services/sync-service/internal/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Sync implements the astra.sync.v1.SyncService server.
type Sync struct {
	sync.UnimplementedSyncServiceServer
	store     repository.Store
	publisher eventbus.Publisher
	now       func() time.Time
}

// NewSync creates a Sync service with the supplied dependencies.
func NewSync(store repository.Store, publisher eventbus.Publisher) *Sync {
	return &Sync{
		store:     store,
		publisher: publisher,
		now:       time.Now,
	}
}

// UploadBatch authenticates the kiosk leader, ingests the supplied deltas into
// PostgreSQL, and publishes a durable NATS notification for downstream
// consumers. Each delta is stored as a raw sync event for later CRDT conflict
// resolution.
func (s *Sync) UploadBatch(ctx context.Context, req *sync.UploadBatchRequest) (*sync.SyncAck, error) {
	if req.Batch == nil {
		return nil, status.Errorf(codes.InvalidArgument, "batch is required")
	}
	b := req.Batch

	if b.KioskId == "" || b.StoreId == "" || b.BatchId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "batch_id, store_id, and kiosk_id are required")
	}

	kioskID, ok := ctx.Value(auth.KioskIDKey{}).(string)
	if ok && kioskID != "" && kioskID != b.KioskId {
		return nil, status.Errorf(codes.PermissionDenied, "kiosk_id in batch does not match authenticated kiosk")
	}

	events, err := s.deltasToEvents(b)
	if err != nil {
		return nil, err
	}

	if err := s.store.InsertSyncEvents(ctx, events); err != nil {
		return nil, status.Errorf(codes.Internal, "store sync events: %v", err)
	}

	if s.publisher != nil {
		if err := s.publisher.PublishBatchIngested(ctx, b.StoreId, b.KioskId, len(b.Deltas)); err != nil {
			return nil, status.Errorf(codes.Internal, "publish batch ingested: %v", err)
		}
	}

	return &sync.SyncAck{
		BatchId:      b.BatchId,
		KioskId:      b.KioskId,
		Accepted:     true,
		ProcessedAt:  s.now().UTC().Format(time.RFC3339Nano),
		ErrorMessage: "",
	}, nil
}

// DownloadBatch computes the delta set for a kiosk since its last sync
// checkpoint and returns them as a SyncBatch. The caller's own deltas are
// excluded from the result.
func (s *Sync) DownloadBatch(ctx context.Context, req *sync.DownloadBatchRequest) (*sync.SyncBatch, error) {
	if req.StoreId == "" || req.KioskId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "store_id and kiosk_id are required")
	}

	var since time.Time
	if req.Since != "" {
		var err error
		since, err = time.Parse(time.RFC3339Nano, req.Since)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid since timestamp: %v", err)
		}
	} else {
		checkpoint, err := s.store.GetLastCheckpoint(ctx, req.StoreId, req.KioskId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load checkpoint: %v", err)
		}
		since = checkpoint
	}

	deltas, err := s.store.GetDeltasSince(ctx, req.StoreId, req.KioskId, since, req.MaxDeltas)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "compute deltas: %v", err)
	}

	return &sync.SyncBatch{
		BatchId: uuid.Must(uuid.NewV7()).String(),
		StoreId: req.StoreId,
		KioskId: req.KioskId,
		Deltas:  eventsToDeltas(deltas),
		SentAt:  s.now().UTC().Format(time.RFC3339Nano),
	}, nil
}

// Heartbeat records a kiosk heartbeat, deduplicating rapid successive calls,
// and returns the current leadership state.
func (s *Sync) Heartbeat(ctx context.Context, req *sync.HeartbeatRequest) (*sync.HeartbeatResponse, error) {
	if req.KioskId == "" || req.StoreId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "kiosk_id and store_id are required")
	}

	hb := model.Heartbeat{
		KioskID:        req.KioskId,
		StoreID:        req.StoreId,
		Status:         req.Status,
		VectorClock:    req.VectorClock,
		AcknowledgedAt: s.now().UTC(),
	}

	if err := s.store.UpsertHeartbeat(ctx, hb); err != nil {
		return nil, status.Errorf(codes.Internal, "record heartbeat: %v", err)
	}

	kiosk, err := s.store.GetKiosk(ctx, req.KioskId)
	if err != nil {
		if errors.Is(err, model.ErrKioskNotFound) {
			return nil, status.Errorf(codes.Unauthenticated, "kiosk not found")
		}
		return nil, status.Errorf(codes.Internal, "lookup kiosk: %v", err)
	}

	return &sync.HeartbeatResponse{
		KioskId:         req.KioskId,
		AcknowledgedAt:  hb.AcknowledgedAt.Format(time.RFC3339Nano),
		IsLeader:        kiosk.IsLeader,
	}, nil
}

func (s *Sync) deltasToEvents(b *sync.SyncBatch) ([]model.SyncEvent, error) {
	sid, err := uuid.Parse(b.StoreId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid store_id: %v", err)
	}
	kid, err := uuid.Parse(b.KioskId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid kiosk_id: %v", err)
	}

	events := make([]model.SyncEvent, 0, len(b.Deltas))
	for _, d := range b.Deltas {
		if d.DeltaId == "" {
			return nil, status.Errorf(codes.InvalidArgument, "delta_id is required")
		}
		did, err := uuid.Parse(d.DeltaId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid delta_id %s: %v", d.DeltaId, err)
		}

		eventType := syncEventTypeToString(d.EventType)
		payload := map[string]any{
			"delta_id":  d.DeltaId,
			"event_type": eventType,
		}
		if len(d.Payload) > 0 {
			var raw any
			if err := json.Unmarshal(d.Payload, &raw); err != nil {
				payload["raw"] = d.Payload
			} else {
				payload["payload"] = raw
			}
		}
		if d.SourceKioskId != "" {
			payload["source_kiosk_id"] = d.SourceKioskId
		}
		if d.CreatedAt != "" {
			payload["created_at"] = d.CreatedAt
		}

		createdAt := s.now().UTC()
		if d.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, d.CreatedAt); err == nil {
				createdAt = t.UTC()
			}
		}

		events = append(events, model.SyncEvent{
			SyncEventID: did,
			StoreID:     sid,
			KioskID:     kid,
			EventType:   eventType,
			PayloadJSON: payload,
			VectorClock: d.VectorClock,
			CreatedAt:   createdAt,
		})
	}
	return events, nil
}

func eventsToDeltas(events []model.SyncEvent) []*sync.SyncDelta {
	out := make([]*sync.SyncDelta, len(events))
	for i, e := range events {
		var payload []byte
		if e.PayloadJSON != nil {
			payload, _ = json.Marshal(e.PayloadJSON)
		}
		out[i] = &sync.SyncDelta{
			DeltaId:       e.SyncEventID.String(),
			EventType:     stringToSyncEventType(e.EventType),
			Payload:       payload,
			VectorClock:   e.VectorClock,
			SourceKioskId: e.KioskID.String(),
			CreatedAt:     e.CreatedAt.Format(time.RFC3339Nano),
		}
	}
	return out
}

func syncEventTypeToString(t sync.SyncEventType) string {
	switch t {
	case sync.SyncEventType_SYNC_EVENT_TYPE_INVENTORY_UPDATE:
		return "inventory_update"
	case sync.SyncEventType_SYNC_EVENT_TYPE_CART_MERGE:
		return "cart_merge"
	case sync.SyncEventType_SYNC_EVENT_TYPE_TRANSACTION_BATCH:
		return "transaction_batch"
	case sync.SyncEventType_SYNC_EVENT_TYPE_ANALYTICS_BATCH:
		return "analytics_batch"
	default:
		return "unspecified"
	}
}

func stringToSyncEventType(s string) sync.SyncEventType {
	switch s {
	case "inventory_update":
		return sync.SyncEventType_SYNC_EVENT_TYPE_INVENTORY_UPDATE
	case "cart_merge":
		return sync.SyncEventType_SYNC_EVENT_TYPE_CART_MERGE
	case "transaction_batch":
		return sync.SyncEventType_SYNC_EVENT_TYPE_TRANSACTION_BATCH
	case "analytics_batch":
		return sync.SyncEventType_SYNC_EVENT_TYPE_ANALYTICS_BATCH
	default:
		return sync.SyncEventType_SYNC_EVENT_TYPE_UNSPECIFIED
	}
}

var _ = sql.NullTime{} // ensure database/sql stays available if future queries need it
