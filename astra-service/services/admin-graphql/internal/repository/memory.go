package repository

import (
	"context"
	"sort"
	"sync"
)

// MemoryRepository is an in-memory implementation of Repository for unit tests.
type MemoryRepository struct {
	mu         sync.RWMutex
	categories []Category
	items      []Item
	orders     []Order
	inventory  []Inventory
	users      []User
	roles      []Role
}

// NewMemoryRepository returns an empty in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{}
}

// SetCategories seeds categories.
func (r *MemoryRepository) SetCategories(cats []Category) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.categories = cats
}

// SetItems seeds items.
func (r *MemoryRepository) SetItems(items []Item) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = items
}

// SetOrders seeds orders.
func (r *MemoryRepository) SetOrders(orders []Order) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders = orders
}

// SetInventory seeds inventory.
func (r *MemoryRepository) SetInventory(inv []Inventory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inventory = inv
}

// SetUsers seeds users.
func (r *MemoryRepository) SetUsers(users []User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users = users
}

// SetRoles seeds roles.
func (r *MemoryRepository) SetRoles(roles []Role) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles = roles
}

// ListCategories returns categories for a store.
func (r *MemoryRepository) ListCategories(ctx context.Context, storeID string, includeInactive bool) ([]Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Category
	for _, c := range r.categories {
		if c.StoreID != storeID {
			continue
		}
		if !includeInactive && !c.IsActive {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DisplayOrder != out[j].DisplayOrder {
			return out[i].DisplayOrder < out[j].DisplayOrder
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// ListItems returns items for a store.
func (r *MemoryRepository) ListItems(ctx context.Context, storeID string, includeInactive bool) ([]Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Item
	for _, it := range r.items {
		if it.StoreID != storeID {
			continue
		}
		if !includeInactive && !it.IsActive {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ListOrders returns orders with pagination.
func (r *MemoryRepository) ListOrders(ctx context.Context, filter OrderFilter) ([]Order, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []Order
	for _, o := range r.orders {
		if filter.StoreID != "" && o.StoreID != filter.StoreID {
			continue
		}
		if filter.KioskID != "" && o.KioskID != filter.KioskID {
			continue
		}
		if filter.Status != "" && o.Status != filter.Status {
			continue
		}
		filtered = append(filtered, o)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.After(filtered[j].CreatedAt) })

	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	total := int64(len(filtered))
	offset := (page - 1) * pageSize
	if offset > int32(len(filtered)) {
		return nil, total, nil
	}
	end := offset + pageSize
	if end > int32(len(filtered)) {
		end = int32(len(filtered))
	}
	return filtered[offset:end], total, nil
}

// GetOrder returns an order by id.
func (r *MemoryRepository) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, o := range r.orders {
		if o.OrderID == orderID {
			return &o, nil
		}
	}
	return nil, ErrNotFound
}

// ListInventory returns inventory rows for a store.
func (r *MemoryRepository) ListInventory(ctx context.Context, storeID string) ([]Inventory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Inventory
	for _, inv := range r.inventory {
		if inv.StoreID == storeID {
			out = append(out, inv)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ItemID < out[j].ItemID })
	return out, nil
}

// ListUsers returns users for a tenant.
func (r *MemoryRepository) ListUsers(ctx context.Context, tenantID string) ([]User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []User
	for _, u := range r.users {
		if u.TenantID == tenantID {
			out = append(out, u)
		}
	}
	return out, nil
}

// ListRoles returns roles for a tenant.
func (r *MemoryRepository) ListRoles(ctx context.Context, tenantID string) ([]Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Role
	for _, r := range r.roles {
		if r.TenantID == tenantID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Ensure MemoryRepository satisfies Repository.
var _ Repository = (*MemoryRepository)(nil)
