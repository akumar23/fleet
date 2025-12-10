package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aryankumar/fleet/internal/executor"
	"gopkg.in/yaml.v3"
)

func TestNewYAMLFormatter(t *testing.T) {
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
			formatter := NewYAMLFormatter(tt.opts)
			if formatter == nil {
				t.Fatal("NewYAMLFormatter returned nil")
			}
			if formatter.options == nil {
				t.Error("formatter.options is nil")
			}
		})
	}
}

func TestYAMLFormatter_Format(t *testing.T) {
	tests := []struct {
		name      string
		data      interface{}
		wantError bool
		validate  func(t *testing.T, output string)
	}{
		{
			name: "simple map",
			data: map[string]interface{}{
				"name":  "test",
				"value": 123,
			},
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
					return
				}
				if result["name"] != "test" {
					t.Errorf("name = %v, want test", result["name"])
				}
				if result["value"] != 123 {
					t.Errorf("value = %v, want 123", result["value"])
				}
			},
		},
		{
			name: "slice of maps",
			data: []map[string]interface{}{
				{"id": 1, "name": "first"},
				{"id": 2, "name": "second"},
			},
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result []map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
					return
				}
				if len(result) != 2 {
					t.Errorf("len(result) = %d, want 2", len(result))
				}
			},
		},
		{
			name:      "string",
			data:      "simple string",
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result string
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
					return
				}
				if result != "simple string" {
					t.Errorf("result = %q, want %q", result, "simple string")
				}
			},
		},
		{
			name:      "nil",
			data:      nil,
			wantError: false,
			validate: func(t *testing.T, output string) {
				trimmed := strings.TrimSpace(output)
				if trimmed != "null" {
					t.Errorf("output = %q, want %q", trimmed, "null")
				}
			},
		},
		{
			name: "nested structure",
			data: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewYAMLFormatter(&Options{})
			var buf bytes.Buffer

			err := formatter.Format(&buf, tt.data)

			if (err != nil) != tt.wantError {
				t.Errorf("Format() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, buf.String())
			}
		})
	}
}

func TestYAMLFormatter_FormatMultiCluster(t *testing.T) {
	tests := []struct {
		name      string
		results   []executor.Result
		wantError bool
		validate  func(t *testing.T, output string)
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
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result []map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
					return
				}

				if len(result) != 2 {
					t.Errorf("len(result) = %d, want 2", len(result))
					return
				}

				// Check first result
				if result[0]["cluster"] != "cluster1" {
					t.Errorf("result[0][cluster] = %v, want cluster1", result[0]["cluster"])
				}
				if result[0]["status"] != "success" {
					t.Errorf("result[0][status] = %v, want success", result[0]["status"])
				}

				// Check second result
				if result[1]["cluster"] != "cluster2" {
					t.Errorf("result[1][cluster] = %v, want cluster2", result[1]["cluster"])
				}
			},
		},
		{
			name: "failed results",
			results: []executor.Result{
				{
					ClusterName: "cluster1",
					Data:        nil,
					Error:       errors.New("connection failed"),
					Duration:    50 * time.Millisecond,
				},
			},
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result []map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
					return
				}

				if len(result) != 1 {
					t.Errorf("len(result) = %d, want 1", len(result))
					return
				}

				if result[0]["status"] != "failed" {
					t.Errorf("result[0][status] = %v, want failed", result[0]["status"])
				}
				if result[0]["error"] != "connection failed" {
					t.Errorf("result[0][error] = %v, want 'connection failed'", result[0]["error"])
				}
			},
		},
		{
			name:      "empty results",
			results:   []executor.Result{},
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result []map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
					return
				}

				if len(result) != 0 {
					t.Errorf("len(result) = %d, want 0", len(result))
				}
			},
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
					Error:       errors.New("timeout"),
					Duration:    50 * time.Millisecond,
				},
			},
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result []map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
					return
				}

				if len(result) != 2 {
					t.Errorf("len(result) = %d, want 2", len(result))
					return
				}

				if result[0]["status"] != "success" {
					t.Errorf("result[0][status] = %v, want success", result[0]["status"])
				}
				if result[1]["status"] != "failed" {
					t.Errorf("result[1][status] = %v, want failed", result[1]["status"])
				}
			},
		},
		{
			name: "duration formatting",
			results: []executor.Result{
				{
					ClusterName: "cluster1",
					Data:        nil,
					Error:       nil,
					Duration:    1500 * time.Millisecond,
				},
			},
			wantError: false,
			validate: func(t *testing.T, output string) {
				var result []map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Failed to parse YAML: %v", err)
					return
				}

				if len(result) != 1 {
					t.Errorf("len(result) = %d, want 1", len(result))
					return
				}

				durationStr, ok := result[0]["duration"].(string)
				if !ok {
					t.Errorf("duration is not a string: %T", result[0]["duration"])
					return
				}

				if durationStr == "" {
					t.Error("duration string is empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewYAMLFormatter(&Options{})
			var buf bytes.Buffer

			err := formatter.FormatMultiCluster(&buf, tt.results)

			if (err != nil) != tt.wantError {
				t.Errorf("FormatMultiCluster() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, buf.String())
			}
		})
	}
}

func TestYAMLFormatter_Indentation(t *testing.T) {
	formatter := NewYAMLFormatter(&Options{})
	data := map[string]interface{}{
		"parent": map[string]interface{}{
			"child": "value",
		},
	}

	var buf bytes.Buffer
	err := formatter.Format(&buf, data)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()

	// Check that output is indented (contains newlines and spaces)
	if !strings.Contains(output, "\n") {
		t.Error("YAML output is not indented (no newlines)")
	}

	// YAML should have proper structure
	if !strings.Contains(output, "parent:") {
		t.Error("YAML output missing expected key")
	}
}

func TestYAMLFormatter_CompareWithJSON(t *testing.T) {
	// Both formatters should handle the same data structure
	data := []executor.Result{
		{
			ClusterName: "cluster1",
			Data:        map[string]string{"key": "value"},
			Error:       nil,
			Duration:    100 * time.Millisecond,
		},
	}

	yamlFormatter := NewYAMLFormatter(&Options{})
	jsonFormatter := NewJSONFormatter(&Options{})

	var yamlBuf bytes.Buffer
	var jsonBuf bytes.Buffer

	if err := yamlFormatter.FormatMultiCluster(&yamlBuf, data); err != nil {
		t.Fatalf("YAML FormatMultiCluster() error = %v", err)
	}

	if err := jsonFormatter.FormatMultiCluster(&jsonBuf, data); err != nil {
		t.Fatalf("JSON FormatMultiCluster() error = %v", err)
	}

	// Both should produce parseable output
	var yamlResult []map[string]interface{}
	if err := yaml.Unmarshal(yamlBuf.Bytes(), &yamlResult); err != nil {
		t.Errorf("Failed to parse YAML output: %v", err)
	}

	// We can parse JSON from the jsonBuf to verify structure is consistent
	// This is just a sanity check that both formatters work correctly
	if yamlBuf.Len() == 0 {
		t.Error("YAML output is empty")
	}
	if jsonBuf.Len() == 0 {
		t.Error("JSON output is empty")
	}
}
