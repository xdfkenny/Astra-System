package cart

import (
	cartv1 "github.com/astra-systems/astra-service/proto/gen/go/cart"
	commonv1 "github.com/astra-systems/astra-service/proto/gen/go/common"
)

// FromProto converts a protobuf Cart into the domain Cart.
func FromProto(pb *cartv1.Cart) *Cart {
	if pb == nil {
		return nil
	}
	lines := make([]Line, 0, len(pb.Lines))
	for _, l := range pb.Lines {
		lines = append(lines, lineFromProto(l))
	}
	return &Cart{
		CartID:            pb.CartId,
		StoreID:           pb.StoreId,
		KioskID:           pb.KioskId,
		SessionID:         pb.SessionId,
		CustomerPhone:     pb.CustomerPhone,
		Status:            CartStatus(pb.Status.String()),
		Finalized:         pb.Finalized,
		Version:           int(pb.Version),
		TotalCents:        int(pb.TotalCents),
		TaxCents:          int(pb.TaxCents),
		DiscountCents:     int(pb.DiscountCents),
		FinalTotalCents:   int(pb.FinalTotalCents),
		Lines:             lines,
		ReservedInventory: pb.ReservedInventory,
	}
}

// ToProto converts the domain Cart into a protobuf Cart.
func (c *Cart) ToProto() *cartv1.Cart {
	if c == nil {
		return nil
	}
	lines := make([]*cartv1.CartLine, 0, len(c.Lines))
	for i := range c.Lines {
		lines = append(lines, c.Lines[i].toProto())
	}
	return &cartv1.Cart{
		CartId:            c.CartID,
		StoreId:           c.StoreID,
		KioskId:           c.KioskID,
		SessionId:         c.SessionID,
		CustomerPhone:     c.CustomerPhone,
		Status:            cartv1.CartStatus(cartv1.CartStatus_value[string(c.Status)]),
		Finalized:         c.Finalized,
		Version:           int32(c.Version),
		TotalCents:        int64(c.TotalCents),
		TaxCents:          int64(c.TaxCents),
		DiscountCents:     int64(c.DiscountCents),
		FinalTotalCents:   int64(c.FinalTotalCents),
		Lines:             lines,
		ReservedInventory: c.ReservedInventory,
		ExpiresAt:         c.ExpiresAt.Format(timeLayoutRFC3339),
		CreatedAt:         c.CreatedAt.Format(timeLayoutRFC3339),
		UpdatedAt:         c.UpdatedAt.Format(timeLayoutRFC3339),
	}
}

func lineFromProto(pb *cartv1.CartLine) Line {
	if pb == nil {
		return Line{}
	}
	mods := make([]Modifier, 0, len(pb.Modifiers))
	for _, m := range pb.Modifiers {
		mods = append(mods, Modifier{
			ModifierOptionID:        m.ModifierOptionId,
			NameSnapshot:            m.NameSnapshot,
			PriceDeltaCentsSnapshot: int(m.PriceDeltaCentsSnapshot),
			ModifierGroupID:         m.ModifierGroupId,
		})
	}
	return Line{
		LineID:                 pb.LineId,
		MenuItemID:             pb.MenuItemId,
		NameSnapshot:           pb.NameSnapshot,
		UnitPriceCentsSnapshot: int(pb.UnitPriceCentsSnapshot),
		Quantity:               int(pb.Quantity),
		Modifiers:              mods,
		AddedAtMs:              pb.AddedAtMs,
	}
}

func (l Line) toProto() *cartv1.CartLine {
	mods := make([]*cartv1.CartModifier, 0, len(l.Modifiers))
	for i := range l.Modifiers {
		mods = append(mods, &cartv1.CartModifier{
			ModifierOptionId:        l.Modifiers[i].ModifierOptionID,
			NameSnapshot:            l.Modifiers[i].NameSnapshot,
			PriceDeltaCentsSnapshot: int64(l.Modifiers[i].PriceDeltaCentsSnapshot),
			ModifierGroupId:         l.Modifiers[i].ModifierGroupID,
		})
	}
	return &cartv1.CartLine{
		LineId:                 l.LineID,
		MenuItemId:             l.MenuItemID,
		NameSnapshot:           l.NameSnapshot,
		UnitPriceCentsSnapshot: int64(l.UnitPriceCentsSnapshot),
		Quantity:               int32(l.Quantity),
		Modifiers:              mods,
		LineTotalCents:         int64(l.LineTotal()),
		AddedAtMs:              l.AddedAtMs,
	}
}

// ModifiersFromProto converts protobuf modifiers into domain modifiers.
func ModifiersFromProto(pb []*cartv1.CartModifier) []Modifier {
	mods := make([]Modifier, 0, len(pb))
	for _, m := range pb {
		if m == nil {
			continue
		}
		mods = append(mods, Modifier{
			ModifierOptionID:        m.ModifierOptionId,
			NameSnapshot:            m.NameSnapshot,
			PriceDeltaCentsSnapshot: int(m.PriceDeltaCentsSnapshot),
			ModifierGroupID:         m.ModifierGroupId,
		})
	}
	return mods
}

// HLCFromProto converts a protobuf HLC into wall-clock milliseconds. It falls
// back to zero when the HLC is absent.
func HLCFromProto(pb *commonv1.HLC) int64 {
	if pb == nil {
		return 0
	}
	return pb.PhysicalTimeMs
}

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"
