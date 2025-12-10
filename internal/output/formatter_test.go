package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/aryankumar/fleet/internal/executor"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name           string
		format         Format
		opts           []Option
		expectedType   string
		checkNoColor   bool
		expectedNoColor bool
	}{
		{
			name:         "table formatter default",
			format:       FormatTable,
			opts:         nil,
			expectedType: "*output.TableFormatter",
		},
		{
			name:         "json formatter",
			format:       FormatJSON,
			opts:         nil,
			expectedType: "*output.JSONFormatter",
		},
		{
			name:         "yaml formatter",
			format:       FormatYAML,
			opts:         nil,
			expectedType: "*output.YAMLFormatter",
		},
		{
			name:         "empty format defaults to table",
			format:       "",
			opts:         nil,
			expectedType: "*output.TableFormatter",
		},
		{
			name:         "unknown format defaults to table",
			format:       "unknown",
			opts:         nil,
			expectedType: "*output.TableFormatter",
		},
		{
			name:         "table with no color option",
			format:       FormatTable,
			opts:         []Option{WithNoColor(true)},
			expectedType: "*output.TableFormatter",
		},
		{
			name:         "table with no headers option",
			format:       FormatTable,
			opts:         []Option{WithNoHeaders(true)},
			expectedType: "*output.TableFormatter",
		},
		{
			name:         "table with wide option",
			format:       FormatTable,
			opts:         []Option{WithWide(true)},
			expectedType: "*output.TableFormatter",
		},
		{
			name:         "table with multiple options",
			format:       FormatTable,
			opts:         []Option{WithNoColor(true), WithNoHeaders(true), WithWide(true)},
			expectedType: "*output.TableFormatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.format, tt.opts...)

			if formatter == nil {
				t.Fatal("NewFormatter returned nil")
			}

			// Check type using type assertion
			switch tt.expectedType {
			case "*output.TableFormatter":
				if _, ok := formatter.(*TableFormatter); !ok {
					t.Errorf("expected TableFormatter, got %T", formatter)
				}
			case "*output.JSONFormatter":
				if _, ok := formatter.(*JSONFormatter); !ok {
					t.Errorf("expected JSONFormatter, got %T", formatter)
				}
			case "*output.YAMLFormatter":
				if _, ok := formatter.(*YAMLFormatter); !ok {
					t.Errorf("expected YAMLFormatter, got %T", formatter)
				}
			}
		})
	}
}

func TestOptions(t *testing.T) {
	tests := []struct {
		name             string
		opts             []Option
		expectedNoColor  bool
		expectedNoHeaders bool
		expectedWide     bool
	}{
		{
			name:             "default options",
			opts:             nil,
			expectedNoColor:  false,
			expectedNoHeaders: false,
			expectedWide:     false,
		},
		{
			name:             "with no color",
			opts:             []Option{WithNoColor(true)},
			expectedNoColor:  true,
			expectedNoHeaders: false,
			expectedWide:     false,
		},
		{
			name:             "with no headers",
			opts:             []Option{WithNoHeaders(true)},
			expectedNoColor:  false,
			expectedNoHeaders: true,
			expectedWide:     false,
		},
		{
			name:             "with wide",
			opts:             []Option{WithWide(true)},
			expectedNoColor:  false,
			expectedNoHeaders: false,
			expectedWide:     true,
		},
		{
			name:             "all options",
			opts:             []Option{WithNoColor(true), WithNoHeaders(true), WithWide(true)},
			expectedNoColor:  true,
			expectedNoHeaders: true,
			expectedWide:     true,
		},
		{
			name:             "override options",
			opts:             []Option{WithNoColor(true), WithNoColor(false)},
			expectedNoColor:  false,
			expectedNoHeaders: false,
			expectedWide:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &Options{}
			for _, opt := range tt.opts {
				opt(options)
			}

			if options.NoColor != tt.expectedNoColor {
				t.Errorf("NoColor = %v, want %v", options.NoColor, tt.expectedNoColor)
			}
			if options.NoHeaders != tt.expectedNoHeaders {
				t.Errorf("NoHeaders = %v, want %v", options.NoHeaders, tt.expectedNoHeaders)
			}
			if options.Wide != tt.expectedWide {
				t.Errorf("Wide = %v, want %v", options.Wide, tt.expectedWide)
			}
		})
	}
}

func TestFormatter_FormatAndFormatMultiCluster(t *testing.T) {
	// Test data
	singleData := map[string]interface{}{
		"name": "test",
		"value": 123,
	}

	results := []executor.Result{
		{
			ClusterName: "cluster1",
			Data:        map[string]string{"key": "value1"},
			Error:       nil,
			Duration:    100 * time.Millisecond,
		},
		{
			ClusterName: "cluster2",
			Data:        nil,
			Error:       nil,
			Duration:    200 * time.Millisecond,
		},
	}

	formats := []Format{FormatTable, FormatJSON, FormatYAML}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			formatter := NewFormatter(format, WithNoColor(true))

			// Test Format
			t.Run("Format", func(t *testing.T) {
				var buf bytes.Buffer
				err := formatter.Format(&buf, singleData)
				if err != nil {
					t.Errorf("Format() error = %v", err)
				}

				if buf.Len() == 0 {
					t.Error("Format() produced no output")
				}
			})

			// Test FormatMultiCluster
			t.Run("FormatMultiCluster", func(t *testing.T) {
				var buf bytes.Buffer
				err := formatter.FormatMultiCluster(&buf, results)
				if err != nil {
					t.Errorf("FormatMultiCluster() error = %v", err)
				}

				if buf.Len() == 0 {
					t.Error("FormatMultiCluster() produced no output")
				}
			})

			// Test FormatMultiCluster with empty results
			t.Run("FormatMultiCluster empty", func(t *testing.T) {
				var buf bytes.Buffer
				err := formatter.FormatMultiCluster(&buf, []executor.Result{})
				if err != nil {
					t.Errorf("FormatMultiCluster() error = %v", err)
				}
			})
		})
	}
}
