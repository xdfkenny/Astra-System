// Package resolver implements GraphQL resolvers for the admin-graphql service.
package resolver

import (
	"context"
	"fmt"

	"github.com/astra-systems/astra-service/services/admin-graphql/internal/auth"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/repository"
	"github.com/graphql-go/graphql"
)

// NewSchema builds the admin GraphQL schema backed by repo.
func NewSchema(repo repository.Repository) (graphql.Schema, error) {
	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType(repo),
		Mutation: mutationType(repo),
	})
}

func queryType(repo repository.Repository) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"menus": &graphql.Field{
				Type: menuType,
				Args: graphql.FieldConfigArgument{
					"storeId":         &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"includeInactive": &graphql.ArgumentConfig{Type: graphql.Boolean, DefaultValue: false},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					storeID := p.Args["storeId"].(string)
					includeInactive := false
					if v, ok := p.Args["includeInactive"].(bool); ok {
						includeInactive = v
					}
					categories, err := repo.ListCategories(p.Context, storeID, includeInactive)
					if err != nil {
						return nil, err
					}
					items, err := repo.ListItems(p.Context, storeID, includeInactive)
					if err != nil {
						return nil, err
					}
					return Menu{StoreID: storeID, Categories: categories, Items: items}, nil
				},
			},
			"orders": &graphql.Field{
				Type: orderConnectionType,
				Args: graphql.FieldConfigArgument{
					"storeId":  &graphql.ArgumentConfig{Type: graphql.ID},
					"kioskId":  &graphql.ArgumentConfig{Type: graphql.ID},
					"status":   &graphql.ArgumentConfig{Type: graphql.String},
					"page":     &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 1},
					"pageSize": &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 20},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					filter := repository.OrderFilter{
						Page:     int32(p.Args["page"].(int)),
						PageSize: int32(p.Args["pageSize"].(int)),
					}
					if v, ok := p.Args["storeId"].(string); ok {
						filter.StoreID = v
					}
					if v, ok := p.Args["kioskId"].(string); ok {
						filter.KioskID = v
					}
					if v, ok := p.Args["status"].(string); ok {
						filter.Status = v
					}
					orders, total, err := repo.ListOrders(p.Context, filter)
					if err != nil {
						return nil, err
					}
					return OrderConnection{Nodes: orders, Total: total}, nil
				},
			},
			"order": &graphql.Field{
				Type: orderType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return repo.GetOrder(p.Context, p.Args["id"].(string))
				},
			},
			"inventory": &graphql.Field{
				Type: graphql.NewList(inventoryType),
				Args: graphql.FieldConfigArgument{
					"storeId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return repo.ListInventory(p.Context, p.Args["storeId"].(string))
				},
			},
			"users": &graphql.Field{
				Type: graphql.NewList(userType),
				Args: graphql.FieldConfigArgument{
					"tenantId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return repo.ListUsers(p.Context, p.Args["tenantId"].(string))
				},
			},
			"roles": &graphql.Field{
				Type: graphql.NewList(roleType),
				Args: graphql.FieldConfigArgument{
					"tenantId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return repo.ListRoles(p.Context, p.Args["tenantId"].(string))
				},
			},
		},
	})
}

func mutationType(repo repository.Repository) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"noop": &graphql.Field{
				Type: graphql.Boolean,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return true, nil
				},
			},
		},
	})
}

// Menu is the GraphQL wrapper for a store menu.
type Menu struct {
	StoreID    string
	Categories []repository.Category
	Items      []repository.Item
}

// OrderConnection is a paginated list of orders.
type OrderConnection struct {
	Nodes []repository.Order
	Total int64
}

var menuType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Menu",
	Fields: graphql.Fields{
		"storeId":    &graphql.Field{Type: graphql.ID},
		"categories": &graphql.Field{Type: graphql.NewList(categoryType)},
		"items":      &graphql.Field{Type: graphql.NewList(itemType)},
	},
})

var categoryType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Category",
	Fields: graphql.Fields{
		"id":           &graphql.Field{Type: graphql.ID},
		"storeId":      &graphql.Field{Type: graphql.ID},
		"name":         &graphql.Field{Type: graphql.String},
		"description":  &graphql.Field{Type: graphql.String},
		"displayOrder": &graphql.Field{Type: graphql.Int},
		"imageUrl":     &graphql.Field{Type: graphql.String},
		"blurhash":     &graphql.Field{Type: graphql.String},
		"isActive":     &graphql.Field{Type: graphql.Boolean},
	},
})

