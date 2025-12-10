package output

import (
	"bytes"
	"os"
	"testing"
)

func TestNewColorScheme(t *testing.T) {
	tests := []struct {
		name             string
		noColor          bool
		writer           *bytes.Buffer
		expectedDisabled bool
	}{
		{
			name:             "colors disabled with noColor flag",
			noColor:          true,
			writer:           &bytes.Buffer{},
			expectedDisabled: true,
		},
		{
			name:             "colors disabled for non-TTY",
			noColor:          false,
			writer:           &bytes.Buffer{},
			expectedDisabled: true, // bytes.Buffer is not a TTY
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewColorScheme(tt.writer, tt.noColor)

			if cs == nil {
				t.Fatal("NewColorScheme returned nil")
			}

			if cs.Disabled != tt.expectedDisabled {
				t.Errorf("Disabled = %v, want %v", cs.Disabled, tt.expectedDisabled)
			}

			// Test that color functions work (even if disabled)
			if cs.ClusterName == nil {
				t.Error("ClusterName function is nil")
			}
			if cs.Success == nil {
				t.Error("Success function is nil")
			}
			if cs.Error == nil {
				t.Error("Error function is nil")
			}
			if cs.Warning == nil {
				t.Error("Warning function is nil")
			}
			if cs.Header == nil {
				t.Error("Header function is nil")
			}
			if cs.Duration == nil {
				t.Error("Duration function is nil")
			}
		})
	}
}

func TestColorScheme_Functions(t *testing.T) {
	// Test with colors disabled
	cs := NewColorScheme(&bytes.Buffer{}, true)

	tests := []struct {
		name     string
		fn       func(format string, a ...interface{}) string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "ClusterName",
			fn:       cs.ClusterName,
			format:   "cluster-%d",
			args:     []interface{}{1},
			expected: "cluster-1",
		},
		{
			name:     "Success",
			fn:       cs.Success,
			format:   "success: %s",
			args:     []interface{}{"ok"},
			expected: "success: ok",
		},
		{
			name:     "Error",
			fn:       cs.Error,
			format:   "error: %s",
			args:     []interface{}{"failed"},
			expected: "error: failed",
		},
		{
			name:     "Warning",
			fn:       cs.Warning,
			format:   "warning: %s",
			args:     []interface{}{"caution"},
			expected: "warning: caution",
		},
		{
			name:     "Header",
			fn:       cs.Header,
			format:   "HEADER",
			args:     nil,
			expected: "HEADER",
		},
		{
			name:     "Duration",
			fn:       cs.Duration,
			format:   "%dms",
			args:     []interface{}{100},
			expected: "100ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.format, tt.args...)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestColorScheme_StatusColor(t *testing.T) {
	cs := NewColorScheme(&bytes.Buffer{}, true)

	tests := []struct {
		name     string
		hasError bool
		format   string
		expected string
	}{
		{
			name:     "success status",
			hasError: false,
			format:   "OK",
			expected: "OK",
		},
		{
			name:     "error status",
			hasError: true,
			format:   "FAILED",
			expected: "FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := cs.StatusColor(tt.hasError)
			result := fn(tt.format)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsTTY(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *os.File
		expected bool
		cleanup  func(*os.File)
	}{
		{
			name: "stdout",
			setup: func() *os.File {
				return os.Stdout
			},
			expected: false, // May be true in actual terminal, but false in test
			cleanup:  func(f *os.File) {},
		},
		{
			name: "stderr",
			setup: func() *os.File {
				return os.Stderr
			},
			expected: false, // May be true in actual terminal, but false in test
			cleanup:  func(f *os.File) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.setup()
			defer tt.cleanup(f)

			result := isTTY(f)
			// We can't reliably test TTY detection in unit tests
			// Just verify the function doesn't panic
			_ = result
		})
	}

	// Test with non-file writer
	t.Run("non-file writer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		result := isTTY(buf)
		if result != false {
			t.Errorf("isTTY(bytes.Buffer) = %v, want false", result)
		}
	})
}
