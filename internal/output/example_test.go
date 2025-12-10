package output_test

import (
	"errors"
	"os"
	"time"

	"github.com/aryankumar/fleet/internal/executor"
	"github.com/aryankumar/fleet/internal/output"
)

// Example_tableFormatter demonstrates using the table formatter
func Example_tableFormatter() {
	// Create a table formatter
	formatter := output.NewFormatter(output.FormatTable, output.WithNoColor(true))

	// Create some results
	results := []executor.Result{
		{
			ClusterName: "production",
			Data:        map[string]string{"pods": "10"},
			Error:       nil,
			Duration:    150 * time.Millisecond,
		},
		{
			ClusterName: "staging",
			Data:        map[string]string{"pods": "5"},
			Error:       nil,
			Duration:    100 * time.Millisecond,
		},
	}

	// Format the results
	formatter.FormatMultiCluster(os.Stdout, results)
}

// Example_jsonFormatter demonstrates using the JSON formatter
func Example_jsonFormatter() {
	// Create a JSON formatter
	formatter := output.NewFormatter(output.FormatJSON)

	// Create results with mixed success/failure
	results := []executor.Result{
		{
			ClusterName: "cluster1",
			Data:        map[string]interface{}{"status": "healthy", "count": 3},
			Error:       nil,
			Duration:    200 * time.Millisecond,
		},
		{
			ClusterName: "cluster2",
			Data:        nil,
			Error:       errors.New("connection timeout"),
			Duration:    50 * time.Millisecond,
		},
	}

	// Format the results
	formatter.FormatMultiCluster(os.Stdout, results)
}

// Example_yamlFormatter demonstrates using the YAML formatter
func Example_yamlFormatter() {
	// Create a YAML formatter
	formatter := output.NewFormatter(output.FormatYAML)

	// Create a single data item
	data := map[string]interface{}{
		"cluster":   "production",
		"namespace": "default",
		"resources": map[string]int{
			"pods":     10,
			"services": 5,
		},
	}

	// Format the data
	formatter.Format(os.Stdout, data)
}

// Example_wideMode demonstrates using wide mode for additional details
func Example_wideMode() {
	// Create a table formatter with wide mode
	formatter := output.NewFormatter(
		output.FormatTable,
		output.WithNoColor(true),
		output.WithWide(true),
	)

	// Create results
	results := []executor.Result{
		{
			ClusterName: "cluster1",
			Data:        "Successfully deployed",
			Error:       nil,
			Duration:    250 * time.Millisecond,
		},
		{
			ClusterName: "cluster2",
			Data:        nil,
			Error:       errors.New("deployment failed"),
			Duration:    100 * time.Millisecond,
		},
	}

	// Format with data column visible
	formatter.FormatMultiCluster(os.Stdout, results)
}

// Example_noHeaders demonstrates table output without headers
func Example_noHeaders() {
	// Create a table formatter without headers
	formatter := output.NewFormatter(
		output.FormatTable,
		output.WithNoColor(true),
		output.WithNoHeaders(true),
	)

	// Create results
	results := []executor.Result{
		{
			ClusterName: "cluster1",
			Data:        nil,
			Error:       nil,
			Duration:    100 * time.Millisecond,
		},
	}

	// Format without headers
	formatter.FormatMultiCluster(os.Stdout, results)
}

// Example_colorOutput demonstrates color output (requires TTY)
func Example_colorOutput() {
	// Create a table formatter with colors enabled
	// Colors will be automatically disabled if not outputting to a TTY
	formatter := output.NewFormatter(output.FormatTable)

	// Create results with successes and failures
	results := []executor.Result{
		{
			ClusterName: "production",
			Data:        "healthy",
			Error:       nil,
			Duration:    120 * time.Millisecond,
		},
		{
			ClusterName: "staging",
			Data:        nil,
			Error:       errors.New("connection refused"),
			Duration:    50 * time.Millisecond,
		},
	}

	// Format with colors (if TTY)
	formatter.FormatMultiCluster(os.Stdout, results)
}
