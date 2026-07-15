package middleware

import (
	"bytes"
	"io"
	"strconv"

	"github.com/astra-service/go-common/security"
	"github.com/gofiber/fiber/v2"
)

// RequireSignedRequest enforces the zero-trust boundary: every mutating
// request from a kiosk (POST/PUT/PATCH/DELETE) must carry an HMAC-SHA256
// signature proving possession of that kiosk's provisioned key, in addition
// to TLS. This defends against a compromised reverse proxy or a
// misconfigured internal network segment that would otherwise let an
// attacker replay/forge kiosk traffic once past the network perimeter.
//
// Header contract:
//
//	X-Astra-Timestamp: <unix seconds>
//	X-Astra-Signature: <hex hmac-sha256>
func RequireSignedRequest(resolveKey func(kioskID string) ([]byte, bool)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead {
			return c.Next() // reads are authenticated by session/JWT, not per-request HMAC
		}

		kioskID := c.Get("X-Astra-Kiosk-Id")
		timestampStr := c.Get("X-Astra-Timestamp")
		signature := c.Get("X-Astra-Signature")

		if kioskID == "" || timestampStr == "" || signature == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing_signature_headers",
			})
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_timestamp"})
		}

		key, ok := resolveKey(kioskID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unknown_kiosk"})
		}

		// Fiber's body is already fully buffered at this point (no streaming
		// uploads in this API), so re-reading it for hashing is safe and cheap
		// at the payload sizes involved (cart/order JSON, never binary blobs).
		body := c.Body()
		bodyHash := security.Sha256Hex(body)

		if err := security.VerifyRequest(key, method, c.Path(), timestamp, bodyHash, signature); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":  "signature_verification_failed",
				"detail": err.Error(),
			})
		}

		// Restore body reader for downstream handlers (defensive; Fiber's
		// c.Body() doesn't consume, but this guards against future middleware
		// that switches to a streaming body reader).
		c.Request().SetBodyStream(io.NopCloser(bytes.NewReader(body)), len(body))

		return c.Next()
	}
}
