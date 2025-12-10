package main

import (
	"log/slog"
	"os"

	"github.com/aryankumar/fleet/internal/cli"
	"github.com/aryankumar/fleet/internal/util"
)

func main() {
	// Setup signal handling for graceful shutdown
	ctx := util.SetupSignalHandler()

	// Execute the CLI
	if err := cli.Execute(ctx); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
