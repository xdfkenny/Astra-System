// Package repository implements persistence for the admin GraphQL API.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Common domain errors returned by the repository.
var (
	ErrNotFound = errors.New("repository: not found")
)

// Menu models.
type Category struct {
	CategoryID   string
	StoreID      string
	Name         string
	Description  *string
	DisplayOrder int32
	ImageURL     *string
	Blurhash     *string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Item struct {
	ItemID        string
	StoreID       string
	CategoryID    string
	Name          string
	Description   *string
	PriceCents    int64
	CostCents     *int64
	PLU           *string
	Barcode       *string
	SKU           *string
	ImageURL      *string
	Blurhash      *string
	TaxCategory   string
	IsWeightBased bool
	WeightUnit    *string
	IsActive      bool
	Metadata      map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Order models.
type Order struct {
	OrderID       string
	StoreID       string
	KioskID       string
	CartID        string
	OrderNumber   string
	Status        string
	SubtotalCents int64
	TaxCents      int64
	DiscountCents int64
	TotalCents    int64
	Items         []OrderItem
	TaxBreakdown  map[string]string
	Metadata      map[string]string
	PaidAt        *time.Time
	FulfilledAt   *time.Time
	CancelledAt   *time.Time
	CreatedAt     time.Time
}

type OrderItem struct {
	OrderItemID        string
	OrderID            string
	ItemID             string
	NameSnapshot       string
	PriceCentsSnapshot int64
	Quantity           int32
	ModifierOptionIDs  []string
	LineTotalCents     int64
	CreatedAt          time.Time
}

// Inventory model.
type Inventory struct {
	InventoryID       string
	StoreID           string
	ItemID            string
	QuantityAvailable int32
	QuantityReserved  int32
	QuantityOnOrder   int32
	ReorderPoint      int32
	ReorderQuantity   int32
	Location          *string
	LastCountedAt     *time.Time
	UpdatedAt         time.Time
}

// User and role models.
type User struct {
	UserID      string
	TenantID    string
	Email       string
	Name        string
	RoleID      string
	RoleName    string
	IsActive    bool
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Role struct {
	RoleID      string
	TenantID    string
	Name        string
	Description *string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Repository is the persistence contract for the admin GraphQL API.
type Repository interface {
	ListCategories(ctx context.Context, storeID string, includeInactive bool) ([]Category, error)
	ListItems(ctx context.Context, storeID string, includeInactive bool) ([]Item, error)
	ListOrders(ctx context.Context, filter OrderFilter) ([]Order, int64, error)
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	ListInventory(ctx context.Context, storeID string) ([]Inventory, error)
	ListUsers(ctx context.Context, tenantID string) ([]User, error)
	ListRoles(ctx context.Context, tenantID string) ([]Role, error)
}

// OrderFilter controls pagination and filtering for ListOrders.
type OrderFilter struct {
	StoreID  string
	KioskID  string
	Status   string
	Page     int32
	PageSize int32
}

// PostgresRepository is the production implementation backed by PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository returns a repository backed by db.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// ListCategories returns categories for a store.
func (r *PostgresRepository) ListCategories(ctx context.Context, storeID string, includeInactive bool) ([]Category, error) {
	query := `SELECT category_id, store_id, name, description, display_order, image_url, blurhash, is_active, created_at, updated_at
		FROM categories WHERE store_id = $1 AND deleted_at IS NULL`
	args := []any{storeID}
	if !includeInactive {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY display_order, name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: list categories: %w", err)
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var c Category
		var desc, imageURL, blurhash sql.NullString
		if err := rows.Scan(&c.CategoryID, &c.StoreID, &c.Name, &desc, &c.DisplayOrder, &imageURL, &blurhash, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan category: %w", err)
		}
		c.Description = nullStringPtr(desc)
		c.ImageURL = nullStringPtr(imageURL)
		c.Blurhash = nullStringPtr(blurhash)
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate categories: %w", err)
	}
	return out, nil
}

// ListItems returns items for a store.
func (r *PostgresRepository) ListItems(ctx context.Context, storeID string, includeInactive bool) ([]Item, error) {
	query := `SELECT item_id, store_id, category_id, name, description, price_cents, cost_cents, plu, barcode, sku, image_url, blurhash, tax_category, is_weight_based, weight_unit, is_active, metadata, created_at, updated_at
		FROM items WHERE store_id = $1 AND deleted_at IS NULL`
	args := []any{storeID}
	if !includeInactive {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: list items: %w", err)
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var it Item
		var desc, costCents, plu, barcode, sku, imageURL, blurhash, weightUnit sql.NullString
		var metadata []byte
		if err := rows.Scan(
			&it.ItemID, &it.StoreID, &it.CategoryID, &it.Name, &desc, &it.PriceCents, &costCents, &plu, &barcode, &sku, &imageURL, &blurhash, &it.TaxCategory, &it.IsWeightBased, &weightUnit, &it.IsActive, &metadata, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan item: %w", err)
		}
		it.Description = nullStringPtr(desc)
		it.CostCents = nullInt64Ptr(costCents)
		it.PLU = nullStringPtr(plu)
		it.Barcode = nullStringPtr(barcode)
		it.SKU = nullStringPtr(sku)
		it.ImageURL = nullStringPtr(imageURL)
		it.Blurhash = nullStringPtr(blurhash)
		it.WeightUnit = nullStringPtr(weightUnit)
		if len(metadata) > 0 {
			_ = json.Unmarshal(metadata, &it.Metadata)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate items: %w", err)
	}
	return out, nil
}

// ListOrders returns a paginated list of orders with total count.
func (r *PostgresRepository) ListOrders(ctx context.Context, filter OrderFilter) ([]Order, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	offset := (filter.Page - 1) * filter.PageSize

	filterArgs := []any{}
	where := "WHERE 1=1"
	if filter.StoreID != "" {
		filterArgs = append(filterArgs, filter.StoreID)
		where += fmt.Sprintf(" AND store_id = $%d", len(filterArgs))
	}
	if filter.KioskID != "" {
		filterArgs = append(filterArgs, filter.KioskID)
		where += fmt.Sprintf(" AND kiosk_id = $%d", len(filterArgs))
	}
	if filter.Status != "" {
		filterArgs = append(filterArgs, filter.Status)
		where += fmt.Sprintf(" AND status = $%d", len(filterArgs))
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM orders " + where
	if err := r.db.QueryRowContext(ctx, countQuery, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository: count orders: %w", err)
	}

	listArgs := append([]any{filter.PageSize, offset}, filterArgs...)
	listWhere := offsetPlaceholders(where, 2)
	query := `SELECT order_id, store_id, kiosk_id, cart_id, order_number, status,
		subtotal_cents, tax_cents, discount_cents, total_cents,
		items_json, tax_breakdown_json, metadata, paid_at, fulfilled_at, cancelled_at, created_at
		FROM orders ` + listWhere + ` ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository: query orders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, *order)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository: iterate orders: %w", err)
	}
	return orders, total, nil
}

// GetOrder loads a single order by id.
func (r *PostgresRepository) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	order, err := r.queryOrder(ctx, `SELECT order_id, store_id, kiosk_id, cart_id, order_number, status,
		subtotal_cents, tax_cents, discount_cents, total_cents,
		items_json, tax_breakdown_json, metadata, paid_at, fulfilled_at, cancelled_at, created_at
		FROM orders WHERE order_id = $1`, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return order, nil
}

func (r *PostgresRepository) queryOrder(ctx context.Context, query string, args ...any) (*Order, error) {
	row, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: query order: %w", err)
	}
	defer row.Close()
	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("repository: query order: %w", err)
		}
		return nil, sql.ErrNoRows
	}
	return scanOrder(row)
}

func scanOrder(scanner interface {
	Scan(dest ...any) error
}) (*Order, error) {
	var order Order
	var itemsJSON, taxJSON, metaJSON []byte
	var paidAt, fulfilledAt, cancelledAt sql.NullTime

	if err := scanner.Scan(
		&order.OrderID, &order.StoreID, &order.KioskID, &order.CartID, &order.OrderNumber, &order.Status,
		&order.SubtotalCents, &order.TaxCents, &order.DiscountCents, &order.TotalCents,
		&itemsJSON, &taxJSON, &metaJSON,
		&paidAt, &fulfilledAt, &cancelledAt,
		&order.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("repository: scan order: %w", err)
	}

	if paidAt.Valid {
		order.PaidAt = &paidAt.Time
	}
	if fulfilledAt.Valid {
		order.FulfilledAt = &fulfilledAt.Time
	}
	if cancelledAt.Valid {
		order.CancelledAt = &cancelledAt.Time
	}
	if len(itemsJSON) > 0 {
		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("repository: unmarshal items: %w", err)
		}
	}
	if len(taxJSON) > 0 {
		if err := json.Unmarshal(taxJSON, &order.TaxBreakdown); err != nil {
			return nil, fmt.Errorf("repository: unmarshal tax breakdown: %w", err)
		}
	}
	if len(metaJSON) > 0 {
		if err := json.Unmarshal(metaJSON, &order.Metadata); err != nil {
			return nil, fmt.Errorf("repository: unmarshal metadata: %w", err)
		}
	}
	return &order, nil
}

// ListInventory returns inventory rows for a store.
func (r *PostgresRepository) ListInventory(ctx context.Context, storeID string) ([]Inventory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT inventory_id, store_id, item_id, quantity_available, quantity_reserved, quantity_on_order, reorder_point, reorder_quantity, location, last_counted_at, updated_at
		FROM inventory WHERE store_id = $1 ORDER BY item_id`, storeID)
	if err != nil {
		return nil, fmt.Errorf("repository: list inventory: %w", err)
	}
	defer rows.Close()

	var out []Inventory
	for rows.Next() {
		var inv Inventory
		var location sql.NullString
		var lastCounted sql.NullTime
		if err := rows.Scan(
			&inv.InventoryID, &inv.StoreID, &inv.ItemID, &inv.QuantityAvailable, &inv.QuantityReserved, &inv.QuantityOnOrder, &inv.ReorderPoint, &inv.ReorderQuantity, &location, &lastCounted, &inv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan inventory: %w", err)
		}
		inv.Location = nullStringPtr(location)
		if lastCounted.Valid {
			inv.LastCountedAt = &lastCounted.Time
		}
		out = append(out, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate inventory: %w", err)
	}
	return out, nil
}

// ListUsers returns users for a tenant with role names.
func (r *PostgresRepository) ListUsers(ctx context.Context, tenantID string) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.user_id, u.tenant_id, u.email, u.name, u.role_id, r.name, u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM users u JOIN roles r ON u.role_id = r.role_id
		WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
		ORDER BY u.created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("repository: list users: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		var lastLogin sql.NullTime
		if err := rows.Scan(&u.UserID, &u.TenantID, &u.Email, &u.Name, &u.RoleID, &u.RoleName, &u.IsActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan user: %w", err)
		}
		if lastLogin.Valid {
			u.LastLoginAt = &lastLogin.Time
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate users: %w", err)
	}
	return out, nil
}

// ListRoles returns roles for a tenant.
func (r *PostgresRepository) ListRoles(ctx context.Context, tenantID string) ([]Role, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT role_id, tenant_id, name, description, is_system, created_at, updated_at
		FROM roles WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("repository: list roles: %w", err)
	}
	defer rows.Close()

	var out []Role
	for rows.Next() {
		var r Role
		var desc sql.NullString
		if err := rows.Scan(&r.RoleID, &r.TenantID, &r.Name, &desc, &r.IsSystem, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan role: %w", err)
		}
		r.Description = nullStringPtr(desc)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate roles: %w", err)
	}
	return out, nil
}

func nullStringPtr(s sql.NullString) *string {
	if s.Valid {
		return &s.String
	}
	return nil
}

func nullInt64Ptr(s sql.NullString) *int64 {
	if !s.Valid {
		return nil
	}
	var v int64
	if _, err := fmt.Sscanf(s.String, "%d", &v); err != nil {
		return nil
	}
	return &v
}

func offsetPlaceholders(where string, offset int) string {
	out := where
	for i := 10; i >= 1; i-- {
		out = strings.ReplaceAll(out, fmt.Sprintf("$%d", i), fmt.Sprintf("$%d", i+offset))
	}
	return out
}

// Ensure PostgresRepository satisfies Repository.
var _ Repository = (*PostgresRepository)(nil)
