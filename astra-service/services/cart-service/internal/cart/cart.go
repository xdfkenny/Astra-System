// Package cart holds the Cart aggregate and value objects shared by the
// service, repository, CRDT merge, and cache layers. It intentionally has no
// external I/O dependencies.
package cart

import (
	"errors"
	"time"
)

var (
	ErrCartNotFound      = errors.New("cart: not found")
	ErrCartFinalized     = errors.New("cart: cannot mutate a finalized cart")
	ErrLineNotFound      = errors.New("cart: line item not found")
	ErrQuantityInvalid   = errors.New("cart: quantity must be positive")
	ErrVersionConflict   = errors.New("cart: optimistic locking conflict")
	ErrInvalidID         = errors.New("cart: invalid identifier")
	ErrInventoryConflict = errors.New("cart: inventory reservation failed")
)

// CartStatus mirrors the database enum values for carts.
type CartStatus string

const (
	CartStatusActive    CartStatus = "active"
	CartStatusFinalized CartStatus = "finalized"
	CartStatusAbandoned CartStatus = "abandoned"
	CartStatusExpired   CartStatus = "expired"
)

// Modifier captures a customer-selected modifier option at the time the line
// was added. The snapshot fields make the cart immutable with respect to
// subsequent menu changes.
type Modifier struct {
	ModifierOptionID      string `json:"modifier_option_id"`
	NameSnapshot          string `json:"name_snapshot"`
	PriceDeltaCentsSnapshot int  `json:"price_delta_cents_snapshot"`
	ModifierGroupID       string `json:"modifier_group_id"`
}

// Line represents a single cart line.
type Line struct {
	CartID                 string     `json:"cart_id"`
	LineID                 string     `json:"line_id"`
	MenuItemID             string     `json:"menu_item_id"`
	NameSnapshot           string     `json:"name_snapshot"`
	UnitPriceCentsSnapshot int        `json:"unit_price_cents_snapshot"`
	Quantity               int        `json:"quantity"`
	Modifiers              []Modifier `json:"modifiers"`
	AddedAtMs              int64      `json:"added_at_ms"`
}

// LineTotal returns the extended price for this line including modifiers.
func (l Line) LineTotal() int {
	modTotal := 0
	for _, m := range l.Modifiers {
		modTotal += m.PriceDeltaCentsSnapshot
	}
	return l.Quantity * (l.UnitPriceCentsSnapshot + modTotal)
}

// Cart is the aggregate root. Version is a monotonic optimistic-lock counter
// incremented on every mutating operation.
type Cart struct {
	CartID            string
	StoreID           string
	KioskID           string
	LaneID            string
	SessionID         string
	CustomerPhone     string
	Status            CartStatus
	Finalized         bool
	Version           int
	TotalCents        int
	TaxCents          int
	DiscountCents     int
	FinalTotalCents   int
	Lines             []Line
	ReservedInventory bool
	ExpiresAt         time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedAtMs       int64
	UpdatedAtMs       int64
}

// NewCart creates a new active cart with an initial version of zero.
func NewCart(cartID, storeID, kioskID, laneID, sessionID, customerPhone string, now time.Time) *Cart {
	ms := now.UnixMilli()
	return &Cart{
		CartID:        cartID,
		StoreID:       storeID,
		KioskID:       kioskID,
		LaneID:        laneID,
		SessionID:     sessionID,
		CustomerPhone: customerPhone,
		Status:        CartStatusActive,
		Version:       0,
		Lines:         []Line{},
		ExpiresAt:     now.Add(10 * time.Minute),
		CreatedAt:     now,
		UpdatedAt:     now,
		CreatedAtMs:   ms,
		UpdatedAtMs:   ms,
	}
}

// AddLine appends a new line to the cart and recomputes totals.
func (c *Cart) AddLine(line Line, now time.Time) error {
	if c.Finalized {
		return ErrCartFinalized
	}
	if line.Quantity <= 0 {
		return ErrQuantityInvalid
	}
	c.Lines = append(c.Lines, line)
	c.recomputeTotals()
	c.touch(now)
	return nil
}

// UpdateLine changes the quantity and modifiers of an existing line.
func (c *Cart) UpdateLine(lineID string, quantity int, modifiers []Modifier, now time.Time) error {
	if c.Finalized {
		return ErrCartFinalized
	}
	if quantity <= 0 {
		return c.RemoveLine(lineID, now)
	}
	for i := range c.Lines {
		if c.Lines[i].LineID == lineID {
			c.Lines[i].Quantity = quantity
			c.Lines[i].Modifiers = modifiers
			c.recomputeTotals()
			c.touch(now)
			return nil
		}
	}
	return ErrLineNotFound
}

// RemoveLine deletes a line from the cart.
func (c *Cart) RemoveLine(lineID string, now time.Time) error {
	if c.Finalized {
		return ErrCartFinalized
	}
	for i := range c.Lines {
		if c.Lines[i].LineID == lineID {
			c.Lines = append(c.Lines[:i], c.Lines[i+1:]...)
			c.recomputeTotals()
			c.touch(now)
			return nil
		}
	}
	return ErrLineNotFound
}

// Finalize transitions the cart to an immutable finalized state and sets the
// final total to the current total plus tax minus discount.
func (c *Cart) Finalize(now time.Time) error {
	if c.Finalized {
		return ErrCartFinalized
	}
	c.Finalized = true
	c.Status = CartStatusFinalized
	c.FinalTotalCents = c.TotalCents + c.TaxCents - c.DiscountCents
	c.touch(now)
	return nil
}

func (c *Cart) recomputeTotals() {
	total := 0
	for _, line := range c.Lines {
		total += line.LineTotal()
	}
	c.TotalCents = total
	c.FinalTotalCents = total + c.TaxCents - c.DiscountCents
}

func (c *Cart) touch(now time.Time) {
	c.Version++
	c.UpdatedAt = now
	c.UpdatedAtMs = now.UnixMilli()
}
