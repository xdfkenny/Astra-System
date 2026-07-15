package service

import (
	"context"
	"fmt"
	"time"

	menupb "github.com/astra-systems/astra-service/proto/gen/go/menu"
	"github.com/astra-systems/astra-service/services/menu-service/internal/cache"
	"github.com/astra-systems/astra-service/services/menu-service/internal/model"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store is the read-only dependency required by the menu gRPC service.
type Store interface {
	GetCategories(ctx context.Context, storeID uuid.UUID, includeInactive bool) ([]model.Category, error)
	GetItems(ctx context.Context, storeID uuid.UUID, includeInactive bool) ([]model.Item, error)
	GetItemByID(ctx context.Context, itemID uuid.UUID) (*model.Item, error)
	GetModifierGroupsByItemIDs(ctx context.Context, itemIDs []uuid.UUID) (map[uuid.UUID][]model.ModifierGroup, error)
	SearchItems(ctx context.Context, storeID uuid.UUID, query, categoryID string, includeInactive bool) ([]model.Item, error)
}

// MenuService implements astra.menu.v1.MenuService.
type MenuService struct {
	menupb.UnimplementedMenuServiceServer
	repo     Store
	cache    *cache.Cache
	cacheTTL time.Duration
}

// NewMenuService creates a MenuService.
func NewMenuService(repo Store, c *cache.Cache, cacheTTL time.Duration) *MenuService {
	return &MenuService{
		repo:     repo,
		cache:    c,
		cacheTTL: cacheTTL,
	}
}

// GetMenu returns the full menu for a store.
func (s *MenuService) GetMenu(ctx context.Context, req *menupb.MenuRequest) (*menupb.MenuResponse, error) {
	storeID, err := uuid.Parse(req.StoreId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid store_id")
	}

	var cached model.Menu
	if hit, err := s.cache.GetMenu(ctx, storeID.String(), &cached); err != nil {
		return nil, status.Errorf(codes.Internal, "cache error: %v", err)
	} else if hit {
		return menuToProto(&cached), nil
	}

	categories, err := s.repo.GetCategories(ctx, storeID, req.IncludeInactive)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load categories: %v", err)
	}
	items, err := s.repo.GetItems(ctx, storeID, req.IncludeInactive)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load items: %v", err)
	}
	if err := s.attachModifierGroups(ctx, items); err != nil {
		return nil, err
	}

	menu := &model.Menu{
		StoreID:    storeID,
		Categories: categories,
		Items:      items,
	}
	if err := s.cache.SetMenu(ctx, storeID.String(), menu, s.cacheTTL); err != nil {
		return nil, status.Errorf(codes.Internal, "cache set error: %v", err)
	}
	return menuToProto(menu), nil
}

// GetCategories returns only categories for a store.
func (s *MenuService) GetCategories(ctx context.Context, req *menupb.MenuRequest) (*menupb.CategoriesResponse, error) {
	storeID, err := uuid.Parse(req.StoreId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid store_id")
	}

	var cached []model.Category
	if hit, err := s.cache.GetCategories(ctx, storeID.String(), &cached); err != nil {
		return nil, status.Errorf(codes.Internal, "cache error: %v", err)
	} else if hit {
		return &menupb.CategoriesResponse{Categories: categoriesToProto(cached)}, nil
	}

	categories, err := s.repo.GetCategories(ctx, storeID, req.IncludeInactive)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load categories: %v", err)
	}
	if err := s.cache.SetCategories(ctx, storeID.String(), categories, s.cacheTTL); err != nil {
		return nil, status.Errorf(codes.Internal, "cache set error: %v", err)
	}
	return &menupb.CategoriesResponse{Categories: categoriesToProto(categories)}, nil
}

// GetItem returns a single item by id.
func (s *MenuService) GetItem(ctx context.Context, req *menupb.GetItemRequest) (*menupb.Item, error) {
	itemID, err := uuid.Parse(req.ItemId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid item_id")
	}
	item, err := s.repo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load item: %v", err)
	}
	if item == nil {
		return nil, status.Error(codes.NotFound, "item not found")
	}
	if err := s.attachModifierGroups(ctx, []model.Item{*item}); err != nil {
		return nil, err
	}
	return itemToProto(item), nil
}

// SearchItems searches items within a store.
func (s *MenuService) SearchItems(ctx context.Context, req *menupb.SearchItemsRequest) (*menupb.SearchItemsResponse, error) {
	storeID, err := uuid.Parse(req.StoreId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid store_id")
	}
	items, err := s.repo.SearchItems(ctx, storeID, req.Query, req.CategoryId, req.IncludeInactive)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search items: %v", err)
	}
	if err := s.attachModifierGroups(ctx, items); err != nil {
		return nil, err
	}
	return &menupb.SearchItemsResponse{Items: itemsToProto(items)}, nil
}

func (s *MenuService) attachModifierGroups(ctx context.Context, items []model.Item) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(items))
	for i := range items {
		ids[i] = items[i].ItemID
	}
	groups, err := s.repo.GetModifierGroupsByItemIDs(ctx, ids)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to load modifier groups: %v", err)
	}
	for i := range items {
		items[i].ModifierGroups = groups[items[i].ItemID]
	}
	return nil
}

