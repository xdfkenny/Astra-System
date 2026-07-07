// Package service implements the astra.cart.v1.CartService gRPC service.
package service

import (
	"context"
	"time"

	"github.com/astra-systems/astra-service/proto/gen/go/cart"
	cartv1 "github.com/astra-systems/astra-service/proto/gen/go/cart"
	commonv1 "github.com/astra-systems/astra-service/proto/gen/go/common"
	"github.com/astra-systems/astra-service/services/cart-service/internal/cache"
	cartdom "github.com/astra-systems/astra-service/services/cart-service/internal/cart"
	"github.com/astra-systems/astra-service/services/cart-service/internal/crdt"
	"github.com/astra-systems/astra-service/services/cart-service/internal/inventory"
	"github.com/astra-systems/astra-service/services/cart-service/internal/outbox"
	"github.com/astra-systems/astra-service/services/cart-service/internal/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CartService implements the generated CartServiceServer interface.
type CartService struct {
	cart.UnimplementedCartServiceServer

	repo      *repository.CartRepository
	cache     *cache.CartCache
	inventory inventory.ReservationClient
	currency  string
}

// NewCartService wires the service dependencies.
func NewCartService(repo *repository.CartRepository, cache *cache.CartCache, inv inventory.ReservationClient, currency string) *CartService {
	if currency == "" {
		currency = "USD"
	}
	return &CartService{
		repo:      repo,
		cache:     cache,
		inventory: inv,
		currency:  currency,
	}
}

// CreateCart creates a new active cart and caches it by lane/session.
func (s *CartService) CreateCart(ctx context.Context, req *cartv1.CreateCartRequest) (*cartv1.Cart, error) {
	if req.StoreId == "" || req.KioskId == "" || req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "store_id, kiosk_id, and session_id are required")
	}

	cartID := uuid.New().String()
	now := time.Now()
	// When lane_id is not provided in the request, the kiosk_id is used as the
	// lane identifier so the cache key format remains "cart:{lane}:{session}".
	c := cartdom.NewCart(cartID, req.StoreId, req.KioskId, req.KioskId, req.SessionId, req.CustomerPhone, now)

	if err := s.repo.CreateCart(ctx, c); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: create cart: %v", err)
	}
	if err := s.cache.Set(ctx, c.LaneID, c.SessionID, c); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: cache cart: %v", err)
	}

	return c.ToProto(), nil
}

// GetCart returns a cart by ID, refreshing the session cache on success.
func (s *CartService) GetCart(ctx context.Context, req *cartv1.GetCartRequest) (*cartv1.Cart, error) {
	if req.CartId == "" {
		return nil, status.Error(codes.InvalidArgument, "cart_id is required")
	}

	c, err := s.repo.GetCart(ctx, req.CartId)
	if err != nil {
		if err == cartdom.ErrCartNotFound {
			return nil, status.Error(codes.NotFound, "cart not found")
		}
		return nil, status.Errorf(codes.Internal, "cart_service: get cart: %v", err)
	}

	if err := s.cache.Set(ctx, c.LaneID, c.SessionID, c); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: cache cart: %v", err)
	}

	return c.ToProto(), nil
}

// AddItem adds a line to the cart, reserves inventory, and emits an outbox
// event. The entire write is atomic via repository.SaveCart.
func (s *CartService) AddItem(ctx context.Context, req *cartv1.AddItemRequest) (*cartv1.Cart, error) {
	if req.CartId == "" || req.MenuItemId == "" || req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "cart_id, menu_item_id, and positive quantity are required")
	}

	c, err := s.loadAndLock(ctx, req.CartId)
	if err != nil {
		return nil, err
	}

	line := cartdom.Line{
		LineID:                 uuid.New().String(),
		MenuItemID:             req.MenuItemId,
		NameSnapshot:           "", // populated by caller/menu service in production
		UnitPriceCentsSnapshot: 0,  // populated by caller/menu service in production
		Quantity:               int(req.Quantity),
		Modifiers:              cartdom.ModifiersFromProto(req.Modifiers),
		AddedAtMs:              time.Now().UnixMilli(),
	}
	if err := c.AddLine(line, time.Now()); err != nil {
		return nil, mapDomainError(err)
	}

	if s.inventory != nil {
		if _, err := s.inventory.Reserve(ctx, c.StoreID, c.KioskID, line.MenuItemID, c.CartID, line.Quantity, inventory.DefaultExpiresAtMs()); err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "cart_service: reserve inventory: %v", err)
		}
	}

	event, err := outbox.NewItemAddedToCart(c, line)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: build event: %v", err)
	}

	if err := s.repo.SaveCart(ctx, c, &event); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: save cart: %v", err)
	}
	if err := s.cache.Set(ctx, c.LaneID, c.SessionID, c); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: cache cart: %v", err)
	}

	return c.ToProto(), nil
}

// UpdateItem modifies an existing cart line and re-reserves inventory.
func (s *CartService) UpdateItem(ctx context.Context, req *cartv1.UpdateItemRequest) (*cartv1.Cart, error) {
	if req.CartId == "" || req.LineId == "" || req.Quantity < 0 {
		return nil, status.Error(codes.InvalidArgument, "cart_id, line_id, and non-negative quantity are required")
	}

	c, err := s.loadAndLock(ctx, req.CartId)
	if err != nil {
		return nil, err
	}

	modifiers := cartdom.ModifiersFromProto(req.Modifiers)
	if err := c.UpdateLine(req.LineId, int(req.Quantity), modifiers, time.Now()); err != nil {
		return nil, mapDomainError(err)
	}

	if err := s.saveWithoutEvent(ctx, c); err != nil {
		return nil, err
	}
	return c.ToProto(), nil
}

