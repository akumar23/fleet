package executor_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aryankumar/fleet/internal/executor"
)

// Example demonstrates basic usage of the worker pool
func Example() {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a pool with 3 workers
	pool := executor.NewPool(3, logger)

	// Submit tasks
	clusters := []string{"prod-us-east", "prod-us-west", "staging"}
	for _, cluster := range clusters {
		clusterName := cluster
		pool.Submit(executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				// Simulate some work
				time.Sleep(50 * time.Millisecond)
				return fmt.Sprintf("result from %s", clusterName), nil
			},
		})
	}

	// Execute all tasks
	ctx := context.Background()
	results := pool.Execute(ctx)

	// Process results
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Cluster %s failed: %v\n", result.ClusterName, result.Error)
		} else {
			fmt.Printf("Cluster %s succeeded: %v (took %v)\n",
				result.ClusterName, result.Data, result.Duration)
		}
	}
}

// Example_withProgress demonstrates progress reporting
func ExamplePool_ExecuteWithProgress() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Reduce log noise
	}))

	pool := executor.NewPool(2, logger)

	// Submit 5 tasks
	for i := 1; i <= 5; i++ {
		clusterName := fmt.Sprintf("cluster-%d", i)
		pool.Submit(executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				time.Sleep(20 * time.Millisecond)
				return "done", nil
			},
		})
	}

	// Execute with progress reporting
	ctx := context.Background()
	results := pool.ExecuteWithProgress(ctx, func(completed, total int) {
		// Progress callback is called for each completed task
		// In a real application, you would update a progress bar or UI here
	})

	fmt.Printf("Completed %d tasks\n", len(results))
	// Output:
	// Completed 5 tasks
}

// Example_resultAggregation demonstrates result filtering and aggregation
func Example_resultAggregation() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	pool := executor.NewPool(3, logger)

	// Submit tasks with some failures
	tasks := map[string]bool{
		"cluster-1": true,  // success
		"cluster-2": false, // failure
		"cluster-3": true,  // success
		"cluster-4": false, // failure
		"cluster-5": true,  // success
	}

	for cluster, shouldSucceed := range tasks {
		succeed := shouldSucceed
		pool.Submit(executor.Task{
			ClusterName: cluster,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				if succeed {
					return "success", nil
				}
				return nil, fmt.Errorf("simulated error")
			},
		})
	}

	results := pool.Execute(context.Background())

	// Use result aggregation functions
	summary := executor.Summarize(results)
	fmt.Println(summary.String())

	fmt.Printf("\nSuccess rate: %.0f%%\n", executor.SuccessRate(results))
	fmt.Printf("Failure rate: %.0f%%\n", executor.FailureRate(results))

	// Filter results
	successful := executor.FilterSuccessful(results)
	failed := executor.FilterFailed(results)

	fmt.Printf("\nSuccessful clusters:\n")
	for _, r := range successful {
		fmt.Printf("  - %s\n", r.ClusterName)
	}

	fmt.Printf("\nFailed clusters:\n")
	for _, r := range failed {
		fmt.Printf("  - %s: %v\n", r.ClusterName, r.Error)
	}
}

// Example_contextCancellation demonstrates context-based cancellation
func Example_contextCancellation() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	pool := executor.NewPool(2, logger)

	// Submit long-running tasks
	for i := 1; i <= 5; i++ {
		clusterName := fmt.Sprintf("cluster-%d", i)
		pool.Submit(executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				select {
				case <-time.After(200 * time.Millisecond):
					return "completed", nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results := pool.Execute(ctx)

	// Count how many were cancelled
	cancelled := 0
	for _, r := range results {
		if r.Error != nil {
			cancelled++
		}
	}

	fmt.Printf("Total tasks: %d\n", len(results))
	fmt.Printf("Cancelled: %d\n", cancelled)
}

// Example_gracefulShutdown demonstrates graceful pool shutdown
func Example_gracefulShutdown() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	pool := executor.NewPool(2, logger)

	// Submit tasks
	for i := 1; i <= 3; i++ {
		clusterName := fmt.Sprintf("cluster-%d", i)
		pool.Submit(executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return "done", nil
			},
		})
	}

	// Start execution in background
	go pool.Execute(context.Background())

	// Wait a bit, then shutdown
	time.Sleep(20 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := pool.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
	} else {
		fmt.Println("Pool shut down gracefully")
	}
}

// Example_realWorldUsage demonstrates realistic usage with cluster operations
func Example_realWorldUsage() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Simulate getting pod counts from multiple clusters
	pool := executor.NewPool(3, logger)

	clusters := []string{"prod-1", "prod-2", "staging", "dev"}

	for _, cluster := range clusters {
		clusterName := cluster
		pool.Submit(executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				// Simulate API call to get pod count
				time.Sleep(30 * time.Millisecond)

				// Simulate different results
				switch clusterName {
				case "prod-1":
					return 150, nil
				case "prod-2":
					return 180, nil
				case "staging":
					return 42, nil
				case "dev":
					return nil, fmt.Errorf("connection timeout")
				default:
					return 0, nil
				}
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := pool.ExecuteWithProgress(ctx, func(completed, total int) {
		fmt.Printf("Checking clusters... %d/%d\n", completed, total)
	})

	// Display results
	fmt.Println("\nCluster Pod Counts:")
	successful := executor.FilterSuccessful(results)
	for _, r := range successful {
		fmt.Printf("  %s: %d pods\n", r.ClusterName, r.Data)
	}

	// Show failures
	failed := executor.FilterFailed(results)
	if len(failed) > 0 {
		fmt.Println("\nFailed Clusters:")
		for _, r := range failed {
			fmt.Printf("  %s: %v\n", r.ClusterName, r.Error)
		}
	}

	// Summary
	summary := executor.Summarize(results)
	fmt.Printf("\n%s\n", summary.String())
}
