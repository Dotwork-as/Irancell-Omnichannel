package catalog

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/middleware"
)

// RegisterRoutes mounts the catalog endpoints on the /api/v1 group:
// GET /products and GET /products/:slug (docs/API.md §2). Reads only —
// no Origin header required.
func RegisterRoutes(router fiber.Router, svc *Service) {
	router.Get("/products", list(svc))
	router.Get("/products/:slug", detail(svc))
}

type listResponse struct {
	Items []ProductSummary `json:"items"`
}

func list(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, err := svc.List(c.Context(), c.Query("q"), middleware.LocaleFrom(c))
		if err != nil {
			return err // falls through to the §4.3 500 INTERNAL_ERROR handler
		}
		return c.JSON(listResponse{Items: items})
	}
}

func detail(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		product, err := svc.GetBySlug(c.Context(), c.Params("slug"), middleware.LocaleFrom(c))
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return middleware.SendInfraError(c, fiber.StatusNotFound, "NOT_FOUND")
			}
			return err
		}
		return c.JSON(product)
	}
}
