package output

import (
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

// ColorScheme provides color functions for different output elements
type ColorScheme struct {
	// ClusterName colors cluster names
	ClusterName func(format string, a ...interface{}) string

	// Success colors success status
	Success func(format string, a ...interface{}) string

	// Error colors error messages
	Error func(format string, a ...interface{}) string

	// Warning colors warning messages
	Warning func(format string, a ...interface{}) string

	// Header colors table headers
	Header func(format string, a ...interface{}) string

	// Duration colors duration values
	Duration func(format string, a ...interface{}) string

	// Disabled indicates if colors are disabled
	Disabled bool
}

// NewColorScheme creates a new color scheme
// Colors are automatically disabled for non-TTY outputs or when noColor is true
func NewColorScheme(w io.Writer, noColor bool) *ColorScheme {
	// Determine if we should use colors
	useColor := !noColor && isTTY(w)

	if !useColor {
		// Return scheme with no-op color functions
		return &ColorScheme{
			ClusterName: color.New().Sprintf,
			Success:     color.New().Sprintf,
			Error:       color.New().Sprintf,
			Warning:     color.New().Sprintf,
			Header:      color.New().Sprintf,
			Duration:    color.New().Sprintf,
			Disabled:    true,
		}
	}

	// Return scheme with actual colors
	return &ColorScheme{
		ClusterName: color.New(color.FgCyan, color.Bold).Sprintf,
		Success:     color.New(color.FgGreen).Sprintf,
		Error:       color.New(color.FgRed, color.Bold).Sprintf,
		Warning:     color.New(color.FgYellow).Sprintf,
		Header:      color.New(color.FgWhite, color.Bold).Sprintf,
		Duration:    color.New(color.FgBlue).Sprintf,
		Disabled:    false,
	}
}

// isTTY checks if the writer is a TTY
func isTTY(w io.Writer) bool {
	// Check if writer is a file
	if f, ok := w.(*os.File); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return false
}

// StatusColor returns an appropriate color function based on error status
func (cs *ColorScheme) StatusColor(hasError bool) func(format string, a ...interface{}) string {
	if hasError {
		return cs.Error
	}
	return cs.Success
}
