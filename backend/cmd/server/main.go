// Command server runs the Irancell omnichannel backend API: it loads
// the configuration, opens SQLite (migrations + seeds apply
// automatically) and serves the middleware chain plus the business
// routers (catalog, cart, checkout) on /api/v1.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/logging"
	"backend/internal/platform/server"
	"backend/internal/seed"
)

func main() {
	logger := logging.New(os.Stdout)

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		logger.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("open database", "err", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.Migrate(ctx, database.Migrations); err != nil {
		logger.Error("migrate database", "err", err)
		os.Exit(1)
	}

	if err := seed.Run(ctx, db, cfg.SeedDemoOrders); err != nil {
		logger.Error("seed database", "err", err)
		os.Exit(1)
	}

	app := server.New(cfg, logger, db)
	logger.Info("backend starting",
		"port", cfg.Port,
		"database", cfg.DatabasePath,
		"cors_origins", cfg.CORSAllowedOrigins,
	)
	if err := server.Run(ctx, app, cfg.Port, logger); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
