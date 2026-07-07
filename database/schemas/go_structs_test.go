package schemas

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEnumValues(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"TenantPlanStandard", string(TenantPlanStandard), "standard"},
		{"TenantPlanEnterprise", string(TenantPlanEnterprise), "enterprise"},
		{"KioskSyncStatusOnline", string(KioskSyncStatusOnline), "online"},
		{"CartStatusActive", string(CartStatusActive), "active"},
		{"OrderStatusPaid", string(OrderStatusPaid), "paid"},
		{"PaymentMethodQRCode", string(PaymentMethodQRCode), "qr_code"},
		{"PaymentStatusDeclined", string(PaymentStatusDeclined), "declined"},
		{"EmployeeRoleManager", string(EmployeeRoleManager), "manager"},
		{"AuditEventTypeOrderCreated", string(AuditEventTypeOrderCreated), "order_created"},
		{"InventoryTransactionTypeWaste", string(InventoryTransactionTypeWaste), "waste"},
		{"SyncEventTypeCartMerge", string(SyncEventTypeCartMerge), "cart_merge"},
		{"RefundStatusCompleted", string(RefundStatusCompleted), "completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, tt.got)
			}
		})
	}
}

func TestStructJSONTags(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	now := time.Now().UTC()

	store := Store{
		StoreID:   id,
		Name:      "Astra Miami Brickell",
		Timezone:  "America/New_York",
		Currency:  "USD",
		TaxRate:   0.07,
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("marshal store: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal store: %v", err)
	}

	if decoded["store_id"] != id.String() {
		t.Fatalf("expected store_id %q, got %v", id.String(), decoded["store_id"])
	}
	if decoded["name"] != "Astra Miami Brickell" {
		t.Fatalf("unexpected name: %v", decoded["name"])
	}
}

func TestUUIDAndTimeFields(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC()

	kiosk := Kiosk{
		KioskID:        id,
		StoreID:        uuid.New(),
		HardwareID:     "HW-TEST-001",
		DisplayName:    "Lane 1",
		SyncStatus:     KioskSyncStatusOnline,
		IsLeader:       true,
		SigningKeyHash: "deadbeef",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if kiosk.KioskID != id {
		t.Fatalf("expected kiosk ID %v, got %v", id, kiosk.KioskID)
	}
	if !kiosk.CreatedAt.Equal(now) {
		t.Fatalf("expected created at %v, got %v", now, kiosk.CreatedAt)
	}
	if kiosk.SyncStatus != KioskSyncStatusOnline {
		t.Fatalf("unexpected sync status: %v", kiosk.SyncStatus)
	}
}

func TestNullableFields(t *testing.T) {
	phone := "+15551234567"
	cart := Cart{
		CartID:      uuid.New(),
		StoreID:     uuid.New(),
		KioskID:     uuid.New(),
		SessionID:   uuid.New(),
		CustomerPhone: &phone,
		Status:      CartStatusActive,
		ExpiresAt:   time.Now().UTC().Add(10 * time.Minute),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		CreatedAtMs: time.Now().UnixMilli(),
		UpdatedAtMs: time.Now().UnixMilli(),
	}

	if cart.CustomerPhone == nil || *cart.CustomerPhone != phone {
		t.Fatalf("unexpected customer phone: %v", cart.CustomerPhone)
	}

	data, err := json.Marshal(cart)
	if err != nil {
		t.Fatalf("marshal cart: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal cart: %v", err)
	}

	if decoded["customer_phone"] != phone {
		t.Fatalf("expected customer_phone %q, got %v", phone, decoded["customer_phone"])
	}
}
