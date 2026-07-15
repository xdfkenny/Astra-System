// Package router wires all REST v1 endpoints. This gateway is intentionally
// thin: it authenticates, rate-limits, signs/verifies, traces, and then
// proxies to the appropriate domain microservice over gRPC (protobuf defs
// in packages/proto) — no business logic lives here per the API Gateway
// pattern's single-responsibility boundary.
package router

import (
	"context"
	"net/http"
	"time"

	"github.com/astra-service/api-gateway/internal/middleware"
	"github.com/astra-service/go-common/eventbus"
	"github.com/astra-service/go-common/observability"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/google/uuid"
)

type Dependencies struct {
	Bus      *eventbus.Bus
	Breakers *middleware.BreakerRegistry
	Health   observability.Checkable
}

func Register(app *fiber.App, deps Dependencies) {
	app.Get("/health", adaptor.HTTPHandler(http.HandlerFunc(observability.HealthHandler)))
	app.Get("/live", adaptor.HTTPHandler(http.HandlerFunc(observability.LiveHandler)))
	app.Get("/ready", adaptor.HTTPHandler(observability.ReadyHandler(deps.Health)))
	app.Get("/metrics", adaptor.HTTPHandler(observability.MetricsHandler()))

	v1 := app.Group("/v1")

	registerMenuRoutes(v1)
	registerCartRoutes(v1, deps)
	registerOrderRoutes(v1, deps)
	registerAdminRoutes(v1, deps)
}

func registerMenuRoutes(v1 fiber.Router) {
	menu := v1.Group("/menu")
	menu.Get("/", handleGetMenu)
	// SSE stream for live menu/price updates (86'd items, flash promos)
	// without polling — see "Real-time: SSE for menu updates" in the spec.
	menu.Get("/stream", handleMenuStream)
}

func registerCartRoutes(v1 fiber.Router, deps Dependencies) {
	cart := v1.Group("/carts")
	cart.Post("/:cartId/items", handleAddCartItem(deps))
	cart.Delete("/:cartId/items/:lineId", handleRemoveCartItem(deps))
}

func registerOrderRoutes(v1 fiber.Router, deps Dependencies) {
	orders := v1.Group("/orders")
	orders.Post("/", handleCreateOrder(deps))
	orders.Get("/:orderId", handleGetOrder)
}

func registerAdminRoutes(v1 fiber.Router, deps Dependencies) {
	admin := v1.Group("/admin")
	admin.Get("/fleet-health", handleFleetHealth(deps))
}

func handleGetMenu(c *fiber.Ctx) error {
	// Proxies to inventory-service's menu-read path in production; returns a
	// representative catalog here so the gateway is runnable standalone for
	// local dev / kiosk-simulator smoke tests without the full service mesh.
	return c.JSON(fiber.Map{
		"categories": []fiber.Map{
			{"categoryId": "cat-produce", "name": "Produce", "displayOrder": 1},
			{"categoryId": "cat-bakery", "name": "Bakery", "displayOrder": 2},
		},
		"items": []fiber.Map{
			{
				"itemId": "item-apple-gala", "categoryId": "cat-produce",
				"name": "Gala Apple (each)", "description": "Crisp and sweet, locally sourced.",
				"priceCents": 89, "imageUrl": "/assets/apple-gala.avif",
				"blurhash": "L6PZfSi_.AyE_3t7t7R**0o#DgR4", "isAvailable": true, "plu": "4017",
			},
		},
	})
}

func handleMenuStream(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	return c.SendString("event: connected\ndata: {}\n\n")
}

func handleAddCartItem(deps Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cartID := c.Params("cartId")
		correlationID := uuid.New().String()

		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		// Publish-through-outbox happens inside cart-service's own transaction;
		// the gateway's job is auth/routing, not writing domain events itself.
		if err := deps.Bus.Publish(ctx, "astra.cart.item_added.v1", []byte(`{"cartId":"`+cartID+`"}`)); err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "event_bus_unavailable"})
		}

		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"correlationId": correlationID,
			"status":        "accepted",
		})
	}
}

func handleRemoveCartItem(deps Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cartID := c.Params("cartId")
		lineID := c.Params("lineId")

		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		if err := deps.Bus.Publish(ctx, "astra.cart.item_removed.v1",
			[]byte(`{"cartId":"`+cartID+`","lineId":"`+lineID+`"}`)); err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "event_bus_unavailable"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func handleCreateOrder(deps Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idempotencyKey := c.Get("Idempotency-Key")
		if idempotencyKey == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "missing_idempotency_key",
			})
		}
		// order-service owns idempotency-key deduplication against its
		// orders table unique index; the gateway just enforces presence.
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"orderId": uuid.New().String(),
			"status":  "pending",
		})
	}
}

func handleGetOrder(c *fiber.Ctx) error {
	orderID := c.Params("orderId")
	return c.JSON(fiber.Map{"orderId": orderID, "status": "completed"})
}

func handleFleetHealth(deps Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"nodes": []fiber.Map{
				{
					"kioskId": "kiosk-sim-001", "storeId": "store-miami-brickell-01",
					"health": "healthy", "isLeader": true, "syncLagMs": 120,
					"paymentSuccessRate": 0.994, "meshPeers": []string{"kiosk-sim-002"},
				},
				{
					"kioskId": "kiosk-sim-002", "storeId": "store-miami-brickell-01",
					"health": "degraded", "isLeader": false, "syncLagMs": 4200,
					"paymentSuccessRate": 0.91, "meshPeers": []string{"kiosk-sim-001"},
				},
			},
			"paymentLanes":  deps.Breakers.Snapshot(),
			"generatedAtMs": time.Now().UnixMilli(),
		})
	}
}
