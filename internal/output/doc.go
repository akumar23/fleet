// Package output provides formatters for displaying Fleet CLI command results.
//
// The package supports multiple output formats (table, JSON, YAML) and provides
// a unified interface for formatting both single-cluster and multi-cluster results.
//
// # Features
//
//   - Multiple output formats: table (kubectl-style), JSON, and YAML
//   - Color support with automatic TTY detection
//   - Configurable options (no-color, no-headers, wide mode)
//   - Multi-cluster result aggregation
//   - Automatic indentation and formatting
//
// # Basic Usage
//
//	// Create a table formatter
//	formatter := output.NewFormatter(output.FormatTable)
//
//	// Format single data item
//	data := map[string]interface{}{"key": "value"}
//	formatter.Format(os.Stdout, data)
//
//	// Format multi-cluster results
//	results := []executor.Result{...}
//	formatter.FormatMultiCluster(os.Stdout, results)
//
// # Options
//
// Formatters can be configured with functional options:
//
//	formatter := output.NewFormatter(
//	    output.FormatTable,
//	    output.WithNoColor(true),
//	    output.WithWide(true),
//	)
//
// # Formatters
//
// Table Formatter (kubectl-style):
//   - Borderless tables with tab-separated columns
//   - Optional color highlighting for status, errors, and cluster names
//   - Summary statistics for multi-cluster results
//   - Wide mode for additional data columns
//
// JSON Formatter:
//   - Clean, indented JSON output
//   - Suitable for scripting and automation
//   - Consistent structure for multi-cluster results
//
// YAML Formatter:
//   - Human-readable YAML output
//   - Proper indentation and formatting
//   - Compatible with kubectl-style workflows
//
// # Color Support
//
// Colors are automatically enabled for TTY outputs and can be disabled with:
//   - WithNoColor(true) option
//   - Non-TTY output (pipes, redirects)
//
// Color scheme:
//   - Cluster names: Cyan, Bold
//   - Success status: Green
//   - Error messages: Red, Bold
//   - Warnings: Yellow
//   - Headers: White, Bold
//   - Durations: Blue
package output
