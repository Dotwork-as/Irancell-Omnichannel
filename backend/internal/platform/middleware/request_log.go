package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestLogger writes one structured log line per request (request
// id, method, path, status, latency). Mount after the request-id
// middleware, whose locals key "requestid" is fiber's default.
func RequestLogger(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		if log != nil {
			log.Info("http_request",
				"request_id", c.Locals("requestid"),
				"method", c.Method(),
				"path", c.Path(),
				"status", c.Response().StatusCode(),
				"latency_ms", float64(time.Since(start).Microseconds())/1000.0,
			)
		}
		return err
	}
}
