package model

import (
	"time"

	"github.com/google/uuid"
)

// Category mirrors the categories table.
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

// ModifierOption mirrors modifier_options.
type ModifierOption struct {
	ModifierOptionID uuid.UUID  `json:"modifier_option_id" db:"modifier_option_id"`
	ModifierGroupID  uuid.UUID  `json:"modifier_group_id" db:"modifier_group_id"`
	Name             string     `json:"name" db:"name"`
	PriceDeltaCents  int        `json:"price_delta_cents" db:"price_delta_cents"`
	IsDefault        bool       `json:"is_default" db:"is_default"`
	DisplayOrder     int        `json:"display_order" db:"display_order"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ModifierGroup mirrors modifier_groups with nested options.
type ModifierGroup struct {
	ModifierGroupID uuid.UUID        `json:"modifier_group_id" db:"modifier_group_id"`
	StoreID         uuid.UUID        `json:"store_id" db:"store_id"`
	Name            string           `json:"name" db:"name"`
	Description     *string          `json:"description,omitempty" db:"description"`
	MinSelect       int              `json:"min_select" db:"min_select"`
	MaxSelect       int              `json:"max_select" db:"max_select"`
	DisplayOrder    int              `json:"display_order" db:"display_order"`
	IsActive        bool             `json:"is_active" db:"is_active"`
	Options         []ModifierOption `json:"options" db:"-"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time       `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Item mirrors items with nested modifier groups.
type Item struct {
	ItemID         uuid.UUID       `json:"item_id" db:"item_id"`
	StoreID        uuid.UUID       `json:"store_id" db:"store_id"`
	CategoryID     uuid.UUID       `json:"category_id" db:"category_id"`
	Name           string          `json:"name" db:"name"`
	Description    *string         `json:"description,omitempty" db:"description"`
	PriceCents     int             `json:"price_cents" db:"price_cents"`
	CostCents      *int            `json:"cost_cents,omitempty" db:"cost_cents"`
	PLU            *string         `json:"plu,omitempty" db:"plu"`
	Barcode        *string         `json:"barcode,omitempty" db:"barcode"`
	SKU            *string         `json:"sku,omitempty" db:"sku"`
	ImageURL       *string         `json:"image_url,omitempty" db:"image_url"`
	Blurhash       *string         `json:"blurhash,omitempty" db:"blurhash"`
	TaxCategory    string          `json:"tax_category" db:"tax_category"`
	IsWeightBased  bool            `json:"is_weight_based" db:"is_weight_based"`
	WeightUnit     *string         `json:"weight_unit,omitempty" db:"weight_unit"`
	IsActive       bool            `json:"is_active" db:"is_active"`
	ModifierGroups []ModifierGroup `json:"modifier_groups" db:"-"`
	Metadata       map[string]any  `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Menu is the aggregate of categories and items for a store.
type Menu struct {
	StoreID    uuid.UUID  `json:"store_id"`
	Categories []Category `json:"categories"`
	Items      []Item     `json:"items"`
}
