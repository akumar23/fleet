package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aryankumar/fleet/internal/executor"
)

func TestNewTableFormatter(t *testing.T) {
	tests := []struct {
		name string
		opts *Options
	}{
		{
			name: "nil options",
			opts: nil,
		},
		{
			name: "with options",
			opts: &Options{NoColor: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewTableFormatter(tt.opts)
			if formatter == nil {
				t.Fatal("NewTableFormatter returned nil")
			}
			if formatter.options == nil {
				t.Error("formatter.options is nil")
			}
		})
	}
}

func TestTableFormatter_Format(t *testing.T) {
	tests := []struct {
		name      string
		data      interface{}
		opts      *Options
		wantError bool
		contains  []string
	}{
		{
			name: "map data",
			data: map[string]interface{}{
				"name":  "test",
				"value": 123,
			},
			opts:      &Options{NoColor: true},
			wantError: false,
			contains:  []string{"name", "value", "test", "123"},
		},
		{
			name: "slice of maps",
			data: []map[string]interface{}{
				{"name": "item1", "count": 10},
				{"name": "item2", "count": 20},
			},
			opts:      &Options{NoColor: true},
			wantError: false,
			contains:  []string{"NAME", "COUNT", "item1", "item2", "10", "20"},
		},
		{
			name:      "empty slice",
			data:      []map[string]interface{}{},
			opts:      &Options{NoColor: true},
			wantError: false,
			contains:  []string{},
		},
		{
			name:      "string data",
			data:      "simple string",
			opts:      &Options{NoColor: true},
			wantError: false,
			contains:  []string{"simple string"},
		},
		{
			name:      "nil data",
			data:      nil,
			opts:      &Options{NoColor: true},
			wantError: false,
			contains:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewTableFormatter(tt.opts)
			var buf bytes.Buffer

			err := formatter.Format(&buf, tt.data)

			if (err != nil) != tt.wantError {
				t.Errorf("Format() error = %v, wantError %v", err, tt.wantError)
				return
			}

			output := buf.String()
			for _, substr := range tt.contains {
				if !strings.Contains(output, substr) {
					t.Errorf("Format() output missing %q\nGot: %s", substr, output)
				}
			}
		})
	}
}

func TestTableFormatter_FormatMultiCluster(t *testing.T) {
	tests := []struct {
		name      string
		results   []executor.Result
		opts      *Options
		wantError bool
		contains  []string
		notContains []string
	}{
		{
			name: "successful results",
			results: []executor.Result{
				{
					ClusterName: "cluster1",
					Data:        map[string]string{"key": "value1"},
					Error:       nil,
					Duration:    100 * time.Millisecond,
				},
				{
					ClusterName: "cluster2",
					Data:        map[string]string{"key": "value2"},
					Error:       nil,
					Duration:    200 * time.Millisecond,
				},
			},
			opts:      &Options{NoColor: true},
			wantError: false,
			contains:  []string{"CLUSTER", "STATUS", "DURATION", "cluster1", "cluster2", "Success", "Summary"},
		},
		{
			name: "mixed results",
			results: []executor.Result{
				{
					ClusterName: "cluster1",
					Data:        "success data",
					Error:       nil,
					Duration:    100 * time.Millisecond,
				},
				{
					ClusterName: "cluster2",
					Data:        nil,
					Error:       errors.New("connection failed"),
					Duration:    50 * time.Millisecond,
				},
			},
			opts:      &Options{NoColor: true},
			wantError: false,
			contains:  []string{"cluster1", "cluster2", "Success", "Failed", "Summary", "1 successful", "1 failed"},
		},
		{
			name:      "empty results",
			results:   []executor.Result{},
			opts:      &Options{NoColor: true},
			wantError: false,
			contains:  []string{"No results"},
		},
		{
			name: "wide mode",
			results: []executor.Result{
				{
					ClusterName: "cluster1",
					Data:        "test data",
					Error:       nil,
					Duration:    100 * time.Millisecond,
				},
			},
			opts:      &Options{NoColor: true, Wide: true},
			wantError: false,
			contains:  []string{"CLUSTER", "STATUS", "DURATION", "DATA", "cluster1", "test data"},
		},
		{
			name: "wide mode with error",
			results: []executor.Result{
				{
					ClusterName: "cluster1",
					Data:        nil,
					Error:       errors.New("connection timeout"),
					Duration:    100 * time.Millisecond,
				},
			},
			opts:      &Options{NoColor: true, Wide: true},
			wantError: false,
			contains:  []string{"DATA", "connection timeout"},
		},
		{
			name: "no headers mode",
			results: []executor.Result{
				{
					ClusterName: "cluster1",
					Data:        "data",
					Error:       nil,
					Duration:    100 * time.Millisecond,
				},
			},
			opts:      &Options{NoColor: true, NoHeaders: true},
			wantError: false,
			contains:  []string{"cluster1", "Success"},
			notContains: []string{"CLUSTER", "STATUS", "DURATION"},
		},
		{
			name: "wide mode with long data",
			results: []executor.Result{
				{
					ClusterName: "cluster1",
					Data:        "this is a very long data string that should be truncated when displayed in the table",
					Error:       nil,
					Duration:    100 * time.Millisecond,
				},
			},
			opts:      &Options{NoColor: true, Wide: true},
			wantError: false,
			contains:  []string{"cluster1", "..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewTableFormatter(tt.opts)
			var buf bytes.Buffer

			err := formatter.FormatMultiCluster(&buf, tt.results)

			if (err != nil) != tt.wantError {
				t.Errorf("FormatMultiCluster() error = %v, wantError %v", err, tt.wantError)
				return
			}

			output := buf.String()
			for _, substr := range tt.contains {
				if !strings.Contains(output, substr) {
					t.Errorf("FormatMultiCluster() output missing %q\nGot: %s", substr, output)
				}
			}

			for _, substr := range tt.notContains {
				if strings.Contains(output, substr) {
					t.Errorf("FormatMultiCluster() output should not contain %q\nGot: %s", substr, output)
				}
			}
		})
	}
}

