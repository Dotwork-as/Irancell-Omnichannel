package checkout

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/i18n"
	"backend/internal/platform/middleware"
	"backend/internal/session"
)

// RegisterRoutes mounts GET /api/v1/shipping-methods and
// POST /api/v1/checkout on the /api/v1 group (docs/API.md §2). Checkout
// is state-changing and already passes through the Origin validation
// middleware.
func RegisterRoutes(router fiber.Router, svc *Service, sessions *session.Service) {
	router.Get("/shipping-methods", shippingMethods(svc))
	router.Post("/checkout", checkout(svc, sessions))
}

type shippingResponse struct {
	Items []ShippingMethod `json:"items"`
}

func shippingMethods(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		methods, err := svc.ShippingMethods(c.Context(), middleware.LocaleFrom(c))
		if err != nil {
			return err
		}
		return c.JSON(shippingResponse{Items: methods})
	}
}

func checkout(svc *Service, sessions *session.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req CheckoutRequest
		if err := c.BodyParser(&req); err != nil {
			return middleware.SendError(c, fiber.StatusBadRequest, "VALIDATION_ERROR",
				i18n.Message(middleware.LocaleFrom(c), "INVALID_REQUEST"))
		}

		receipt, err := svc.Checkout(c.Context(), session.TokenFrom(c), req, middleware.LocaleFrom(c))
		if err != nil {
			return checkoutError(c, err)
		}
		return c.Status(fiber.StatusCreated).JSON(receipt)
	}
}

// checkoutError maps service errors to §4.3 envelopes without ever
// exposing internal details.
func checkoutError(c *fiber.Ctx, err error) error {
	locale := middleware.LocaleFrom(c)

	var validation *ValidationError
	if errors.As(err, &validation) {
		details := make([]map[string]any, 0, len(validation.Fields))
		for _, f := range validation.Fields {
			details = append(details, map[string]any{
				"field":   f.Field,
				"code":    f.Code,
				"message": i18n.Message(locale, f.Code),
			})
		}
		return middleware.SendErrorWithDetails(c, fiber.StatusBadRequest,
			"VALIDATION_ERROR", i18n.Message(locale, "VALIDATION_ERROR"), details)
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

	var turnstileReq *TurnstileRequired
	if errors.As(err, &turnstileReq) {
		return middleware.SendError(c, fiber.StatusBadRequest, "TURNSTILE_REQUIRED",
			i18n.Message(locale, "TURNSTILE_REQUIRED"))
	}

	var turnstileFailed *TurnstileFailed
	if errors.As(err, &turnstileFailed) {
		return middleware.SendError(c, fiber.StatusForbidden, "TURNSTILE_FAILED",
			i18n.Message(locale, "TURNSTILE_FAILED"))
	}

	if errors.Is(err, ErrEmptyCart) {
		return middleware.SendInfraError(c, fiber.StatusConflict, "CART_EMPTY")
	}
	// Anything unexpected becomes a generic localized 500 — no
	// database or implementation details leak.
	return middleware.SendInfraError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR")
}
