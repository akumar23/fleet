package output

import (
	"io"

	"github.com/aryankumar/fleet/internal/executor"
)

// Format represents the output format type
type Format string

const (
	// FormatTable outputs data in a table format (kubectl-style)
	FormatTable Format = "table"
	// FormatJSON outputs data in JSON format
	FormatJSON Format = "json"
	// FormatYAML outputs data in YAML format
	FormatYAML Format = "yaml"
)

// Formatter defines the interface for output formatting
// All formatters must implement both single and multi-cluster output methods
type Formatter interface {
	// Format outputs a single data item to the writer
	Format(w io.Writer, data interface{}) error

	// FormatMultiCluster outputs multiple cluster results to the writer
	FormatMultiCluster(w io.Writer, results []executor.Result) error
}

// Option is a functional option for configuring formatters
type Option func(*Options)

// Options holds configuration for formatters
type Options struct {
	// NoColor disables color output
	NoColor bool

	// NoHeaders disables table headers
	NoHeaders bool

	// Wide enables wide output with additional columns
	Wide bool
}

// WithNoColor disables color output
func WithNoColor(noColor bool) Option {
	return func(o *Options) {
		o.NoColor = noColor
	}
}

// WithNoHeaders disables table headers
func WithNoHeaders(noHeaders bool) Option {
	return func(o *Options) {
		o.NoHeaders = noHeaders
	}
}

// WithWide enables wide output
func WithWide(wide bool) Option {
	return func(o *Options) {
		o.Wide = wide
	}
}

// NewFormatter creates a new formatter based on the specified format
func NewFormatter(format Format, opts ...Option) Formatter {
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	switch format {
	case FormatJSON:
		return NewJSONFormatter(options)
	case FormatYAML:
		return NewYAMLFormatter(options)
	case FormatTable:
		fallthrough
	default:
		return NewTableFormatter(options)
	}
}