func TestTableFormatter_CreateTable(t *testing.T) {
	formatter := NewTableFormatter(&Options{})
	var buf bytes.Buffer

	table := formatter.createTable(&buf)

	if table == nil {
		t.Fatal("createTable returned nil")
	}

	// Test that table has kubectl-style configuration
	// We can't directly inspect table configuration, so we'll test by rendering
	table.SetHeader([]string{"COL1", "COL2"})
	table.Append([]string{"val1", "val2"})
	table.Render()

	output := buf.String()

	// Should not contain borders
	if strings.Contains(output, "+") || strings.Contains(output, "|") {
		t.Error("Table contains borders (should be borderless)")
	}
}

func TestTableFormatter_FormatResultRow(t *testing.T) {
	formatter := NewTableFormatter(&Options{NoColor: true})
	colors := NewColorScheme(&bytes.Buffer{}, true)

	tests := []struct {
		name           string
		result         executor.Result
		wide           bool
		checkPositions map[int]string // position -> expected substring
	}{
		{
			name: "success result",
			result: executor.Result{
				ClusterName: "cluster1",
				Data:        "test data",
				Error:       nil,
				Duration:    100 * time.Millisecond,
			},
			wide: false,
			checkPositions: map[int]string{
				0: "cluster1",
				1: "Success",
			},
		},
		{
			name: "error result",
			result: executor.Result{
				ClusterName: "cluster2",
				Data:        nil,
				Error:       errors.New("failed"),
				Duration:    50 * time.Millisecond,
			},
			wide: false,
			checkPositions: map[int]string{
				0: "cluster2",
				1: "Failed",
			},
		},
		{
			name: "wide mode with data",
			result: executor.Result{
				ClusterName: "cluster3",
				Data:        "some data",
				Error:       nil,
				Duration:    200 * time.Millisecond,
			},
			wide: true,
			checkPositions: map[int]string{
				0: "cluster3",
				1: "Success",
				3: "some data", // DATA is at index 3
			},
		},
		{
			name: "wide mode with error",
			result: executor.Result{
				ClusterName: "cluster4",
				Data:        nil,
				Error:       errors.New("connection error"),
				Duration:    100 * time.Millisecond,
			},
			wide: true,
			checkPositions: map[int]string{
				0: "cluster4",
				1: "Failed",
				3: "connection error", // ERROR is at index 3
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter.options.Wide = tt.wide
			row := formatter.formatResultRow(tt.result, colors)

			for pos, expected := range tt.checkPositions {
				if pos >= len(row) {
					t.Errorf("Row too short: expected at least %d elements, got %d", pos+1, len(row))
					continue
				}
				if !strings.Contains(row[pos], expected) {
					t.Errorf("Row[%d] = %q, want to contain %q", pos, row[pos], expected)
				}
			}
		})
	}
}