// RemoveItem deletes a cart line.
func (s *CartService) RemoveItem(ctx context.Context, req *cartv1.RemoveItemRequest) (*cartv1.Cart, error) {
	if req.CartId == "" || req.LineId == "" {
		return nil, status.Error(codes.InvalidArgument, "cart_id and line_id are required")
	}

	c, err := s.loadAndLock(ctx, req.CartId)
	if err != nil {
		return nil, err
	}

	if err := c.RemoveLine(req.LineId, time.Now()); err != nil {
		return nil, mapDomainError(err)
	}

	if err := s.saveWithoutEvent(ctx, c); err != nil {
		return nil, err
	}
	return c.ToProto(), nil
}

// FinalizeCart transitions the cart to immutable state and emits CartFinalized.
func (s *CartService) FinalizeCart(ctx context.Context, req *cartv1.FinalizeCartRequest) (*cartv1.Cart, error) {
	if req.CartId == "" {
		return nil, status.Error(codes.InvalidArgument, "cart_id is required")
	}

	c, err := s.loadAndLock(ctx, req.CartId)
	if err != nil {
		return nil, err
	}

	if req.CustomerPhone != "" {
		c.CustomerPhone = req.CustomerPhone
	}

	if err := c.Finalize(time.Now()); err != nil {
		return nil, mapDomainError(err)
	}

	orderID := uuid.New().String()
	event, err := outbox.NewCartFinalized(c, orderID, s.currency)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: build event: %v", err)
	}

	if err := s.repo.SaveCart(ctx, c, &event); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: save cart: %v", err)
	}
	if err := s.cache.Delete(ctx, c.LaneID, c.SessionID); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: delete cache: %v", err)
	}

	return c.ToProto(), nil
}

// MergeGhostCart merges a ghost cart into the target cart using CRDT semantics.
func (s *CartService) MergeGhostCart(ctx context.Context, req *cartv1.MergeGhostCartRequest) (*cartv1.Cart, error) {
	if req.TargetCartId == "" || req.GhostCartId == "" {
		return nil, status.Error(codes.InvalidArgument, "target_cart_id and ghost_cart_id are required")
	}

	target, err := s.loadAndLock(ctx, req.TargetCartId)
	if err != nil {
		return nil, err
	}

	ghost, err := s.repo.GetCart(ctx, req.GhostCartId)
	if err != nil {
		if err == cartdom.ErrCartNotFound {
			return nil, status.Error(codes.NotFound, "ghost cart not found")
		}
		return nil, status.Errorf(codes.Internal, "cart_service: get ghost cart: %v", err)
	}

	sourceHLC := cartdom.HLCFromProto(req.SourceHlc)
	merged, err := crdt.MergeCarts(target, ghost, sourceHLC, time.Now())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: merge carts: %v", err)
	}
	c := merged.Cart

	if err := s.saveWithoutEvent(ctx, c); err != nil {
		return nil, err
	}
	if err := s.repo.DeleteCart(ctx, ghost.CartID); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: delete ghost cart: %v", err)
	}
	if err := s.cache.Set(ctx, c.LaneID, c.SessionID, c); err != nil {
		return nil, status.Errorf(codes.Internal, "cart_service: cache cart: %v", err)
	}

	return c.ToProto(), nil
}

func (s *CartService) loadAndLock(ctx context.Context, cartID string) (*cartdom.Cart, error) {
	c, err := s.repo.GetCart(ctx, cartID)
	if err != nil {
		if err == cartdom.ErrCartNotFound {
			return nil, status.Error(codes.NotFound, "cart not found")
		}
		return nil, status.Errorf(codes.Internal, "cart_service: load cart: %v", err)
	}
	return c, nil
}

func (s *CartService) saveWithoutEvent(ctx context.Context, c *cartdom.Cart) error {
	if err := s.repo.SaveCart(ctx, c, nil); err != nil {
		return status.Errorf(codes.Internal, "cart_service: save cart: %v", err)
	}
	if err := s.cache.Set(ctx, c.LaneID, c.SessionID, c); err != nil {
		return status.Errorf(codes.Internal, "cart_service: cache cart: %v", err)
	}
	return nil
}

func mapDomainError(err error) error {
	switch err {
	case cartdom.ErrCartNotFound:
		return status.Error(codes.NotFound, "cart not found")
	case cartdom.ErrCartFinalized:
		return status.Error(codes.FailedPrecondition, "cart is finalized")
	case cartdom.ErrLineNotFound:
		return status.Error(codes.NotFound, "line not found")
	case cartdom.ErrQuantityInvalid:
		return status.Error(codes.InvalidArgument, "invalid quantity")
	case cartdom.ErrVersionConflict:
		return status.Error(codes.Aborted, "optimistic locking conflict")
	default:
		return status.Errorf(codes.Internal, "cart_service: %v", err)
	}
}

// compile-time interface assertion.
var _ cartv1.CartServiceServer = (*CartService)(nil)
var _ = (*commonv1.HLC)(nil)
