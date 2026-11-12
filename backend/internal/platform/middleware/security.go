package middleware

import "github.com/gofiber/fiber/v2"

// SecurityHeaders sets the baseline HTTP security headers on every
// response (API JSON today; the Next.js server sets its own policy on
// HTML). CSP `default-src 'none'; frame-ancestors 'none'` is safe for
// pure-JSON endpoints and blocks framing/embedding entirely.
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		return c.Next()
	}
}
