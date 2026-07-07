package schemas

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

type TenantPlan string

const (
	TenantPlanStandard  TenantPlan = "standard"
	TenantPlanEnterprise TenantPlan = "enterprise"
)

type KioskSyncStatus string

const (
	KioskSyncStatusOnline      KioskSyncStatus = "online"
	KioskSyncStatusOffline     KioskSyncStatus = "offline"
	KioskSyncStatusDegraded    KioskSyncStatus = "degraded"
	KioskSyncStatusMaintenance KioskSyncStatus = "maintenance"
)

type ItemTaxCategory string

const (
	ItemTaxCategoryStandard ItemTaxCategory = "standard"
	ItemTaxCategoryExempt   ItemTaxCategory = "exempt"
	ItemTaxCategoryReduced  ItemTaxCategory = "reduced"
)

type WeightUnit string

const (
	WeightUnitGram     WeightUnit = "g"
	WeightUnitKilogram WeightUnit = "kg"
	WeightUnitPound    WeightUnit = "lb"
	WeightUnitOunce    WeightUnit = "oz"
)

type CartStatus string

const (
	CartStatusActive    CartStatus = "active"
	CartStatusFinalized CartStatus = "finalized"
	CartStatusAbandoned CartStatus = "abandoned"
	CartStatusExpired   CartStatus = "expired"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusFulfilled OrderStatus = "fulfilled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRefunded  OrderStatus = "refunded"
)

type PaymentMethod string

const (
	PaymentMethodCreditDebit   PaymentMethod = "credit_debit"
	PaymentMethodNFCApplePay   PaymentMethod = "nfc_apple_pay"
	PaymentMethodNFCGooglePay  PaymentMethod = "nfc_google_pay"
	PaymentMethodQRCode        PaymentMethod = "qr_code"
	PaymentMethodCashRecycler  PaymentMethod = "cash_recycler"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusAuthorized PaymentStatus = "authorized"
	PaymentStatusCaptured   PaymentStatus = "captured"
	PaymentStatusDeclined   PaymentStatus = "declined"
	PaymentStatusVoided     PaymentStatus = "voided"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

type EmployeeRole string

const (
	EmployeeRoleCashier    EmployeeRole = "cashier"
	EmployeeRoleSupervisor EmployeeRole = "supervisor"
	EmployeeRoleManager    EmployeeRole = "manager"
	EmployeeRoleAdmin      EmployeeRole = "admin"
)

type AuditEventType string

const (
	AuditEventTypeOrderCreated      AuditEventType = "order_created"
	AuditEventTypeOrderPaid         AuditEventType = "order_paid"
	AuditEventTypeOrderRefunded     AuditEventType = "order_refunded"
	AuditEventTypePaymentProcessed  AuditEventType = "payment_processed"
	AuditEventTypeInventoryAdjusted AuditEventType = "inventory_adjusted"
	AuditEventTypeEmployeeLogin     AuditEventType = "employee_login"
	AuditEventTypeEmployeeLogout    AuditEventType = "employee_logout"
	AuditEventTypeSystemBoot        AuditEventType = "system_boot"
	AuditEventTypeSystemShutdown    AuditEventType = "system_shutdown"
	AuditEventTypeSyncEvent         AuditEventType = "sync_event"
	AuditEventTypeSecurityEvent     AuditEventType = "security_event"
)

type InventoryTransactionType string

const (
	InventoryTransactionTypeSale       InventoryTransactionType = "sale"
	InventoryTransactionTypeRestock    InventoryTransactionType = "restock"
	InventoryTransactionTypeAdjustment InventoryTransactionType = "adjustment"
	InventoryTransactionTypeReserved   InventoryTransactionType = "reserved"
	InventoryTransactionTypeReleased   InventoryTransactionType = "released"
	InventoryTransactionTypeWaste      InventoryTransactionType = "waste"
	InventoryTransactionTypeReturn     InventoryTransactionType = "return"
)

type SyncEventType string

const (
	SyncEventTypeInventoryUpdate   SyncEventType = "inventory_update"
	SyncEventTypeCartMerge         SyncEventType = "cart_merge"
	SyncEventTypeTransactionBatch  SyncEventType = "transaction_batch"
	SyncEventTypeAnalyticsBatch    SyncEventType = "analytics_batch"
)

type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "pending"
	RefundStatusCompleted RefundStatus = "completed"
	RefundStatusFailed    RefundStatus = "failed"
)

