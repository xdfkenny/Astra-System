package service

import (
	"context"
	"testing"
	"time"

	"github.com/astra-systems/astra-service/proto/gen/go/sync"
	"github.com/astra-systems/astra-service/services/sync-service/internal/model"
	"github.com/astra-systems/astra-service/services/sync-service/internal/repository/fake"
	"github.com/google/uuid"
)

type fakePublisher struct {
	calls []BatchIngestedCall
}

type BatchIngestedCall struct {
	StoreID    string
	KioskID    string
	DeltaCount int
}

func (f *fakePublisher) PublishBatchIngested(ctx context.Context, storeID, kioskID string, deltaCount int) error {
	f.calls = append(f.calls, BatchIngestedCall{StoreID: storeID, KioskID: kioskID, DeltaCount: deltaCount})
	return nil
}

func TestUploadBatch_IngestsAndPublishes(t *testing.T) {
	store := fake.NewStore()
	pub := &fakePublisher{}
	svc := NewSync(store, pub)
	svc.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

	storeID := uuid.Must(uuid.NewV7())
	kioskID := uuid.Must(uuid.NewV7())
	deltaID := uuid.Must(uuid.NewV7())

	req := &sync.UploadBatchRequest{
		Batch: &sync.SyncBatch{
			BatchId: uuid.Must(uuid.NewV7()).String(),
			StoreId: storeID.String(),
			KioskId: kioskID.String(),
			Deltas: []*sync.SyncDelta{
				{
					DeltaId:       deltaID.String(),
					EventType:     sync.SyncEventType_SYNC_EVENT_TYPE_INVENTORY_UPDATE,
					Payload:       []byte(`{"sku":"coffee","qty":5}`),
					VectorClock:   map[string]int64{"a": 1},
					SourceKioskId: kioskID.String(),
					CreatedAt:     "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	ack, err := svc.UploadBatch(context.Background(), req)
	if err != nil {
		t.Fatalf("UploadBatch: %v", err)
	}
	if !ack.Accepted {
		t.Fatal("expected batch to be accepted")
	}
	if ack.BatchId != req.Batch.BatchId {
		t.Fatalf("expected ack batch_id %q, got %q", req.Batch.BatchId, ack.BatchId)
	}

	events := store.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(events))
	}
	if events[0].SyncEventID != deltaID {
		t.Fatalf("expected event id %q, got %q", deltaID, events[0].SyncEventID)
	}
	if events[0].EventType != "inventory_update" {
		t.Fatalf("expected event type inventory_update, got %q", events[0].EventType)
	}

	if len(pub.calls) != 1 {
		t.Fatalf("expected 1 publish call, got %d", len(pub.calls))
	}
	if pub.calls[0].DeltaCount != 1 {
		t.Fatalf("expected delta_count 1, got %d", pub.calls[0].DeltaCount)
	}
}

func TestUploadBatch_Idempotent(t *testing.T) {
	store := fake.NewStore()
	pub := &fakePublisher{}
	svc := NewSync(store, pub)

	storeID := uuid.Must(uuid.NewV7())
	kioskID := uuid.Must(uuid.NewV7())
	deltaID := uuid.Must(uuid.NewV7())

	req := &sync.UploadBatchRequest{
		Batch: &sync.SyncBatch{
			BatchId: uuid.Must(uuid.NewV7()).String(),
			StoreId: storeID.String(),
			KioskId: kioskID.String(),
			Deltas: []*sync.SyncDelta{
				{
					DeltaId:     deltaID.String(),
					EventType:   sync.SyncEventType_SYNC_EVENT_TYPE_CART_MERGE,
					VectorClock: map[string]int64{"a": 1},
				},
			},
		},
	}

	if _, err := svc.UploadBatch(context.Background(), req); err != nil {
		t.Fatalf("first UploadBatch: %v", err)
	}
	if _, err := svc.UploadBatch(context.Background(), req); err != nil {
		t.Fatalf("second UploadBatch: %v", err)
	}

	if len(store.Events()) != 1 {
		t.Fatalf("expected 1 stored event after duplicate upload, got %d", len(store.Events()))
	}
}

func TestDownloadBatch_DeltaCalculation(t *testing.T) {
	store := fake.NewStore()
	svc := NewSync(store, nil)

	storeID := uuid.Must(uuid.NewV7())
	kioskA := uuid.Must(uuid.NewV7())
	kioskB := uuid.Must(uuid.NewV7())
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_ = store.InsertSyncEvents(context.Background(), []model.SyncEvent{
		{
			SyncEventID: uuid.Must(uuid.NewV7()),
			StoreID:     storeID,
			KioskID:     kioskA,
			EventType:   "inventory_update",
			PayloadJSON: map[string]any{"old": true},
			CreatedAt:   base.Add(-1 * time.Hour),
		},
		{
			SyncEventID: uuid.Must(uuid.NewV7()),
			StoreID:     storeID,
			KioskID:     kioskA,
			EventType:   "inventory_update",
			PayloadJSON: map[string]any{"new": true},
			CreatedAt:   base.Add(time.Hour),
		},
		{
			SyncEventID: uuid.Must(uuid.NewV7()),
			StoreID:     storeID,
			KioskID:     kioskB,
			EventType:   "cart_merge",
			PayloadJSON: map[string]any{"cart": "b"},
			CreatedAt:   base.Add(2 * time.Hour),
		},
	})

	req := &sync.DownloadBatchRequest{
		StoreId:    storeID.String(),
		KioskId:    kioskA.String(),
		Since:      base.Format(time.RFC3339Nano),
		MaxDeltas:  10,
	}

	batch, err := svc.DownloadBatch(context.Background(), req)
	if err != nil {
		t.Fatalf("DownloadBatch: %v", err)
	}
	if len(batch.Deltas) != 1 {
		t.Fatalf("expected 1 delta, got %d", len(batch.Deltas))
	}
	if batch.Deltas[0].SourceKioskId != kioskB.String() {
		t.Fatalf("expected delta from kioskB, got %q", batch.Deltas[0].SourceKioskId)
	}
}

func TestDownloadBatch_ExcludesOwnDeltas(t *testing.T) {
	store := fake.NewStore()
	svc := NewSync(store, nil)

	storeID := uuid.Must(uuid.NewV7())
	kioskID := uuid.Must(uuid.NewV7())
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_ = store.InsertSyncEvents(context.Background(), []model.SyncEvent{
		{
			SyncEventID: uuid.Must(uuid.NewV7()),
			StoreID:     storeID,
			KioskID:     kioskID,
			EventType:   "inventory_update",
			CreatedAt:   base.Add(time.Hour),
		},
	})

	req := &sync.DownloadBatchRequest{
		StoreId: storeID.String(),
		KioskId: kioskID.String(),
		Since:   base.Format(time.RFC3339Nano),
	}

	batch, err := svc.DownloadBatch(context.Background(), req)
	if err != nil {
		t.Fatalf("DownloadBatch: %v", err)
	}
	if len(batch.Deltas) != 0 {
		t.Fatalf("expected 0 own deltas, got %d", len(batch.Deltas))
	}
}

func TestHeartbeat_Dedup(t *testing.T) {
	store := fake.NewStore()
	svc := NewSync(store, nil)

	storeID := uuid.Must(uuid.NewV7())
	kioskID := uuid.Must(uuid.NewV7())
	store.SeedKiosk(model.Kiosk{
		KioskID:        kioskID,
		StoreID:        storeID,
		SigningKeyHash: "key",
		IsLeader:       true,
	})

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	req := &sync.HeartbeatRequest{
		KioskId:     kioskID.String(),
		StoreId:     storeID.String(),
		Status:      "online",
		VectorClock: map[string]int64{"a": 1},
	}

	if _, err := svc.Heartbeat(context.Background(), req); err != nil {
		t.Fatalf("first Heartbeat: %v", err)
	}
	if _, err := svc.Heartbeat(context.Background(), req); err != nil {
		t.Fatalf("second Heartbeat: %v", err)
	}

	hbs := store.Heartbeats(kioskID.String())
	if len(hbs) != 1 {
		t.Fatalf("expected 1 heartbeat after dedup, got %d", len(hbs))
	}
}

func TestHeartbeat_UpdatesStatus(t *testing.T) {
	store := fake.NewStore()
	svc := NewSync(store, nil)

	storeID := uuid.Must(uuid.NewV7())
	kioskID := uuid.Must(uuid.NewV7())
	store.SeedKiosk(model.Kiosk{
		KioskID:        kioskID,
		StoreID:        storeID,
		SigningKeyHash: "key",
		IsLeader:       true,
	})

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return base }

	_, err := svc.Heartbeat(context.Background(), &sync.HeartbeatRequest{
		KioskId: kioskID.String(),
		StoreId: storeID.String(),
		Status:  "online",
	})
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	svc.now = func() time.Time { return base.Add(2 * time.Second) }
	resp, err := svc.Heartbeat(context.Background(), &sync.HeartbeatRequest{
		KioskId: kioskID.String(),
		StoreId: storeID.String(),
		Status:  "degraded",
	})
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if resp.IsLeader != true {
		t.Fatalf("expected is_leader true, got %v", resp.IsLeader)
	}

	hbs := store.Heartbeats(kioskID.String())
	if len(hbs) != 2 {
		t.Fatalf("expected 2 heartbeats across different seconds, got %d", len(hbs))
	}
	latest, _ := store.GetLatestHeartbeat(context.Background(), kioskID.String())
	if latest.Status != "degraded" {
		t.Fatalf("expected latest status degraded, got %q", latest.Status)
	}
}
