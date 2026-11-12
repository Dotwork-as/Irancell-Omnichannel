package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// OriginValidation enforces the approved CSRF posture (docs/API.md §5):
// every state-changing request (POST/PATCH/DELETE) must carry an
// Origin header that is in the allowed list; anything else is
// rejected with 403 FORBIDDEN_ORIGIN. Reads are exempt. Mount after
// Locale so the rejection message is localized.
func OriginValidation(allowed []string) fiber.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[o] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPatch, fiber.MethodDelete:
			origin := c.Get(fiber.HeaderOrigin)
			if _, ok := set[origin]; !ok {
				return SendInfraError(c, fiber.StatusForbidden, "FORBIDDEN_ORIGIN")
			}
		}
		return c.Next()
	}
}
