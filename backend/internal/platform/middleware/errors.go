// Package middleware hosts the cross-cutting HTTP middleware of the
// backend (locale resolution, Origin validation, request logging) and
// the API error envelope shared by every handler (docs/API.md §3).
// Localized message texts live in platform/i18n.
package middleware

import (
	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/i18n"
)

// apiError and errorBody model the §4.3 envelope:
// {"error":{"code":..,"message":..,"details":..}}.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type errorBody struct {
	Error apiError `json:"error"`
}

// SendError writes a §4.3 error response with the given status and
// machine-readable code.
func SendError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(errorBody{Error: apiError{Code: code, Message: message}})
}

// SendErrorWithDetails writes a §4.3 error response with the optional
// details payload (§4.3: shape depends on the code).
func SendErrorWithDetails(c *fiber.Ctx, status int, code, message string, details any) error {
	return c.Status(status).JSON(errorBody{
		Error: apiError{Code: code, Message: message, Details: details},
	})
}

// SendInfraError writes a §4.3 error response for an infrastructure
// code, with the message localized from the request locale.
func SendInfraError(c *fiber.Ctx, status int, code string) error {
	return SendError(c, status, code, i18n.Message(LocaleFrom(c), code))
}
