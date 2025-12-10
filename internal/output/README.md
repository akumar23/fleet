# Output Package

The output package provides flexible formatters for displaying Fleet CLI command results. It supports multiple output formats (table, JSON, YAML) with automatic color support and TTY detection.

## Features

- **Multiple Formats**: Table (kubectl-style), JSON, and YAML
- **Color Support**: Automatic TTY detection with configurable color scheme
- **Flexible Options**: No-color, no-headers, and wide mode support
- **Multi-Cluster Support**: Aggregate results from multiple clusters with summary statistics
- **Production Ready**: 93.3% test coverage with comprehensive unit tests

## Quick Start

```go
import "github.com/aryankumar/fleet/internal/output"

// Create a formatter
formatter := output.NewFormatter(output.FormatTable)

// Format multi-cluster results
results := []executor.Result{...}
formatter.FormatMultiCluster(os.Stdout, results)
```

## Formatters

### Table Formatter (kubectl-style)

Borderless, tab-separated tables with optional color highlighting:

```go
formatter := output.NewFormatter(output.FormatTable)
formatter.FormatMultiCluster(os.Stdout, results)
```

Output:
```
CLUSTER      STATUS    DURATION
production   Success   150ms
staging      Success   100ms

Summary: 2 successful, 0 failed, avg=125ms
```

#### Wide Mode

Add a DATA column for additional details:

```go
formatter := output.NewFormatter(
    output.FormatTable,
    output.WithWide(true),
)
```

#### No Headers

Disable column headers:

```go
formatter := output.NewFormatter(
    output.FormatTable,
    output.WithNoHeaders(true),
)
```

### JSON Formatter

Clean, indented JSON output for scripting:

```go
formatter := output.NewFormatter(output.FormatJSON)
formatter.FormatMultiCluster(os.Stdout, results)
```

Output:
```json
[
  {
    "cluster": "production",
    "status": "success",
    "duration": "150ms",
    "data": {"pods": "10"}
  },
  {
    "cluster": "staging",
    "status": "failed",
    "duration": "50ms",
    "error": "connection timeout"
  }
]
```

### YAML Formatter

Human-readable YAML output:

```go
formatter := output.NewFormatter(output.FormatYAML)
formatter.FormatMultiCluster(os.Stdout, results)
```

Output:
```yaml
- cluster: production
  status: success
  duration: 150ms
  data:
    pods: "10"
- cluster: staging
  status: failed
  duration: 50ms
  error: connection timeout
```

## Options

Configure formatters using functional options:

```go
formatter := output.NewFormatter(
    output.FormatTable,
    output.WithNoColor(true),    // Disable colors
    output.WithNoHeaders(true),  // Disable headers
    output.WithWide(true),       // Enable wide mode
)
```

Available options:

- `WithNoColor(bool)` - Disable color output
- `WithNoHeaders(bool)` - Disable table headers
- `WithWide(bool)` - Enable wide output mode

## Color Support

Colors are automatically enabled for TTY outputs and can be disabled with the `WithNoColor` option.

### Color Scheme

- **Cluster Names**: Cyan, Bold
- **Success Status**: Green
- **Error Messages**: Red, Bold
- **Warnings**: Yellow
- **Headers**: White, Bold
- **Durations**: Blue

### TTY Detection

The package automatically detects if output is going to a terminal:

```go
// Colors enabled if stdout is a TTY
formatter := output.NewFormatter(output.FormatTable)
formatter.FormatMultiCluster(os.Stdout, results)

// Colors disabled for pipes/redirects
./fleet list | less  // No colors in less
```

## Architecture

### Interface

All formatters implement the `Formatter` interface:

```go
type Formatter interface {
    Format(w io.Writer, data interface{}) error
    FormatMultiCluster(w io.Writer, results []executor.Result) error
}
```

### Factory Pattern

Use `NewFormatter` to create formatters:

```go
func NewFormatter(format Format, opts ...Option) Formatter
```

Supported formats:
- `FormatTable` - kubectl-style table (default)
- `FormatJSON` - JSON output
- `FormatYAML` - YAML output

## Testing

Run the test suite:

```bash
go test ./internal/output/... -v -cover
```

Coverage: 93.3%

## Examples

See `example_test.go` for comprehensive usage examples:

- Basic table formatting
- JSON and YAML output
- Wide mode with additional columns
- No-headers mode
- Color configuration

## Integration

The output package integrates with the executor package to format multi-cluster results:

```go
import (
    "github.com/aryankumar/fleet/internal/executor"
    "github.com/aryankumar/fleet/internal/output"
)

// Execute tasks across clusters
pool := executor.NewPool(5, logger)
// ... submit tasks ...
results := pool.Execute(ctx)

// Format the results
formatter := output.NewFormatter(output.FormatTable)
formatter.FormatMultiCluster(os.Stdout, results)
```

## Best Practices

1. **Use appropriate formats**: Table for humans, JSON/YAML for scripts
2. **Respect no-color**: Check `NO_COLOR` environment variable if needed
3. **Handle errors**: Always check error returns from Format methods
4. **Buffer output**: Use `bytes.Buffer` for testing or string output
5. **Configure wisely**: Use options to customize behavior for different contexts

## Dependencies

- `github.com/olekukonko/tablewriter` - Table rendering
- `github.com/fatih/color` - Terminal color support
- `github.com/mattn/go-isatty` - TTY detection
- `gopkg.in/yaml.v3` - YAML encoding
