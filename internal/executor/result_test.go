package executor

import (
	"errors"
	"testing"
	"time"
)

func TestCountSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected int
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 0,
		},
		{
			name: "all successful",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: nil},
				{ClusterName: "c3", Error: nil},
			},
			expected: 3,
		},
		{
			name: "all failed",
			results: []Result{
				{ClusterName: "c1", Error: errors.New("error1")},
				{ClusterName: "c2", Error: errors.New("error2")},
			},
			expected: 0,
		},
		{
			name: "mixed",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: errors.New("error")},
				{ClusterName: "c3", Error: nil},
				{ClusterName: "c4", Error: errors.New("error")},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountSuccessful(tt.results)
			if got != tt.expected {
				t.Errorf("CountSuccessful() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestCountFailed(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected int
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 0,
		},
		{
			name: "all successful",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: nil},
			},
			expected: 0,
		},
		{
			name: "all failed",
			results: []Result{
				{ClusterName: "c1", Error: errors.New("error1")},
				{ClusterName: "c2", Error: errors.New("error2")},
				{ClusterName: "c3", Error: errors.New("error3")},
			},
			expected: 3,
		},
		{
			name: "mixed",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: errors.New("error")},
				{ClusterName: "c3", Error: nil},
				{ClusterName: "c4", Error: errors.New("error")},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountFailed(tt.results)
			if got != tt.expected {
				t.Errorf("CountFailed() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestFilterSuccessful(t *testing.T) {
	results := []Result{
		{ClusterName: "c1", Error: nil, Data: "data1"},
		{ClusterName: "c2", Error: errors.New("error"), Data: nil},
		{ClusterName: "c3", Error: nil, Data: "data3"},
		{ClusterName: "c4", Error: errors.New("error"), Data: nil},
	}

	filtered := FilterSuccessful(results)

	if len(filtered) != 2 {
		t.Errorf("expected 2 successful results, got %d", len(filtered))
	}

	for _, r := range filtered {
		if r.Error != nil {
			t.Errorf("filtered result has error: %v", r.Error)
		}
	}

	expectedClusters := map[string]bool{"c1": true, "c3": true}
	for _, r := range filtered {
		if !expectedClusters[r.ClusterName] {
			t.Errorf("unexpected cluster in filtered results: %s", r.ClusterName)
		}
	}
}

func TestFilterFailed(t *testing.T) {
	results := []Result{
		{ClusterName: "c1", Error: nil, Data: "data1"},
		{ClusterName: "c2", Error: errors.New("error"), Data: nil},
		{ClusterName: "c3", Error: nil, Data: "data3"},
		{ClusterName: "c4", Error: errors.New("error"), Data: nil},
	}

	filtered := FilterFailed(results)

	if len(filtered) != 2 {
		t.Errorf("expected 2 failed results, got %d", len(filtered))
	}

	for _, r := range filtered {
		if r.Error == nil {
			t.Errorf("filtered result has no error")
		}
	}

	expectedClusters := map[string]bool{"c2": true, "c4": true}
	for _, r := range filtered {
		if !expectedClusters[r.ClusterName] {
			t.Errorf("unexpected cluster in filtered results: %s", r.ClusterName)
		}
	}
}

func TestFilterByCluster(t *testing.T) {
	results := []Result{
		{ClusterName: "cluster1", Data: "data1"},
		{ClusterName: "cluster2", Data: "data2"},
		{ClusterName: "cluster1", Data: "data3"},
		{ClusterName: "cluster3", Data: "data4"},
	}

	filtered := FilterByCluster(results, "cluster1")

	if len(filtered) != 2 {
		t.Errorf("expected 2 results for cluster1, got %d", len(filtered))
	}

	for _, r := range filtered {
		if r.ClusterName != "cluster1" {
			t.Errorf("expected cluster1, got %s", r.ClusterName)
		}
	}

	// Test with non-existent cluster
	filtered = FilterByCluster(results, "nonexistent")
	if len(filtered) != 0 {
		t.Errorf("expected 0 results for nonexistent cluster, got %d", len(filtered))
	}
}

func TestGroupByCluster(t *testing.T) {
	results := []Result{
		{ClusterName: "cluster1", Data: "data1"},
		{ClusterName: "cluster2", Data: "data2"},
		{ClusterName: "cluster1", Data: "data3"},
		{ClusterName: "cluster3", Data: "data4"},
		{ClusterName: "cluster2", Data: "data5"},
	}

	grouped := GroupByCluster(results)

	if len(grouped) != 3 {
		t.Errorf("expected 3 clusters, got %d", len(grouped))
	}

	if len(grouped["cluster1"]) != 2 {
		t.Errorf("expected 2 results for cluster1, got %d", len(grouped["cluster1"]))
	}

	if len(grouped["cluster2"]) != 2 {
		t.Errorf("expected 2 results for cluster2, got %d", len(grouped["cluster2"]))
	}

	if len(grouped["cluster3"]) != 1 {
		t.Errorf("expected 1 result for cluster3, got %d", len(grouped["cluster3"]))
	}
}

func TestAverageDuration(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected time.Duration
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 0,
		},
		{
			name: "single result",
			results: []Result{
				{Duration: 100 * time.Millisecond},
			},
			expected: 100 * time.Millisecond,
		},
		{
			name: "multiple results",
			results: []Result{
				{Duration: 100 * time.Millisecond},
				{Duration: 200 * time.Millisecond},
				{Duration: 300 * time.Millisecond},
			},
			expected: 200 * time.Millisecond,
		},
		{
			name: "different durations",
			results: []Result{
				{Duration: 50 * time.Millisecond},
				{Duration: 150 * time.Millisecond},
			},
			expected: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AverageDuration(tt.results)
			if got != tt.expected {
				t.Errorf("AverageDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMaxDuration(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected time.Duration
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 0,
		},
		{
			name: "single result",
			results: []Result{
				{Duration: 100 * time.Millisecond},
			},
			expected: 100 * time.Millisecond,
		},
		{
			name: "multiple results",
			results: []Result{
				{Duration: 100 * time.Millisecond},
				{Duration: 500 * time.Millisecond},
				{Duration: 200 * time.Millisecond},
			},
			expected: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxDuration(tt.results)
			if got != tt.expected {
				t.Errorf("MaxDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMinDuration(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected time.Duration
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 0,
		},
		{
			name: "single result",
			results: []Result{
				{Duration: 100 * time.Millisecond},
			},
			expected: 100 * time.Millisecond,
		},
		{
			name: "multiple results",
			results: []Result{
				{Duration: 100 * time.Millisecond},
				{Duration: 50 * time.Millisecond},
				{Duration: 200 * time.Millisecond},
			},
			expected: 50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MinDuration(tt.results)
			if got != tt.expected {
				t.Errorf("MinDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrors(t *testing.T) {
	results := []Result{
		{ClusterName: "c1", Error: nil},
		{ClusterName: "c2", Error: errors.New("error1")},
		{ClusterName: "c3", Error: nil},
		{ClusterName: "c4", Error: errors.New("error2")},
	}

	errs := GetErrors(results)

	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}

	for _, err := range errs {
		if err == nil {
			t.Error("got nil error in error list")
		}
	}
}

func TestGetClusterNames(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected int
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 0,
		},
		{
			name: "unique clusters",
			results: []Result{
				{ClusterName: "c1"},
				{ClusterName: "c2"},
				{ClusterName: "c3"},
			},
			expected: 3,
		},
		{
			name: "duplicate clusters",
			results: []Result{
				{ClusterName: "c1"},
				{ClusterName: "c2"},
				{ClusterName: "c1"},
				{ClusterName: "c3"},
				{ClusterName: "c2"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := GetClusterNames(tt.results)
			if len(names) != tt.expected {
				t.Errorf("expected %d unique cluster names, got %d", tt.expected, len(names))
			}

			// Check uniqueness
			seen := make(map[string]bool)
			for _, name := range names {
				if seen[name] {
					t.Errorf("duplicate cluster name: %s", name)
				}
				seen[name] = true
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	results := []Result{
		{ClusterName: "c1", Error: nil, Duration: 100 * time.Millisecond},
		{ClusterName: "c2", Error: errors.New("error"), Duration: 200 * time.Millisecond},
		{ClusterName: "c3", Error: nil, Duration: 300 * time.Millisecond},
		{ClusterName: "c4", Error: errors.New("error"), Duration: 50 * time.Millisecond},
		{ClusterName: "c5", Error: nil, Duration: 150 * time.Millisecond},
	}

	summary := Summarize(results)

	if summary.Total != 5 {
		t.Errorf("expected Total=5, got %d", summary.Total)
	}

	if summary.Successful != 3 {
		t.Errorf("expected Successful=3, got %d", summary.Successful)
	}

	if summary.Failed != 2 {
		t.Errorf("expected Failed=2, got %d", summary.Failed)
	}

	expectedAvg := 160 * time.Millisecond
	if summary.AvgDuration != expectedAvg {
		t.Errorf("expected AvgDuration=%v, got %v", expectedAvg, summary.AvgDuration)
	}

	expectedMax := 300 * time.Millisecond
	if summary.MaxDuration != expectedMax {
		t.Errorf("expected MaxDuration=%v, got %v", expectedMax, summary.MaxDuration)
	}

	expectedMin := 50 * time.Millisecond
	if summary.MinDuration != expectedMin {
		t.Errorf("expected MinDuration=%v, got %v", expectedMin, summary.MinDuration)
	}
}

func TestSummary_String(t *testing.T) {
	summary := Summary{
		Total:       10,
		Successful:  7,
		Failed:      3,
		AvgDuration: 123456789 * time.Nanosecond,
		MaxDuration: 200 * time.Millisecond,
		MinDuration: 50 * time.Millisecond,
	}

	str := summary.String()

	// Check that key information is present
	requiredSubstrings := []string{
		"Total: 10",
		"Successful: 7",
		"Failed: 3",
		"Avg:",
		"Max:",
		"Min:",
	}

	for _, substr := range requiredSubstrings {
		if !containsSubstring(str, substr) {
			t.Errorf("summary string missing %q: %s", substr, str)
		}
	}
}

func TestSummary_String_Empty(t *testing.T) {
	summary := Summary{
		Total:      0,
		Successful: 0,
		Failed:     0,
	}

	str := summary.String()

	if !containsSubstring(str, "Total: 0") {
		t.Errorf("expected 'Total: 0' in summary string: %s", str)
	}
}

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected bool
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: false,
		},
		{
			name: "no errors",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: nil},
			},
			expected: false,
		},
		{
			name: "has errors",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: errors.New("error")},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasErrors(tt.results)
			if got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAllSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected bool
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: true,
		},
		{
			name: "all successful",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: nil},
			},
			expected: true,
		},
		{
			name: "has failures",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: errors.New("error")},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllSuccessful(tt.results)
			if got != tt.expected {
				t.Errorf("AllSuccessful() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSuccessRate(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected float64
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 0.0,
		},
		{
			name: "all successful",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: nil},
				{ClusterName: "c3", Error: nil},
			},
			expected: 100.0,
		},
		{
			name: "all failed",
			results: []Result{
				{ClusterName: "c1", Error: errors.New("error")},
				{ClusterName: "c2", Error: errors.New("error")},
			},
			expected: 0.0,
		},
		{
			name: "50% success",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: errors.New("error")},
			},
			expected: 50.0,
		},
		{
			name: "75% success",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: nil},
				{ClusterName: "c3", Error: nil},
				{ClusterName: "c4", Error: errors.New("error")},
			},
			expected: 75.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuccessRate(tt.results)
			if got != tt.expected {
				t.Errorf("SuccessRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFailureRate(t *testing.T) {
	tests := []struct {
		name     string
		results  []Result
		expected float64
	}{
		{
			name:     "empty results",
			results:  []Result{},
			expected: 0.0,
		},
		{
			name: "all successful",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: nil},
			},
			expected: 0.0,
		},
		{
			name: "all failed",
			results: []Result{
				{ClusterName: "c1", Error: errors.New("error")},
				{ClusterName: "c2", Error: errors.New("error")},
			},
			expected: 100.0,
		},
		{
			name: "50% failure",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: errors.New("error")},
			},
			expected: 50.0,
		},
		{
			name: "25% failure",
			results: []Result{
				{ClusterName: "c1", Error: nil},
				{ClusterName: "c2", Error: nil},
				{ClusterName: "c3", Error: nil},
				{ClusterName: "c4", Error: errors.New("error")},
			},
			expected: 25.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FailureRate(tt.results)
			if got != tt.expected {
				t.Errorf("FailureRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to check if string contains substring
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