var itemType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Item",
	Fields: graphql.Fields{
		"id":            &graphql.Field{Type: graphql.ID},
		"storeId":       &graphql.Field{Type: graphql.ID},
		"categoryId":    &graphql.Field{Type: graphql.ID},
		"name":          &graphql.Field{Type: graphql.String},
		"description":   &graphql.Field{Type: graphql.String},
		"priceCents":    &graphql.Field{Type: graphql.Int},
		"costCents":     &graphql.Field{Type: graphql.Int},
		"plu":           &graphql.Field{Type: graphql.String},
		"barcode":       &graphql.Field{Type: graphql.String},
		"sku":           &graphql.Field{Type: graphql.String},
		"imageUrl":      &graphql.Field{Type: graphql.String},
		"blurhash":      &graphql.Field{Type: graphql.String},
		"taxCategory":   &graphql.Field{Type: graphql.String},
		"isWeightBased": &graphql.Field{Type: graphql.Boolean},
		"weightUnit":    &graphql.Field{Type: graphql.String},
		"isActive":      &graphql.Field{Type: graphql.Boolean},
		"metadata":      &graphql.Field{Type: graphql.String},
	},
})

var orderConnectionType = graphql.NewObject(graphql.ObjectConfig{
	Name: "OrderConnection",
	Fields: graphql.Fields{
		"nodes": &graphql.Field{Type: graphql.NewList(orderType)},
		"total": &graphql.Field{Type: graphql.Int},
	},
})

var orderType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Order",
	Fields: graphql.Fields{
		"id":            &graphql.Field{Type: graphql.ID},
		"storeId":       &graphql.Field{Type: graphql.ID},
		"kioskId":       &graphql.Field{Type: graphql.ID},
		"cartId":        &graphql.Field{Type: graphql.ID},
		"orderNumber":   &graphql.Field{Type: graphql.String},
		"status":        &graphql.Field{Type: graphql.String},
		"subtotalCents": &graphql.Field{Type: graphql.Int},
		"taxCents":      &graphql.Field{Type: graphql.Int},
		"discountCents": &graphql.Field{Type: graphql.Int},
		"totalCents":    &graphql.Field{Type: graphql.Int},
		"items":         &graphql.Field{Type: graphql.NewList(orderItemType)},
		"paidAt":        &graphql.Field{Type: graphql.String},
		"fulfilledAt":   &graphql.Field{Type: graphql.String},
		"cancelledAt":   &graphql.Field{Type: graphql.String},
		"createdAt":     &graphql.Field{Type: graphql.String},
	},
})

var orderItemType = graphql.NewObject(graphql.ObjectConfig{
	Name: "OrderItem",
	Fields: graphql.Fields{
		"id":                 &graphql.Field{Type: graphql.ID},
		"orderId":            &graphql.Field{Type: graphql.ID},
		"itemId":             &graphql.Field{Type: graphql.ID},
		"nameSnapshot":       &graphql.Field{Type: graphql.String},
		"priceCentsSnapshot": &graphql.Field{Type: graphql.Int},
		"quantity":           &graphql.Field{Type: graphql.Int},
		"lineTotalCents":     &graphql.Field{Type: graphql.Int},
	},
})

var inventoryType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Inventory",
	Fields: graphql.Fields{
		"id":                &graphql.Field{Type: graphql.ID},
		"storeId":           &graphql.Field{Type: graphql.ID},
		"itemId":            &graphql.Field{Type: graphql.ID},
		"quantityAvailable": &graphql.Field{Type: graphql.Int},
		"quantityReserved":  &graphql.Field{Type: graphql.Int},
		"quantityOnOrder":   &graphql.Field{Type: graphql.Int},
		"reorderPoint":      &graphql.Field{Type: graphql.Int},
		"reorderQuantity":   &graphql.Field{Type: graphql.Int},
		"location":          &graphql.Field{Type: graphql.String},
		"lastCountedAt":     &graphql.Field{Type: graphql.String},
	},
})

var userType = graphql.NewObject(graphql.ObjectConfig{
	Name: "User",
	Fields: graphql.Fields{
		"id":          &graphql.Field{Type: graphql.ID},
		"tenantId":    &graphql.Field{Type: graphql.ID},
		"email":       &graphql.Field{Type: graphql.String},
		"name":        &graphql.Field{Type: graphql.String},
		"roleId":      &graphql.Field{Type: graphql.ID},
		"roleName":    &graphql.Field{Type: graphql.String},
		"isActive":    &graphql.Field{Type: graphql.Boolean},
		"lastLoginAt": &graphql.Field{Type: graphql.String},
		"createdAt":   &graphql.Field{Type: graphql.String},
	},
})

var roleType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Role",
	Fields: graphql.Fields{
		"id":          &graphql.Field{Type: graphql.ID},
		"tenantId":    &graphql.Field{Type: graphql.ID},
		"name":        &graphql.Field{Type: graphql.String},
		"description": &graphql.Field{Type: graphql.String},
		"isSystem":    &graphql.Field{Type: graphql.Boolean},
	},
})

// MustAdmin extracts admin claims from context and returns an error if missing.
func MustAdmin(ctx context.Context) (*auth.Claims, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return nil, fmt.Errorf("unauthorized")
	}
	if !claims.IsAdmin {
		return nil, fmt.Errorf("admin access required")
	}
	return claims, nil
}
