package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astra-systems/astra-service/services/admin-graphql/internal/auth"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/config"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/repository"
	"github.com/astra-systems/astra-service/services/admin-graphql/internal/resolver"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleHealthz(t *testing.T) {
	cfg, _ := config.Load()
	schema, _ := resolver.NewSchema(repository.NewMemoryRepository())
	srv := New(cfg, schema)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.handleHealthz(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleGraphQLRequiresAuth(t *testing.T) {
	cfg, _ := config.Load()
	schema, err := resolver.NewSchema(repository.NewMemoryRepository())
	require.NoError(t, err)
	srv := New(cfg, schema)

	body, _ := json.Marshal(map[string]string{"query": "{ __typename }"})
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler := auth.Middleware(auth.Config{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience})(http.HandlerFunc(srv.handleGraphQL))
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleGraphQLAllowsAdmin(t *testing.T) {
	cfg, _ := config.Load()
	repo := repository.NewMemoryRepository()
	repo.SetCategories([]repository.Category{{CategoryID: "cat-1", StoreID: "store-1", Name: "Food", IsActive: true}})
	schema, err := resolver.NewSchema(repo)
	require.NoError(t, err)
	srv := New(cfg, schema)

	body, _ := json.Marshal(map[string]string{"query": `query { menus(storeId: "store-1") { storeId } }`})
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+signAdminToken(t, cfg))
	rec := httptest.NewRecorder()

	handler := auth.Middleware(auth.Config{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience})(http.HandlerFunc(srv.handleGraphQL))
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var result struct {
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.NotNil(t, result.Data["menus"])
}

func signAdminToken(t *testing.T, cfg *config.Config) string {
	t.Helper()
	return authTestToken(t, cfg, true)
}

func authTestToken(t *testing.T, cfg *config.Config, isAdmin bool) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":      cfg.JWTIssuer,
		"aud":      cfg.JWTAudience,
		"sub":      "admin-1",
		"tenant_id": "tenant-1",
		"role":     "admin",
		"is_admin": isAdmin,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(cfg.JWTSecret)
	require.NoError(t, err)
	return signed
}
