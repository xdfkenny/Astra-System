// Package crdt implements the Ghost Cart merge strategy for Astra-Service.
//
// A Ghost Cart is created when a kiosk operates offline: a customer adds items
// locally, and when connectivity returns the ghost cart must be merged into
// the authoritative cloud cart without losing data. The merge is modeled as a
// state-based CRDT: the result is independent of merge order and idempotent.
package crdt

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	"github.com/astra-systems/astra-service/services/cart-service/internal/cart"
)

// MergeResult describes the outcome of a ghost-cart merge.
type MergeResult struct {
	Cart       *cart.Cart
	SourceHLC  int64
	TargetHLC  int64
	LinesAdded int
}

// MergeCarts merges source (ghost) into target (authoritative) using a
// last-write-wins strategy for cart-level metadata and a commutative
// item-grouping strategy for lines.
//
// Line merge rules:
//   - Lines are grouped by a canonical fingerprint of menu_item_id + sorted
//     modifier_option_ids. This treats identical menu configurations as the
//     same logical item regardless of line_id.
//   - Quantities for the same logical item are summed.
//   - The added_at_ms of a merged group is the maximum of its constituents.
//   - The name_snapshot and unit_price_cents_snapshot come from the source
//     line if the source HLC is newer, otherwise from the target.
func MergeCarts(target, source *cart.Cart, sourceHLC int64, now time.Time) (*MergeResult, error) {
	if target == nil {
		return nil, cart.ErrCartNotFound
	}
	if source == nil {
		return &MergeResult{Cart: target}, nil
	}

	merged := *target
	merged.Lines = make([]cart.Line, 0, len(target.Lines)+len(source.Lines))

	groups := make(map[string]*lineGroup)
	for _, line := range target.Lines {
		fp := lineFingerprint(line)
		groups[fp] = &lineGroup{
			line:      line,
			quantity:  line.Quantity,
			addedAtMs: line.AddedAtMs,
			source:    false,
		}
	}

	for _, line := range source.Lines {
		fp := lineFingerprint(line)
		g, exists := groups[fp]
		if !exists {
			groups[fp] = &lineGroup{
				line:      line,
				quantity:  line.Quantity,
				addedAtMs: line.AddedAtMs,
				source:    true,
			}
			continue
		}
		g.quantity += line.Quantity
		if line.AddedAtMs > g.addedAtMs {
			g.addedAtMs = line.AddedAtMs
		}
		// If the ghost cart wrote this logical item more recently, prefer its
		// price/name snapshot so the cart reflects the most recent menu state.
		if sourceHLC > target.UpdatedAtMs {
			g.line.NameSnapshot = line.NameSnapshot
			g.line.UnitPriceCentsSnapshot = line.UnitPriceCentsSnapshot
			g.line.Modifiers = line.Modifiers
		}
		g.source = true
	}

	for _, g := range groups {
		l := g.line
		l.Quantity = g.quantity
		l.AddedAtMs = g.addedAtMs
		merged.Lines = append(merged.Lines, l)
	}

	sortLines(merged.Lines)
	recomputeTotals(&merged)
	touch(&merged, now)

	return &MergeResult{
		Cart:       &merged,
		SourceHLC:  sourceHLC,
		TargetHLC:  target.UpdatedAtMs,
		LinesAdded: len(merged.Lines),
	}, nil
}

type lineGroup struct {
	line      cart.Line
	quantity  int
	addedAtMs int64
	source    bool
}

func lineFingerprint(l cart.Line) string {
	ids := make([]string, 0, len(l.Modifiers))
	for _, m := range l.Modifiers {
		ids = append(ids, m.ModifierOptionID)
	}
	sort.Strings(ids)
	h := sha256.New()
	h.Write([]byte(l.MenuItemID))
	for _, id := range ids {
		h.Write([]byte(id))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sortLines(lines []cart.Line) {
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].AddedAtMs != lines[j].AddedAtMs {
			return lines[i].AddedAtMs < lines[j].AddedAtMs
		}
		return lines[i].LineID < lines[j].LineID
	})
}

func recomputeTotals(c *cart.Cart) {
	total := 0
	for _, line := range c.Lines {
		total += line.LineTotal()
	}
	c.TotalCents = total
	c.FinalTotalCents = total + c.TaxCents - c.DiscountCents
}

func touch(c *cart.Cart, now time.Time) {
	c.Version++
	c.UpdatedAt = now
	c.UpdatedAtMs = now.UnixMilli()
}
