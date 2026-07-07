// Package routes wires all gateway HTTP routes. Public health, readiness and
// metrics endpoints bypass auth; all other traffic is proxied to downstream
// Astra services via HTTP reverse proxy or gRPC clients.
package routes

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	cartpb "github.com/astra-systems/astra-service/proto/gen/go/cart"
	menupb "github.com/astra-systems/astra-service/proto/gen/go/menu"
	"github.com/astra-systems/astra-service/services/gateway/internal/clients"
	"github.com/astra-systems/astra-service/services/gateway/internal/config"
	"github.com/astra-systems/astra-service/services/gateway/internal/health"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

//go:embed all:docs
var docsFS embed.FS

// Register wires handlers and proxies into the Fiber app.
func Register(app *fiber.App, cfg *config.Config, checker health.Checker, serviceClients *clients.Registry) {
	app.Get("/health", handleHealth)
	app.Get("/live", handleLive)
	app.Get("/ready", handleReady(checker))
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	registerDocs(app)
	registerServiceProxies(app, cfg)
	registerGRPCRoutes(app, serviceClients)
}

// handleHealth returns a simple health status.
//
//	@Summary	Health check
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/health [get]
func handleHealth(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok", "service": "astra-gateway"})
}

// handleLive returns a liveness status.
//
//	@Summary	Liveness check
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/live [get]
func handleLive(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "alive"})
}

// handleReady returns readiness only when all dependencies are reachable.
//
//	@Summary	Readiness check
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Router		/ready [get]
func handleReady(checker health.Checker) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := checker.Check(c.Context()); err != nil {
			logger.LogAttrs(c.Context(), slog.LevelWarn, "readiness_check_failed", slog.String("error", err.Error()))
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "not_ready",
				"detail": err.Error(),
			})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	}
}

func registerDocs(app *fiber.App) {
	sub, err := fs.Sub(docsFS, "docs")
	if err != nil {
		logger.LogAttrs(nil, slog.LevelError, "docs_embed_failed", slog.String("error", err.Error()))
		return
	}
	app.Get("/docs/*", adaptor.HTTPHandler(http.StripPrefix("/docs/", http.FileServer(http.FS(sub)))))
}

func registerServiceProxies(app *fiber.App, cfg *config.Config) {
	v1 := app.Group("/v1")
	for name, downstream := range cfg.Services {
		handler := proxyHandler(name, downstream.HTTPBaseURL)
		group := v1.Group("/" + name)
		group.Use(handler)
	}
}

func proxyHandler(name string, baseURL *url.URL) fiber.Handler {
	proxy := httputil.NewSingleHostReverseProxy(baseURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = baseURL.Host
		req.URL.Path = stripServicePrefix(req.URL.Path, name)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		logger.LogAttrs(req.Context(), slog.LevelError, "proxy_error",
			slog.String("service", name),
			slog.String("error", err.Error()),
		)
		w.WriteHeader(http.StatusBadGateway)
		if _, writeErr := w.Write([]byte(`{"error":"bad_gateway"}`)); writeErr != nil {
			logger.LogAttrs(req.Context(), slog.LevelError, "proxy_error_write_failed", slog.String("error", writeErr.Error()))
		}
	}
	return adaptor.HTTPHandler(proxy)
}

func stripServicePrefix(path, name string) string {
	prefix := "/v1/" + name
	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix)
	}
	return path
}

func registerGRPCRoutes(app *fiber.App, serviceClients *clients.Registry) {
	v1 := app.Group("/v1")
	v1.Get("/menu", handleGetMenu(serviceClients))
	v1.Get("/carts/:cartId", handleGetCart(serviceClients))
}

// handleGetMenu proxies a menu lookup to the downstream Menu gRPC service.
//
//	@Summary	Get menu
//	@Tags		menu
//	@Produce	json
//	@Param		store_id		query		string	false	"Store identifier"
//	@Param		include_inactive	query		boolean	false	"Include inactive items"
//	@Success	200	{object}	github.com/astra-systems/astra-service/proto/gen/go/menu.MenuResponse
//	@Failure	502	{object}	map[string]string
//	@Router		/v1/menu [get]
func handleGetMenu(serviceClients *clients.Registry) fiber.Handler {
	return func(c fiber.Ctx) error {
		if serviceClients == nil || serviceClients.Menu == nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "menu_service_unavailable"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		req := &menupb.MenuRequest{
			StoreId:         c.Query("store_id"),
			IncludeInactive: c.Query("include_inactive") == "true",
		}
		resp, err := serviceClients.Menu.GetMenu(ctx, req)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "menu_service_unavailable", "detail": err.Error()})
		}
		return c.JSON(protojsonMarshal(resp))
	}
}

// handleGetCart proxies a cart lookup to the downstream Cart gRPC service.
//
//	@Summary	Get cart
//	@Tags		cart
//	@Produce	json
//	@Param		cartId	path		string	true	"Cart identifier"
//	@Success	200	{object}	github.com/astra-systems/astra-service/proto/gen/go/cart.Cart
//	@Failure	502	{object}	map[string]string
//	@Router		/v1/carts/{cartId} [get]
func handleGetCart(serviceClients *clients.Registry) fiber.Handler {
	return func(c fiber.Ctx) error {
		if serviceClients == nil || serviceClients.Cart == nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "cart_service_unavailable"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		req := &cartpb.GetCartRequest{CartId: c.Params("cartId")}
		resp, err := serviceClients.Cart.GetCart(ctx, req)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "cart_service_unavailable", "detail": err.Error()})
		}
		return c.JSON(protojsonMarshal(resp))
	}
}

func protojsonMarshal(msg proto.Message) interface{} {
	b, err := protojson.Marshal(msg)
	if err != nil {
		return fiber.Map{"error": "marshal_failed", "detail": err.Error()}
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return fiber.Map{"error": "unmarshal_failed", "detail": err.Error()}
	}
	return out
}
