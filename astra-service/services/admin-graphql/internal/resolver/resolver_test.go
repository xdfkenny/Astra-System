package resolver

import (
	"context"
	"testing"

	"github.com/astra-systems/astra-service/services/admin-graphql/internal/auth"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/repository"
	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaMenus(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetCategories([]repository.Category{
		{CategoryID: "cat-1", StoreID: "store-1", Name: "Drinks", DisplayOrder: 1, IsActive: true},
	})
	repo.SetItems([]repository.Item{
		{ItemID: "item-1", StoreID: "store-1", CategoryID: "cat-1", Name: "Coffee", PriceCents: 399, IsActive: true},
	})

	schema, err := NewSchema(repo)
	require.NoError(t, err)

	ctx := contextWithAdmin(context.Background())
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: `query { menus(storeId: "store-1") { storeId categories { id name } items { id name priceCents } } }`,
		Context:       ctx,
	})
	require.Empty(t, result.Errors, "%v", result.Errors)

	data := result.Data.(map[string]interface{})
	menus := data["menus"].(map[string]interface{})
	assert.Equal(t, "store-1", menus["storeId"])
	cats := menus["categories"].([]interface{})
	assert.Len(t, cats, 1)
}

func TestSchemaOrders(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetOrders([]repository.Order{
		{OrderID: "order-1", StoreID: "store-1", KioskID: "kiosk-1", OrderNumber: "1001", Status: "pending", TotalCents: 500},
	})

	schema, err := NewSchema(repo)
	require.NoError(t, err)

	ctx := contextWithAdmin(context.Background())
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: `query { orders(storeId: "store-1") { total nodes { id orderNumber status totalCents } } }`,
		Context:       ctx,
	})
	require.Empty(t, result.Errors, "%v", result.Errors)

	data := result.Data.(map[string]interface{})
	orders := data["orders"].(map[string]interface{})
	assert.EqualValues(t, 1, orders["total"])
}

func TestSchemaInventory(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetInventory([]repository.Inventory{
		{InventoryID: "inv-1", StoreID: "store-1", ItemID: "item-1", QuantityAvailable: 10},
	})

	schema, err := NewSchema(repo)
	require.NoError(t, err)

	ctx := contextWithAdmin(context.Background())
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: `query { inventory(storeId: "store-1") { id itemId quantityAvailable } }`,
		Context:       ctx,
	})
	require.Empty(t, result.Errors, "%v", result.Errors)

	data := result.Data.(map[string]interface{})
	inv := data["inventory"].([]interface{})
	assert.Len(t, inv, 1)
}

func TestSchemaUsersAndRoles(t *testing.T) {
	repo := repository.NewMemoryRepository()
	repo.SetUsers([]repository.User{
		{UserID: "user-1", TenantID: "tenant-1", Email: "admin@example.com", Name: "Admin", RoleName: "admin"},
	})
	repo.SetRoles([]repository.Role{
		{RoleID: "role-1", TenantID: "tenant-1", Name: "admin"},
	})

	schema, err := NewSchema(repo)
	require.NoError(t, err)

	ctx := contextWithAdmin(context.Background())
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: `query { users(tenantId: "tenant-1") { id email roleName } roles(tenantId: "tenant-1") { id name } }`,
		Context:       ctx,
	})
	require.Empty(t, result.Errors, "%v", result.Errors)

	data := result.Data.(map[string]interface{})
	assert.Len(t, data["users"].([]interface{}), 1)
	assert.Len(t, data["roles"].([]interface{}), 1)
}

func contextWithAdmin(ctx context.Context) context.Context {
	return auth.NewContextWithClaims(ctx, &auth.Claims{Subject: "admin-1", IsAdmin: true, TenantID: "tenant-1"})
}
