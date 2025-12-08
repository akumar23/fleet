package util

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandler creates a context that is cancelled on receiving SIGINT or SIGTERM.
// A second signal will force immediate exit.
func SetupSignalHandler() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	// Create channel to receive OS signals
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received shutdown signal", "signal", sig.String())
		cancel()

		// Second signal forces immediate exit
		sig = <-sigCh
		slog.Warn("received second shutdown signal, forcing exit", "signal", sig.String())
		os.Exit(1)
	}()

	return ctx
}