func menuToProto(m *model.Menu) *menupb.MenuResponse {
	return &menupb.MenuResponse{
		StoreId:    m.StoreID.String(),
		Categories: categoriesToProto(m.Categories),
		Items:      itemsToProto(m.Items),
	}
}

func categoriesToProto(in []model.Category) []*menupb.Category {
	out := make([]*menupb.Category, len(in))
	for i, c := range in {
		out[i] = categoryToProto(c)
	}
	return out
}

func categoryToProto(c model.Category) *menupb.Category {
	pb := &menupb.Category{
		CategoryId:   c.CategoryID.String(),
		StoreId:      c.StoreID.String(),
		Name:         c.Name,
		Description:  nullString(c.Description),
		DisplayOrder: int32(c.DisplayOrder),
		ImageUrl:     nullString(c.ImageURL),
		Blurhash:     nullString(c.Blurhash),
		IsActive:     c.IsActive,
	}
	if c.ParentID != nil {
		pb.ParentId = c.ParentID.String()
	}
	return pb
}

func itemsToProto(in []model.Item) []*menupb.Item {
	out := make([]*menupb.Item, len(in))
	for i, it := range in {
		out[i] = itemToProto(&it)
	}
	return out
}

func itemToProto(it *model.Item) *menupb.Item {
	pb := &menupb.Item{
		ItemId:         it.ItemID.String(),
		StoreId:        it.StoreID.String(),
		CategoryId:     it.CategoryID.String(),
		Name:           it.Name,
		Description:    nullString(it.Description),
		PriceCents:     int64(it.PriceCents),
		CostCents:      nullInt64(it.CostCents),
		Plu:            nullString(it.PLU),
		Barcode:        nullString(it.Barcode),
		Sku:            nullString(it.SKU),
		ImageUrl:       nullString(it.ImageURL),
		Blurhash:       nullString(it.Blurhash),
		TaxCategory:    taxCategoryToProto(it.TaxCategory),
		IsWeightBased:  it.IsWeightBased,
		IsActive:       it.IsActive,
		ModifierGroups: modifierGroupsToProto(it.ModifierGroups),
		Metadata:       metadataToProto(it.Metadata),
	}
	if it.WeightUnit != nil {
		pb.WeightUnit = weightUnitToProto(*it.WeightUnit)
	}
	return pb
}

func metadataToProto(in map[string]any) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

func modifierGroupsToProto(in []model.ModifierGroup) []*menupb.ModifierGroup {
	out := make([]*menupb.ModifierGroup, len(in))
	for i, g := range in {
		out[i] = &menupb.ModifierGroup{
			ModifierGroupId: g.ModifierGroupID.String(),
			StoreId:         g.StoreID.String(),
			Name:            g.Name,
			Description:     nullString(g.Description),
			MinSelect:       int32(g.MinSelect),
			MaxSelect:       int32(g.MaxSelect),
			DisplayOrder:    int32(g.DisplayOrder),
			IsActive:        g.IsActive,
			Options:         modifierOptionsToProto(g.Options),
		}
	}
	return out
}

func modifierOptionsToProto(in []model.ModifierOption) []*menupb.ModifierOption {
	out := make([]*menupb.ModifierOption, len(in))
	for i, o := range in {
		out[i] = &menupb.ModifierOption{
			ModifierOptionId: o.ModifierOptionID.String(),
			ModifierGroupId:  o.ModifierGroupID.String(),
			Name:             o.Name,
			PriceDeltaCents:  int64(o.PriceDeltaCents),
			IsDefault:        o.IsDefault,
			DisplayOrder:     int32(o.DisplayOrder),
			IsActive:         o.IsActive,
		}
	}
	return out
}

func taxCategoryToProto(s string) menupb.ItemTaxCategory {
	switch s {
	case "exempt":
		return menupb.ItemTaxCategory_ITEM_TAX_CATEGORY_EXEMPT
	case "reduced":
		return menupb.ItemTaxCategory_ITEM_TAX_CATEGORY_REDUCED
	case "standard":
		return menupb.ItemTaxCategory_ITEM_TAX_CATEGORY_STANDARD
	default:
		return menupb.ItemTaxCategory_ITEM_TAX_CATEGORY_UNSPECIFIED
	}
}

func weightUnitToProto(s string) menupb.WeightUnit {
	switch s {
	case "g":
		return menupb.WeightUnit_WEIGHT_UNIT_GRAM
	case "kg":
		return menupb.WeightUnit_WEIGHT_UNIT_KILOGRAM
	case "lb":
		return menupb.WeightUnit_WEIGHT_UNIT_POUND
	case "oz":
		return menupb.WeightUnit_WEIGHT_UNIT_OUNCE
	default:
		return menupb.WeightUnit_WEIGHT_UNIT_UNSPECIFIED
	}
}

func nullString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullInt64(v *int) int64 {
	if v == nil {
		return 0
	}
	return int64(*v)
}

// compile-time interface check.
var _ menupb.MenuServiceServer = (*MenuService)(nil)