// ---------------------------------------------------------------------------
// Tenant / Location / Lane hierarchy
// ---------------------------------------------------------------------------

type Tenant struct {
	TenantID    uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Slug        string     `json:"slug" db:"slug"`
	Name        string     `json:"name" db:"name"`
	BillingEmail string    `json:"billing_email" db:"billing_email"`
	Plan        TenantPlan `json:"plan" db:"plan"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type Location struct {
	LocationID uuid.UUID  `json:"location_id" db:"location_id"`
	TenantID   uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Slug       string     `json:"slug" db:"slug"`
	Name       string     `json:"name" db:"name"`
	Address    *string    `json:"address,omitempty" db:"address"`
	Timezone   string     `json:"timezone" db:"timezone"`
	Currency   string     `json:"currency" db:"currency"`
	TaxRate    float64    `json:"tax_rate" db:"tax_rate"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type Lane struct {
	LaneID      uuid.UUID  `json:"lane_id" db:"lane_id"`
	LocationID  uuid.UUID  `json:"location_id" db:"location_id"`
	DisplayName string     `json:"display_name" db:"display_name"`
	LaneNumber  int        `json:"lane_number" db:"lane_number"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ---------------------------------------------------------------------------
// Stores / Kiosks
// ---------------------------------------------------------------------------

type Store struct {
	StoreID    uuid.UUID  `json:"store_id" db:"store_id"`
	TenantID   *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	LocationID *uuid.UUID `json:"location_id,omitempty" db:"location_id"`
	Name       string     `json:"name" db:"name"`
	Address    *string    `json:"address,omitempty" db:"address"`
	Timezone   string     `json:"timezone" db:"timezone"`
	Currency   string     `json:"currency" db:"currency"`
	TaxRate    float64    `json:"tax_rate" db:"tax_rate"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type Kiosk struct {
	KioskID          uuid.UUID       `json:"kiosk_id" db:"kiosk_id"`
	StoreID          uuid.UUID       `json:"store_id" db:"store_id"`
	LaneID           *uuid.UUID      `json:"lane_id,omitempty" db:"lane_id"`
	TenantID         *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	HardwareID       string          `json:"hardware_id" db:"hardware_id"`
	DisplayName      string          `json:"display_name" db:"display_name"`
	IPAddress        *string         `json:"ip_address,omitempty" db:"ip_address"`
	LastSeenAt       *time.Time      `json:"last_seen_at,omitempty" db:"last_seen_at"`
	SyncStatus       KioskSyncStatus `json:"sync_status" db:"sync_status"`
	IsLeader         bool            `json:"is_leader" db:"is_leader"`
	SigningKeyHash   string          `json:"signing_key_hash" db:"signing_key_hash"`
	FirmwareVersion  *string         `json:"firmware_version,omitempty" db:"firmware_version"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ---------------------------------------------------------------------------
// Menu: Categories / Items / Modifiers
// ---------------------------------------------------------------------------

type Category struct {
	CategoryID   uuid.UUID  `json:"category_id" db:"category_id"`
	StoreID      uuid.UUID  `json:"store_id" db:"store_id"`
	ParentID     *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	Name         string     `json:"name" db:"name"`
	Description  *string    `json:"description,omitempty" db:"description"`
	DisplayOrder int        `json:"display_order" db:"display_order"`
	ImageURL     *string    `json:"image_url,omitempty" db:"image_url"`
	Blurhash     *string    `json:"blurhash,omitempty" db:"blurhash"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type Item struct {
	ItemID        uuid.UUID         `json:"item_id" db:"item_id"`
	StoreID       uuid.UUID         `json:"store_id" db:"store_id"`
	CategoryID    uuid.UUID         `json:"category_id" db:"category_id"`
	Name          string            `json:"name" db:"name"`
	Description   *string           `json:"description,omitempty" db:"description"`
	PriceCents    int               `json:"price_cents" db:"price_cents"`
	CostCents     *int              `json:"cost_cents,omitempty" db:"cost_cents"`
	PLU           *string           `json:"plu,omitempty" db:"plu"`
	Barcode       *string           `json:"barcode,omitempty" db:"barcode"`
	SKU           *string           `json:"sku,omitempty" db:"sku"`
	ImageURL      *string           `json:"image_url,omitempty" db:"image_url"`
	Blurhash      *string           `json:"blurhash,omitempty" db:"blurhash"`
	TaxCategory   ItemTaxCategory   `json:"tax_category" db:"tax_category"`
	IsWeightBased bool              `json:"is_weight_based" db:"is_weight_based"`
	WeightUnit    *WeightUnit       `json:"weight_unit,omitempty" db:"weight_unit"`
	IsActive      bool              `json:"is_active" db:"is_active"`
	Metadata      map[string]any    `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
}

type ModifierGroup struct {
	ModifierGroupID uuid.UUID  `json:"modifier_group_id" db:"modifier_group_id"`
	StoreID         uuid.UUID  `json:"store_id" db:"store_id"`
	Name            string     `json:"name" db:"name"`
	Description     *string    `json:"description,omitempty" db:"description"`
	MinSelect       int        `json:"min_select" db:"min_select"`
	MaxSelect       int        `json:"max_select" db:"max_select"`
	DisplayOrder    int        `json:"display_order" db:"display_order"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type ModifierOption struct {
	ModifierOptionID uuid.UUID `json:"modifier_option_id" db:"modifier_option_id"`
	ModifierGroupID  uuid.UUID `json:"modifier_group_id" db:"modifier_group_id"`
	Name             string    `json:"name" db:"name"`
	PriceDeltaCents  int       `json:"price_delta_cents" db:"price_delta_cents"`
	IsDefault        bool      `json:"is_default" db:"is_default"`
	DisplayOrder     int       `json:"display_order" db:"display_order"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type ItemModifierGroup struct {
	ItemID          uuid.UUID `json:"item_id" db:"item_id"`
	ModifierGroupID uuid.UUID `json:"modifier_group_id" db:"modifier_group_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

type Inventory struct {
	InventoryID       uuid.UUID `json:"inventory_id" db:"inventory_id"`
	StoreID           uuid.UUID `json:"store_id" db:"store_id"`
	ItemID            uuid.UUID `json:"item_id" db:"item_id"`
	QuantityAvailable int       `json:"quantity_available" db:"quantity_available"`
	QuantityReserved  int       `json:"quantity_reserved" db:"quantity_reserved"`
	QuantityOnOrder   int       `json:"quantity_on_order" db:"quantity_on_order"`
	ReorderPoint      int       `json:"reorder_point" db:"reorder_point"`
	ReorderQuantity   int       `json:"reorder_quantity" db:"reorder_quantity"`
	Location          *string   `json:"location,omitempty" db:"location"`
	LastCountedAt     *time.Time `json:"last_counted_at,omitempty" db:"last_counted_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type InventoryTransaction struct {
	TransactionID   uuid.UUID                `json:"transaction_id" db:"transaction_id"`
	StoreID         uuid.UUID                `json:"store_id" db:"store_id"`
	ItemID          uuid.UUID                `json:"item_id" db:"item_id"`
	TransactionType InventoryTransactionType `json:"transaction_type" db:"transaction_type"`
	QuantityDelta   int                      `json:"quantity_delta" db:"quantity_delta"`
	RunningBalance  int                      `json:"running_balance" db:"running_balance"`
	ReferenceID     *uuid.UUID               `json:"reference_id,omitempty" db:"reference_id"`
	ReferenceType   *string                  `json:"reference_type,omitempty" db:"reference_type"`
	KioskID         *uuid.UUID               `json:"kiosk_id,omitempty" db:"kiosk_id"`
	EmployeeID      *uuid.UUID               `json:"employee_id,omitempty" db:"employee_id"`
	Notes           *string                  `json:"notes,omitempty" db:"notes"`
	CreatedAt       time.Time                `json:"created_at" db:"created_at"`
}

type InventoryReservation struct {
	ReservationID uuid.UUID `json:"reservation_id" db:"reservation_id"`
	StoreID       uuid.UUID `json:"store_id" db:"store_id"`
	KioskID       uuid.UUID `json:"kiosk_id" db:"kiosk_id"`
	ItemID        uuid.UUID `json:"item_id" db:"item_id"`
	CartID        uuid.UUID `json:"cart_id" db:"cart_id"`
	Quantity      int       `json:"quantity" db:"quantity"`
	ExpiresAtMs   int64     `json:"expires_at_ms" db:"expires_at_ms"`
	CreatedAtMs   int64     `json:"created_at_ms" db:"created_at_ms"`
}

// ---------------------------------------------------------------------------
// Carts
// ---------------------------------------------------------------------------

type Cart struct {
	CartID            uuid.UUID       `json:"cart_id" db:"cart_id"`
	StoreID           uuid.UUID       `json:"store_id" db:"store_id"`
	KioskID           uuid.UUID       `json:"kiosk_id" db:"kiosk_id"`
	SessionID         uuid.UUID       `json:"session_id" db:"session_id"`
	CustomerPhone     *string         `json:"customer_phone,omitempty" db:"customer_phone"`
	Status            CartStatus      `json:"status" db:"status"`
	Finalized         bool            `json:"finalized" db:"finalized"`
	Version           int             `json:"version" db:"version"`
	TotalCents        int             `json:"total_cents" db:"total_cents"`
	TaxCents          int             `json:"tax_cents" db:"tax_cents"`
	DiscountCents     int             `json:"discount_cents" db:"discount_cents"`
	FinalTotalCents   int             `json:"final_total_cents" db:"final_total_cents"`
	ItemsJSON         []any           `json:"items_json" db:"items_json"`
	ReservedInventory bool            `json:"reserved_inventory" db:"reserved_inventory"`
	ExpiresAt         time.Time       `json:"expires_at" db:"expires_at"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
	CreatedAtMs       int64           `json:"created_at_ms" db:"created_at_ms"`
	UpdatedAtMs       int64           `json:"updated_at_ms" db:"updated_at_ms"`
}

type CartLine struct {
	LineID               uuid.UUID `json:"line_id" db:"line_id"`
	CartID               uuid.UUID `json:"cart_id" db:"cart_id"`
	MenuItemID           uuid.UUID `json:"menu_item_id" db:"menu_item_id"`
	NameSnapshot         string    `json:"name_snapshot" db:"name_snapshot"`
	UnitPriceCentsSnapshot int     `json:"unit_price_cents_snapshot" db:"unit_price_cents_snapshot"`
	Quantity             int       `json:"quantity" db:"quantity"`
	Modifiers            []any     `json:"modifiers" db:"modifiers"`
	AddedAtMs            int64     `json:"added_at_ms" db:"added_at_ms"`
}

// ---------------------------------------------------------------------------
// Orders
// ---------------------------------------------------------------------------

type Order struct {
	OrderID          uuid.UUID       `json:"order_id" db:"order_id"`
	StoreID          uuid.UUID       `json:"store_id" db:"store_id"`
	KioskID          uuid.UUID       `json:"kiosk_id" db:"kiosk_id"`
	CartID           uuid.UUID       `json:"cart_id" db:"cart_id"`
	OrderNumber      string          `json:"order_number" db:"order_number"`
	Status           OrderStatus     `json:"status" db:"status"`
	SubtotalCents    int             `json:"subtotal_cents" db:"subtotal_cents"`
	TaxCents         int             `json:"tax_cents" db:"tax_cents"`
	DiscountCents    int             `json:"discount_cents" db:"discount_cents"`
	TotalCents       int             `json:"total_cents" db:"total_cents"`
	ItemsJSON        []any           `json:"items_json" db:"items_json"`
	TaxBreakdownJSON map[string]any  `json:"tax_breakdown_json,omitempty" db:"tax_breakdown_json"`
	Metadata         map[string]any  `json:"metadata,omitempty" db:"metadata"`
	PaidAt           *time.Time      `json:"paid_at,omitempty" db:"paid_at"`
	FulfilledAt      *time.Time      `json:"fulfilled_at,omitempty" db:"fulfilled_at"`
	CancelledAt      *time.Time      `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

type OrderItem struct {
	OrderItemID      uuid.UUID `json:"order_item_id" db:"order_item_id"`
	OrderID          uuid.UUID `json:"order_id" db:"order_id"`
	ItemID           uuid.UUID `json:"item_id" db:"item_id"`
	NameSnapshot     string    `json:"name_snapshot" db:"name_snapshot"`
	PriceCentsSnapshot int     `json:"price_cents_snapshot" db:"price_cents_snapshot"`
	Quantity         int       `json:"quantity" db:"quantity"`
	ModifiersJSON    []any     `json:"modifiers_json" db:"modifiers_json"`
	LineTotalCents   int       `json:"line_total_cents" db:"line_total_cents"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// ---------------------------------------------------------------------------
// Payments / Refunds / Offline Tokens
// ---------------------------------------------------------------------------

type Payment struct {
	PaymentID        uuid.UUID      `json:"payment_id" db:"payment_id"`
	OrderID          uuid.UUID      `json:"order_id" db:"order_id"`
	KioskID          uuid.UUID      `json:"kiosk_id" db:"kiosk_id"`
	IdempotencyKey   uuid.UUID      `json:"idempotency_key" db:"idempotency_key"`
	AmountCents      int            `json:"amount_cents" db:"amount_cents"`
	Currency         string         `json:"currency" db:"currency"`
	Method           PaymentMethod  `json:"method" db:"method"`
	Status           PaymentStatus  `json:"status" db:"status"`
	VerifoneToken    *string        `json:"verifone_token,omitempty" db:"verifone_token"`
	VerifoneAuthCode *string        `json:"verifone_auth_code,omitempty" db:"verifone_auth_code"`
	CardBrand        *string        `json:"card_brand,omitempty" db:"card_brand"`
	CardLastFour     *string        `json:"card_last_four,omitempty" db:"card_last_four"`
	DeclineReason    *string        `json:"decline_reason,omitempty" db:"decline_reason"`
	ReceiptText      *string        `json:"receipt_text,omitempty" db:"receipt_text"`
	IsOfflineToken   bool           `json:"is_offline_token" db:"is_offline_token"`
	OfflineTokenHMAC *string        `json:"offline_token_hmac,omitempty" db:"offline_token_hmac"`
	SyncedAt         *time.Time     `json:"synced_at,omitempty" db:"synced_at"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
}

type Refund struct {
	RefundID         uuid.UUID     `json:"refund_id" db:"refund_id"`
	PaymentID        uuid.UUID     `json:"payment_id" db:"payment_id"`
	OrderID          uuid.UUID     `json:"order_id" db:"order_id"`
	KioskID          uuid.UUID     `json:"kiosk_id" db:"kiosk_id"`
	AmountCents      int           `json:"amount_cents" db:"amount_cents"`
	Currency         string        `json:"currency" db:"currency"`
	Reason           string        `json:"reason" db:"reason"`
	Status           RefundStatus  `json:"status" db:"status"`
	VerifoneReference *string       `json:"verifone_reference,omitempty" db:"verifone_reference"`
	ProcessedBy      *uuid.UUID    `json:"processed_by,omitempty" db:"processed_by"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at" db:"updated_at"`
}

type OfflineToken struct {
	TokenID             uuid.UUID      `json:"token_id" db:"token_id"`
	StoreID             uuid.UUID      `json:"store_id" db:"store_id"`
	KioskID             uuid.UUID      `json:"kiosk_id" db:"kiosk_id"`
	CartID              uuid.UUID      `json:"cart_id" db:"cart_id"`
	AmountCents         int            `json:"amount_cents" db:"amount_cents"`
	Currency            string         `json:"currency" db:"currency"`
	Method              string         `json:"method" db:"method"`
	VerifoneOpaqueToken string         `json:"verifone_opaque_token" db:"verifone_opaque_token"`
	HMACSignature       string         `json:"hmac_signature" db:"hmac_signature"`
	ExpiresAt           time.Time      `json:"expires_at" db:"expires_at"`
	SettledAt           *time.Time     `json:"settled_at,omitempty" db:"settled_at"`
	SettlementResult    map[string]any `json:"settlement_result,omitempty" db:"settlement_result"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// ---------------------------------------------------------------------------
// Employees / Users / RBAC
// ---------------------------------------------------------------------------

type Employee struct {
	EmployeeID         uuid.UUID     `json:"employee_id" db:"employee_id"`
	StoreID            uuid.UUID     `json:"store_id" db:"store_id"`
	Name               string        `json:"name" db:"name"`
	Email              string        `json:"email" db:"email"`
	Role               EmployeeRole  `json:"role" db:"role"`
	BiometricHash      *string       `json:"biometric_hash,omitempty" db:"biometric_hash"`
	WebauthnCredentialID *string     `json:"webauthn_credential_id,omitempty" db:"webauthn_credential_id"`
	WebauthnPublicKey  []byte        `json:"webauthn_public_key,omitempty" db:"webauthn_public_key"`
	IsActive           bool          `json:"is_active" db:"is_active"`
	LastLoginAt        *time.Time    `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt          time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at" db:"updated_at"`
	DeletedAt          *time.Time    `json:"deleted_at,omitempty" db:"deleted_at"`
}

type Role struct {
	RoleID      uuid.UUID `json:"role_id" db:"role_id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	IsSystem    bool      `json:"is_system" db:"is_system"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Permission struct {
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	Resource     string    `json:"resource" db:"resource"`
	Action       string    `json:"action" db:"action"`
	Description  *string   `json:"description,omitempty" db:"description"`
}

type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id" db:"role_id"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type User struct {
	UserID               uuid.UUID  `json:"user_id" db:"user_id"`
	TenantID             uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Email                string     `json:"email" db:"email"`
	Name                 string     `json:"name" db:"name"`
	RoleID               uuid.UUID  `json:"role_id" db:"role_id"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	WebauthnCredentialID *string    `json:"webauthn_credential_id,omitempty" db:"webauthn_credential_id"`
	WebauthnPublicKey    []byte     `json:"webauthn_public_key,omitempty" db:"webauthn_public_key"`
	LastLoginAt          *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt            *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ---------------------------------------------------------------------------
// Audit / Event Store / Outbox / Sync / Analytics
// ---------------------------------------------------------------------------

type AuditLog struct {
	AuditID       int64          `json:"audit_id" db:"audit_id"`
	StoreID       uuid.UUID      `json:"store_id" db:"store_id"`
	TenantID      *uuid.UUID     `json:"tenant_id,omitempty" db:"tenant_id"`
	LaneID        *uuid.UUID     `json:"lane_id,omitempty" db:"lane_id"`
	KioskID       *uuid.UUID     `json:"kiosk_id,omitempty" db:"kiosk_id"`
	EmployeeID    *uuid.UUID     `json:"employee_id,omitempty" db:"employee_id"`
	UserID        *uuid.UUID     `json:"user_id,omitempty" db:"user_id"`
	EventType     AuditEventType `json:"event_type" db:"event_type"`
	EntityType    string         `json:"entity_type" db:"entity_type"`
	EntityID      uuid.UUID      `json:"entity_id" db:"entity_id"`
	PayloadJSON   map[string]any `json:"payload_json" db:"payload_json"`
	PreviousHash  string         `json:"previous_hash" db:"previous_hash"`
	CurrentHash   string         `json:"current_hash" db:"current_hash"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
}

type EventStoreRecord struct {
	EventID         uuid.UUID      `json:"event_id" db:"event_id"`
	EventSchema     string         `json:"event_schema" db:"event_schema"`
	AggregateType   string         `json:"aggregate_type" db:"aggregate_type"`
	AggregateID     uuid.UUID      `json:"aggregate_id" db:"aggregate_id"`
	SequenceNumber  int64          `json:"sequence_number" db:"sequence_number"`
	Payload         map[string]any `json:"payload" db:"payload"`
	Metadata        map[string]any `json:"metadata" db:"metadata"`
	OccurredAt      time.Time      `json:"occurred_at" db:"occurred_at"`
	RecordedAt      time.Time      `json:"recorded_at" db:"recorded_at"`
}

type OutboxEvent struct {
	EventID       uuid.UUID      `json:"event_id" db:"event_id"`
	AggregateType string         `json:"aggregate_type" db:"aggregate_type"`
	AggregateID   uuid.UUID      `json:"aggregate_id" db:"aggregate_id"`
	EventType     string         `json:"event_type" db:"event_type"`
	Payload       map[string]any `json:"payload" db:"payload"`
	OccurredAtMs  int64          `json:"occurred_at_ms" db:"occurred_at_ms"`
	Published     bool           `json:"published" db:"published"`
	PublishedAtMs *int64         `json:"published_at_ms,omitempty" db:"published_at_ms"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
}

type SyncEvent struct {
	SyncEventID uuid.UUID     `json:"sync_event_id" db:"sync_event_id"`
	StoreID     uuid.UUID     `json:"store_id" db:"store_id"`
	KioskID     uuid.UUID     `json:"kiosk_id" db:"kiosk_id"`
	EventType   SyncEventType `json:"event_type" db:"event_type"`
	PayloadJSON map[string]any `json:"payload_json" db:"payload_json"`
	VectorClock map[string]int64 `json:"vector_clock" db:"vector_clock"`
	ProcessedAt *time.Time    `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}

type AnalyticsEvent struct {
	AnalyticsID  int64          `json:"analytics_id" db:"analytics_id"`
	StoreID      uuid.UUID      `json:"store_id" db:"store_id"`
	KioskID      *uuid.UUID     `json:"kiosk_id,omitempty" db:"kiosk_id"`
	EventType    string         `json:"event_type" db:"event_type"`
	SessionID    *uuid.UUID     `json:"session_id,omitempty" db:"session_id"`
	CustomerHash *string        `json:"customer_hash,omitempty" db:"customer_hash"`
	ItemID       *uuid.UUID     `json:"item_id,omitempty" db:"item_id"`
	CategoryID   *uuid.UUID     `json:"category_id,omitempty" db:"category_id"`
	Quantity     *int           `json:"quantity,omitempty" db:"quantity"`
	AmountCents  *int           `json:"amount_cents,omitempty" db:"amount_cents"`
	DurationMs   *int           `json:"duration_ms,omitempty" db:"duration_ms"`
	Metadata     map[string]any `json:"metadata,omitempty" db:"metadata"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
}
