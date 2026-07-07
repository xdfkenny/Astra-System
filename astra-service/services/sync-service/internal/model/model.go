// Package model contains the domain types shared by the sync-service layers.
package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrKioskNotFound     = errors.New("model: kiosk not found")
	ErrHeartbeatNotFound = errors.New("model: heartbeat not found")
)

// Kiosk mirrors the kiosks table schema.
type Kiosk struct {
	KioskID        uuid.UUID
	StoreID        uuid.UUID
	HardwareID     string
	DisplayName    string
	SigningKeyHash string
	IsLeader       bool
	SyncStatus     string
	LastSeenAt     *time.Time
	CreatedAt      time.Time
}

// SyncEvent mirrors the sync_events table schema.
type SyncEvent struct {
	SyncEventID uuid.UUID
	StoreID     uuid.UUID
	KioskID     uuid.UUID
	EventType   string
	PayloadJSON map[string]any
	VectorClock map[string]int64
	ProcessedAt *time.Time
	CreatedAt   time.Time
}

// Heartbeat mirrors the sync_heartbeats table schema.
type Heartbeat struct {
	KioskID        string
	StoreID        string
	Status         string
	VectorClock    map[string]int64
	AcknowledgedAt time.Time
}
