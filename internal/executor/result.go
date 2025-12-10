package executor

import (
	"fmt"
	"strings"
	"time"
)

// CountSuccessful returns the number of successful results (no error)
func CountSuccessful(results []Result) int {
	count := 0
	for _, r := range results {
		if r.Error == nil {
			count++
		}
	}
	return count
}

// CountFailed returns the number of failed results (has error)
func CountFailed(results []Result) int {
	count := 0
	for _, r := range results {
		if r.Error != nil {
			count++
		}
	}
	return count
}

// FilterSuccessful returns only the successful results
func FilterSuccessful(results []Result) []Result {
	filtered := make([]Result, 0, len(results))
	for _, r := range results {
		if r.Error == nil {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterFailed returns only the failed results
func FilterFailed(results []Result) []Result {
	filtered := make([]Result, 0, len(results))
	for _, r := range results {
		if r.Error != nil {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FilterByCluster returns results for a specific cluster
func FilterByCluster(results []Result, clusterName string) []Result {
	filtered := make([]Result, 0)
	for _, r := range results {
		if r.ClusterName == clusterName {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// GroupByCluster groups results by cluster name
// Returns a map where the key is the cluster name and value is a slice of results
func GroupByCluster(results []Result) map[string][]Result {
	grouped := make(map[string][]Result)
	for _, r := range results {
		grouped[r.ClusterName] = append(grouped[r.ClusterName], r)
	}
	return grouped
}

// AverageDuration calculates the average duration of all results
func AverageDuration(results []Result) time.Duration {
	if len(results) == 0 {
		return 0
	}

	var total time.Duration
	for _, r := range results {
		total += r.Duration
	}

	return total / time.Duration(len(results))
}

// MaxDuration returns the maximum duration among all results
func MaxDuration(results []Result) time.Duration {
	if len(results) == 0 {
		return 0
	}

	max := results[0].Duration
	for _, r := range results {
		if r.Duration > max {
			max = r.Duration
		}
	}
	return max
}

// MinDuration returns the minimum duration among all results
func MinDuration(results []Result) time.Duration {
	if len(results) == 0 {
		return 0
	}

	min := results[0].Duration
	for _, r := range results {
		if r.Duration < min {
			min = r.Duration
		}
	}
	return min
}

// GetErrors extracts all errors from results
// Returns a slice of errors for failed results
func GetErrors(results []Result) []error {
	errors := make([]error, 0)
	for _, r := range results {
		if r.Error != nil {
			errors = append(errors, r.Error)
		}
	}
	return errors
}

// GetClusterNames extracts unique cluster names from results
func GetClusterNames(results []Result) []string {
	seen := make(map[string]bool)
	names := make([]string, 0)

	for _, r := range results {
		if !seen[r.ClusterName] {
			seen[r.ClusterName] = true
			names = append(names, r.ClusterName)
		}
	}

	return names
}

// Summary provides a summary of execution results
type Summary struct {
	Total       int
	Successful  int
	Failed      int
	AvgDuration time.Duration
	MaxDuration time.Duration
	MinDuration time.Duration
}

// Summarize creates a summary of the results
func Summarize(results []Result) Summary {
	return Summary{
		Total:       len(results),
		Successful:  CountSuccessful(results),
		Failed:      CountFailed(results),
		AvgDuration: AverageDuration(results),
		MaxDuration: MaxDuration(results),
		MinDuration: MinDuration(results),
	}
}

// String returns a human-readable string representation of the summary
func (s Summary) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Total: %d, ", s.Total))
	sb.WriteString(fmt.Sprintf("Successful: %d, ", s.Successful))
	sb.WriteString(fmt.Sprintf("Failed: %d", s.Failed))

	if s.Total > 0 {
		sb.WriteString(fmt.Sprintf(", Avg: %s", s.AvgDuration.Round(time.Millisecond)))
		sb.WriteString(fmt.Sprintf(", Max: %s", s.MaxDuration.Round(time.Millisecond)))
		sb.WriteString(fmt.Sprintf(", Min: %s", s.MinDuration.Round(time.Millisecond)))
	}

	return sb.String()
}

// HasErrors returns true if any results contain errors
func HasErrors(results []Result) bool {
	for _, r := range results {
		if r.Error != nil {
			return true
		}
	}
	return false
}

// AllSuccessful returns true if all results are successful
func AllSuccessful(results []Result) bool {
	return !HasErrors(results)
}

// SuccessRate returns the success rate as a percentage (0.0 to 100.0)
func SuccessRate(results []Result) float64 {
	if len(results) == 0 {
		return 0.0
	}
	return float64(CountSuccessful(results)) / float64(len(results)) * 100.0
}

// FailureRate returns the failure rate as a percentage (0.0 to 100.0)
func FailureRate(results []Result) float64 {
	if len(results) == 0 {
		return 0.0
	}
	return float64(CountFailed(results)) / float64(len(results)) * 100.0
}
