// Package logging configures the application's structured logger.
package logging

import (
	"io"
	"log/slog"
)

// New returns a JSON structured logger writing to w. The backend logs
// structured JSON from the first phase on so every later phase keeps
// one machine-readable log format.
func New(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
