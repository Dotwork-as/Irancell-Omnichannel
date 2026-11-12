package middleware

import (
	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/i18n"
)

// LocaleKey is the fiber Locals key holding the resolved locale.
const LocaleKey = "locale"

// Locale resolves the request locale from Accept-Language once per
// request and stores it under the LocaleKey locals (docs/API.md §1);
// resolution itself lives in platform/i18n.
func Locale() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(LocaleKey, i18n.ResolveAcceptLanguage(c.Get(fiber.HeaderAcceptLanguage)))
		return c.Next()
	}
}

// LocaleFrom returns the locale stored by Locale, defaulting to fa.
func LocaleFrom(c *fiber.Ctx) string {
	if v, ok := c.Locals(LocaleKey).(string); ok && i18n.Supported(v) {
		return v
	}
	return i18n.LocaleFa
}
