package cart

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/i18n"
	"backend/internal/platform/middleware"
	"backend/internal/session"
)

// RegisterRoutes mounts the cart endpoints on the /api/v1 group
// (docs/API.md §2). All writes are state-changing and already pass
// through the Origin validation middleware.
func RegisterRoutes(router fiber.Router, svc *Service, sessions *session.Service) {
	router.Get("/cart", getCart(svc))
	router.Post("/cart/items", addItem(svc, sessions))
	router.Patch("/cart/items/:id", updateItem(svc))
	router.Delete("/cart/items/:id", removeItem(svc))
}

func getCart(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cart, err := svc.GetCart(c.Context(), session.TokenFrom(c), middleware.LocaleFrom(c))
		if err != nil {
			return err
		}
		// Reads never issue a session cookie (§4.5).
		return c.JSON(cart)
	}
}

func addItem(svc *Service, sessions *session.Service) fiber.Handler {
	type request struct {
		VariantID int `json:"variantId"`
		Quantity  int `json:"quantity"`
	}
	return func(c *fiber.Ctx) error {
		var req request
		if err := c.BodyParser(&req); err != nil {
			return middleware.SendError(c, fiber.StatusBadRequest, "VALIDATION_ERROR",
				i18n.Message(middleware.LocaleFrom(c), "INVALID_REQUEST"))
		}

		cart, newToken, err := svc.AddItem(c.Context(), session.TokenFrom(c),
			int64(req.VariantID), req.Quantity, middleware.LocaleFrom(c))
		if err != nil {
			return cartError(c, err)
		}
		// The cookie is set only when this write created the session.
		if newToken != "" {
			c.Cookie(sessions.Cookie(newToken))
		}
		return c.JSON(cart)
	}
}

func updateItem(svc *Service) fiber.Handler {
	type request struct {
		Quantity int `json:"quantity"`
	}
	return func(c *fiber.Ctx) error {
		var req request
		if err := c.BodyParser(&req); err != nil {
			return middleware.SendError(c, fiber.StatusBadRequest, "VALIDATION_ERROR",
				i18n.Message(middleware.LocaleFrom(c), "INVALID_REQUEST"))
		}

		lineID, err := c.ParamsInt("id")
		if err != nil || lineID <= 0 {
			return validationResponse(c, []FieldError{{Field: "id", Code: "NOT_FOUND"}})
		}
		cart, err := svc.UpdateItem(c.Context(), session.TokenFrom(c),
			int64(lineID), int64(req.Quantity), middleware.LocaleFrom(c))
		if err != nil {
			return cartError(c, err)
		}
		return c.JSON(cart)
	}
}

func removeItem(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		lineID, err := c.ParamsInt("id")
		if err != nil || lineID <= 0 {
			return validationResponse(c, []FieldError{{Field: "id", Code: "NOT_FOUND"}})
		}
		cart, err := svc.RemoveItem(c.Context(), session.TokenFrom(c),
			int64(lineID), middleware.LocaleFrom(c))
		if err != nil {
			return cartError(c, err)
		}
		return c.JSON(cart)
	}
}

// cartError maps service errors to §4.3 envelopes.
func cartError(c *fiber.Ctx, err error) error {
	locale := middleware.LocaleFrom(c)

	var validation *ValidationError
	if errors.As(err, &validation) {
		return validationResponse(c, validation.Fields)
	}

	var stock *StockExceeded
	if errors.As(err, &stock) {
		return middleware.SendErrorWithDetails(c, fiber.StatusConflict,
			"STOCK_UNAVAILABLE", i18n.Message(locale, "STOCK_UNAVAILABLE"),
			[]map[string]any{{
				"variantId":   stock.VariantID,
				"productName": stock.ProductName,
				"requested":   stock.Requested,
				"available":   stock.Available,
			}})
	}

	if errors.Is(err, ErrNotFound) {
		return middleware.SendInfraError(c, fiber.StatusNotFound, "NOT_FOUND")
	}
	return err
}

func validationResponse(c *fiber.Ctx, fields []FieldError) error {
	locale := middleware.LocaleFrom(c)
	details := make([]map[string]any, 0, len(fields))
	for _, f := range fields {
		details = append(details, map[string]any{
			"field":   f.Field,
			"code":    f.Code,
			"message": i18n.Message(locale, f.Code),
		})
	}
	return middleware.SendErrorWithDetails(c, fiber.StatusBadRequest,
		"VALIDATION_ERROR", i18n.Message(locale, "VALIDATION_ERROR"), details)
}
