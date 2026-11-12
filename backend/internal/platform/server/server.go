// Package server assembles the Fiber application: the middleware
// chain, the /api/v1 router and the graceful lifecycle. Business
// routers (catalog, cart, checkout, …) mount onto /api/v1.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"backend/internal/cart"
	"backend/internal/catalog"
	"backend/internal/checkout"
	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/i18n"
	"backend/internal/platform/middleware"
	"backend/internal/session"
)

// New builds the Fiber app with the middleware chain
// (docs/API.md §3/§5): panic recovery → request id → request logging
// → CORS → locale → Origin validation → catch-all 404.
//
// The catalog module mounts the product routes (GET /api/v1/products,
// GET /api/v1/products/{slug}); the cart module mounts the cart
// routes (GET /api/v1/cart, POST/PATCH/DELETE /api/v1/cart/items…)
// with guest sessions; shipping + checkout mount last.
// GET /api/health is a liveness-only
// probe: HTTP 200 {"status":"ok"} while the process runs. It makes
// NO readiness claim about the database or any external service and
// exposes no configuration.
func New(cfg config.Config, log *slog.Logger, db *database.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "irancell-omnichannel-backend",
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler,
		// Resource limits (hardening): small JSON API — bodies are a
		// few KB; timeouts bound slow-client resource use.
		BodyLimit:    256 * 1024,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		// Never trust X-Forwarded-For: with an empty trusted-proxy list
		// c.IP() is always the connection address, so rate limiting and
		// logging cannot be spoofed by header injection.
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{},
	})

	app.Use(middleware.SecurityHeaders())
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.RequestLogger(log))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.CORSAllowedOrigins, ","),
		AllowCredentials: true,
		AllowMethods: strings.Join([]string{
			http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete,
		}, ","),
		AllowHeaders: "Content-Type,Accept-Language",
		MaxAge:       600,
	}))
	app.Use(middleware.Locale())
	app.Use(middleware.OriginValidation(cfg.CORSAllowedOrigins))

	// GET /api/health — liveness probe (outside /api/v1, no session,
	// no auth): 200 {"status":"ok"} while the process is running.
	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Rate limiting (hardening): a stricter per-IP cap on the sensitive
	// checkout write path; the generous global API cap is mounted on
	// the /api/v1 group below. In-memory per instance — multi-instance
	// deployments need a shared store (documented limitation).
	app.All("/api/v1/checkout", middleware.RateLimit(cfg.RateLimitCheckoutMax, time.Minute))

	// /api/v1 — business routers mount here: catalog, cart (with
	// guest sessions) and shipping + checkout.
	// Global per-IP rate limit first (hardening): generous cap over the
	// whole versioned API.
	api := app.Group("/api/v1")
	api.Use(middleware.RateLimit(cfg.RateLimitMax, time.Minute))
	catalog.RegisterRoutes(api, catalog.New(db))
	sessions := session.New(db, cfg.CookieSecure)
	cart.RegisterRoutes(api, cart.New(db, sessions), sessions)
	checkout.RegisterRoutes(api, checkout.New(db, sessions, cfg), sessions)

	// Catch-all (mounted last): anything no route matched gets the
	// §4.3 envelope. Note: fiber v2 reports a method mismatch on a
	// registered path as 404 (v3 would 405) — documented behavior.
	app.Use(notFoundHandler)

	return app
}

// Run serves HTTP until ctx is cancelled (SIGINT/SIGTERM via
// signal.NotifyContext in main), then drains in-flight requests with
// a 10s graceful-shutdown window.
func Run(ctx context.Context, app *fiber.App, port string, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(":" + port)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		if log != nil {
			log.Info("shutting down")
		}
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		return app.ShutdownWithContext(shutdownCtx)
	}
}

func errorHandler(c *fiber.Ctx, err error) error {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		switch fiberErr.Code {
		case http.StatusNotFound:
			return middleware.SendInfraError(c, http.StatusNotFound, "NOT_FOUND")
		case http.StatusMethodNotAllowed:
			return middleware.SendInfraError(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		case http.StatusBadRequest, fiber.StatusUnprocessableEntity:
			// Malformed JSON bodies (fiber v2 parses failures as 422)
			// become the §4.3 VALIDATION_ERROR envelope.
			return middleware.SendError(c, http.StatusBadRequest, "VALIDATION_ERROR",
				i18n.Message(middleware.LocaleFrom(c), "INVALID_REQUEST"))
		default:
			// Other framework errors (e.g. body too large) keep
			// their status but flow through the same envelope.
			return middleware.SendError(c, fiberErr.Code, http.StatusText(fiberErr.Code), fiberErr.Message)
		}
	}
	return middleware.SendInfraError(c, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func notFoundHandler(c *fiber.Ctx) error {
	return middleware.SendInfraError(c, http.StatusNotFound, "NOT_FOUND")
}
