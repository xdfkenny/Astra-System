// Package fake provides an in-memory Store implementation for unit tests.
package fake

import (
	"context"
	"sync"
	"time"

	"github.com/astra-systems/astra-service/services/sync-service/internal/model"
	"github.com/google/uuid"
)

// Store is a thread-safe in-memory implementation of repository.Store.
type Store struct {
	mu          sync.RWMutex
	kiosks      map[string]*model.Kiosk
	events      map[uuid.UUID]model.SyncEvent
	heartbeats  map[string][]model.Heartbeat
	checkpoints map[string]time.Time
}

// NewStore creates an empty fake store.
func NewStore() *Store {
	return &Store{
		kiosks:      make(map[string]*model.Kiosk),
		events:      make(map[uuid.UUID]model.SyncEvent),
		heartbeats:  make(map[string][]model.Heartbeat),
		checkpoints: make(map[string]time.Time),
	}
}

// SeedKiosk inserts a kiosk into the fake store for test setup.
func (s *Store) SeedKiosk(k model.Kiosk) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kiosks[k.KioskID.String()] = &k
}

// GetKiosk returns a kiosk by ID.
func (s *Store) GetKiosk(ctx context.Context, kioskID string) (*model.Kiosk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.kiosks[kioskID]
	if !ok {
		return nil, model.ErrKioskNotFound
	}
	return k, nil
}

// InsertSyncEvents writes events idempotently.
func (s *Store) InsertSyncEvents(ctx context.Context, events []model.SyncEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range events {
		if _, exists := s.events[e.SyncEventID]; !exists {
			s.events[e.SyncEventID] = e
			key := e.StoreID.String() + "/" + e.KioskID.String()
			if e.CreatedAt.After(s.checkpoints[key]) {
				s.checkpoints[key] = e.CreatedAt
			}
		}
	}
	return nil
}

// GetDeltasSince returns events from the same store that are newer than since
// and were not created by the requesting kiosk.
func (s *Store) GetDeltasSince(ctx context.Context, storeID, kioskID string, since time.Time, limit int32) ([]model.SyncEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sid, err := uuid.Parse(storeID)
	if err != nil {
		return nil, err
	}
	kid, err := uuid.Parse(kioskID)
	if err != nil {
		return nil, err
	}

	var out []model.SyncEvent
	for _, e := range s.events {
		if e.StoreID != sid {
			continue
		}
		if e.KioskID == kid {
			continue
		}
		if e.CreatedAt.After(since) {
			out = append(out, e)
		}
	}

	// Stable sort by created_at then sync_event_id.
	less := func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].SyncEventID.String() < out[j].SyncEventID.String()
	}
	sortSlice(out, less)

	if limit > 0 && int(limit) < len(out) {
		out = out[:limit]
	}
	return out, nil
}

// UpsertHeartbeat records a heartbeat, collapsing entries within the same
// second to mimic the PostgreSQL unique constraint.
func (s *Store) UpsertHeartbeat(ctx context.Context, hb model.Heartbeat) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket := hb.AcknowledgedAt.Truncate(time.Second)
	list := s.heartbeats[hb.KioskID]
	found := false
	for i := range list {
		if list[i].AcknowledgedAt.Truncate(time.Second).Equal(bucket) {
			list[i] = hb
			found = true
			break
		}
	}
	if !found {
		list = append(list, hb)
	}
	s.heartbeats[hb.KioskID] = list
	return nil
}

// GetLatestHeartbeat returns the most recent heartbeat for a kiosk.
func (s *Store) GetLatestHeartbeat(ctx context.Context, kioskID string) (*model.Heartbeat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.heartbeats[kioskID]
	if len(list) == 0 {
		return nil, model.ErrHeartbeatNotFound
	}
	latest := &list[0]
	for i := range list {
		if list[i].AcknowledgedAt.After(latest.AcknowledgedAt) {
			latest = &list[i]
		}
	}
	return latest, nil
}

// GetLastCheckpoint returns the newest created_at for events from the kiosk.
func (s *Store) GetLastCheckpoint(ctx context.Context, storeID, kioskID string) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := storeID + "/" + kioskID
	return s.checkpoints[key], nil
}

// Events returns a snapshot of all stored sync events.
func (s *Store) Events() []model.SyncEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.SyncEvent, 0, len(s.events))
	for _, e := range s.events {
		out = append(out, e)
	}
	return out
}

// Heartbeats returns a snapshot of all stored heartbeats for a kiosk.
func (s *Store) Heartbeats(kioskID string) []model.Heartbeat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Heartbeat, len(s.heartbeats[kioskID]))
	copy(out, s.heartbeats[kioskID])
	return out
}

func sortSlice[T any](s []T, less func(i, j int) bool) {
	// Minimal bubble sort for test determinism; ok for small slices.
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if less(j, i) {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
