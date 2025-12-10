package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aryankumar/fleet/internal/cli/apply"
	"github.com/aryankumar/fleet/internal/cli/cluster"
	"github.com/aryankumar/fleet/internal/cli/delete"
	"github.com/aryankumar/fleet/internal/cli/get"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// Execute runs the root command with the provided context
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}

// newRootCmd creates the root command
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "fleet",
		Short: "Fleet - Multi-cluster Kubernetes management tool",
		Long: `Fleet is a powerful CLI tool for managing multiple Kubernetes clusters.
It provides a unified interface for executing commands, managing resources,
and monitoring across all your clusters.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(cmd)
		},
	}

	// Define persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fleet.yaml)")
	rootCmd.PersistentFlags().String("kubeconfig", "", "path to kubeconfig file (default is $HOME/.kube/config)")
	rootCmd.PersistentFlags().StringSlice("clusters", []string{}, "target clusters (comma-separated, empty means all)")
	rootCmd.PersistentFlags().StringP("output", "o", "", "output format (json, yaml, table)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output with debug logging")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().Duration("timeout", 30*time.Second, "timeout for operations")
	rootCmd.PersistentFlags().IntP("parallel", "p", 5, "number of parallel operations")

	// Bind flags to viper
	viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindPFlag("clusters", rootCmd.PersistentFlags().Lookup("clusters"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("parallel", rootCmd.PersistentFlags().Lookup("parallel"))

	// Add subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(cluster.NewClusterCmd())
	rootCmd.AddCommand(get.NewGetCmd())
	rootCmd.AddCommand(apply.NewApplyCmd())
	rootCmd.AddCommand(delete.NewDeleteCmd())

	return rootCmd
}

// initConfig initializes configuration and logging
func initConfig(cmd *cobra.Command) error {
	// Initialize viper configuration
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".fleet")
	}

	// Read environment variables
	viper.SetEnvPrefix("FLEET")
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Setup structured logging
	setupLogging(cmd)

	return nil
}

// setupLogging configures structured logging with slog
func setupLogging(cmd *cobra.Command) {
	verbose, _ := cmd.Flags().GetBool("verbose")
	noColor, _ := cmd.Flags().GetBool("no-color")

	// Set log level based on verbose flag
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if noColor {
		// Use JSON handler for no-color mode
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		// Use text handler for colored output
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	// Set default logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	if verbose {
		slog.Debug("verbose logging enabled")
		if viper.ConfigFileUsed() != "" {
			slog.Debug("loaded configuration", "file", viper.ConfigFileUsed())
		}
	}
}
