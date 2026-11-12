package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimit returns per-client-IP rate limiting middleware (fiber's
// in-memory limiter — single-instance; multi-instance deployments
// would need a shared store, documented as a known limitation).
// Rejections are localized §4.3 envelopes with Retry-After.
func RateLimit(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "60")
			return SendInfraError(c, fiber.StatusTooManyRequests, "RATE_LIMITED")
		},
	})
}
