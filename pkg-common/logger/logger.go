// Package logger builds the structured logger every Fusewall service uses.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// Options configures the logger. Nothing here is product specific.
type Options struct {
	App         string
	Environment string
	// JSON selects machine-readable output, used in production.
	JSON  bool
	Level slog.Level
	// Out defaults to stdout.
	Out io.Writer
}

// New returns a logger that stamps every line with the app and environment.
func New(opts Options) *slog.Logger {
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	handlerOpts := &slog.HandlerOptions{Level: opts.Level}

	var h slog.Handler
	if opts.JSON {
		h = slog.NewJSONHandler(out, handlerOpts)
	} else {
		h = slog.NewTextHandler(out, handlerOpts)
	}
	return slog.New(h).With("app", opts.App, "env", opts.Environment)
}
