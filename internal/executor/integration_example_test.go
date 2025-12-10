package executor_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aryankumar/fleet/internal/executor"
)

// Example_integrationWithClusterManager demonstrates how to use the executor
// with Fleet's cluster manager for multi-cluster operations.
//
// This example shows a realistic workflow:
// 1. Create a worker pool
// 2. Submit tasks for each cluster
// 3. Execute with progress reporting
// 4. Process and display results
func Example_integrationWithClusterManager() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	// Create executor pool with optimal worker count for API calls
	// For I/O-bound operations like Kubernetes API calls, we can use more workers
	pool := executor.NewPool(5, logger)

	// Simulate getting clusters from cluster manager
	// In real usage: clusters := clusterManager.GetAllClients()
	clusterNames := []string{"prod-us-east", "prod-us-west", "staging-us", "dev"}

	// Submit tasks for each cluster
	// Each task fetches pod information from a cluster
	for _, cluster := range clusterNames {
		clusterName := cluster // Capture for closure

		pool.Submit(executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				// In real implementation, this would be:
				// k8sClient := client.(*cluster.Client)
				// pods, err := k8sClient.GetPods(ctx, namespace)

				// Simulate API call with varying durations
				time.Sleep(20 * time.Millisecond)

				// Simulate different results per cluster
				switch clusterName {
				case "prod-us-east":
					return map[string]interface{}{
						"pods":  150,
						"nodes": 10,
					}, nil
				case "prod-us-west":
					return map[string]interface{}{
						"pods":  180,
						"nodes": 12,
					}, nil
				case "staging-us":
					return map[string]interface{}{
						"pods":  42,
						"nodes": 3,
					}, nil
				case "dev":
					// Simulate a cluster with connection issues
					return nil, fmt.Errorf("connection timeout")
				default:
					return nil, nil
				}
			},
		})
	}

	// Execute all tasks with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Track progress
	fmt.Println("Fetching cluster information...")
	results := pool.ExecuteWithProgress(ctx, func(completed, total int) {
		// In a real CLI, this would update a progress bar
		// For now, we'll just track completion
	})

	// Process results
	fmt.Println("\nCluster Status:")
	fmt.Println("---------------")

	successful := executor.FilterSuccessful(results)
	for _, r := range successful {
		data := r.Data.(map[string]interface{})
		fmt.Printf("%s: %d pods, %d nodes (%.0fms)\n",
			r.ClusterName,
			data["pods"].(int),
			data["nodes"].(int),
			r.Duration.Seconds()*1000)
	}

	// Show failures
	failed := executor.FilterFailed(results)
	if len(failed) > 0 {
		fmt.Println("\nFailed Clusters:")
		for _, r := range failed {
			fmt.Printf("%s: %v\n", r.ClusterName, r.Error)
		}
	}

	// Summary statistics
	summary := executor.Summarize(results)
	fmt.Printf("\nSummary: %d total, %d successful, %d failed\n",
		summary.Total, summary.Successful, summary.Failed)
}

// Example_gracefulShutdownWithCleanup demonstrates proper cleanup and shutdown
func Example_gracefulShutdownWithCleanup() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	pool := executor.NewPool(3, logger)

	// Submit some long-running tasks
	for i := 1; i <= 5; i++ {
		clusterName := fmt.Sprintf("cluster-%d", i)
		pool.Submit(executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				// Simulate work that respects context
				select {
				case <-time.After(100 * time.Millisecond):
					return "completed", nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		})
	}

	// Start execution in background
	ctx := context.Background()
	done := make(chan []executor.Result)
	go func() {
		results := pool.Execute(ctx)
		done <- results
	}()

	// Simulate user interrupt after some time
	time.Sleep(50 * time.Millisecond)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	fmt.Println("Initiating graceful shutdown...")
	if err := pool.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Shutdown error: %v\n", err)
	} else {
		fmt.Println("Shutdown completed successfully")
	}

	// Wait for results
	results := <-done
	fmt.Printf("Received %d results\n", len(results))
}

// Example_errorHandlingPatterns demonstrates various error handling patterns
func Example_errorHandlingPatterns() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	pool := executor.NewPool(3, logger)

	// Submit tasks with various error scenarios
	tasks := map[string]error{
		"cluster-1": nil,                                 // Success
		"cluster-2": fmt.Errorf("connection refused"),    // Connection error
		"cluster-3": nil,                                 // Success
		"cluster-4": fmt.Errorf("authentication failed"), // Auth error
		"cluster-5": fmt.Errorf("timeout"),               // Timeout
		"cluster-6": nil,                                 // Success
	}

	for cluster, expectedError := range tasks {
		clusterName := cluster
		taskError := expectedError

		pool.Submit(executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				if taskError != nil {
					return nil, taskError
				}
				return "success", nil
			},
		})
	}

	results := pool.Execute(context.Background())

	// Pattern 1: Check overall success
	if executor.AllSuccessful(results) {
		fmt.Println("All operations succeeded!")
	} else {
		successRate := executor.SuccessRate(results)
		fmt.Printf("Success rate: %.1f%%\n", successRate)
	}

	// Pattern 2: Process successful results
	successful := executor.FilterSuccessful(results)
	fmt.Printf("Successfully processed %d clusters\n", len(successful))

	// Pattern 3: Handle failures
	failed := executor.FilterFailed(results)
	if len(failed) > 0 {
		fmt.Printf("Failed clusters (%d):\n", len(failed))
		for _, r := range failed {
			fmt.Printf("  - %s: %v\n", r.ClusterName, r.Error)
		}
	}

	// Pattern 4: Group errors by type
	errors := executor.GetErrors(results)
	errorTypes := make(map[string]int)
	for _, err := range errors {
		errorTypes[err.Error()]++
	}
	fmt.Println("\nError breakdown:")
	for errType, count := range errorTypes {
		fmt.Printf("  %s: %d\n", errType, count)
	}
}

// Example_performanceMonitoring demonstrates monitoring task performance
func Example_performanceMonitoring() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	pool := executor.NewPool(4, logger)

	// Submit tasks with varying durations
	clusters := []struct {
		name     string
		duration time.Duration
	}{
		{"fast-cluster", 10 * time.Millisecond},
		{"medium-cluster-1", 50 * time.Millisecond},
		{"medium-cluster-2", 50 * time.Millisecond},
		{"slow-cluster", 100 * time.Millisecond},
	}

	for _, c := range clusters {
		cluster := c
		pool.Submit(executor.Task{
			ClusterName: cluster.name,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				time.Sleep(cluster.duration)
				return "done", nil
			},
		})
	}

	results := pool.Execute(context.Background())

	// Performance analysis
	summary := executor.Summarize(results)

	fmt.Printf("Performance Metrics:\n")
	fmt.Printf("  Total clusters: %d\n", summary.Total)
	fmt.Printf("  Average duration: %v\n", summary.AvgDuration.Round(time.Millisecond))
	fmt.Printf("  Fastest: %v\n", summary.MinDuration.Round(time.Millisecond))
	fmt.Printf("  Slowest: %v\n", summary.MaxDuration.Round(time.Millisecond))

	// Identify slow clusters (> 75ms)
	fmt.Println("\nSlow clusters (>75ms):")
	for _, r := range results {
		if r.Duration > 75*time.Millisecond {
			fmt.Printf("  - %s: %v\n", r.ClusterName, r.Duration.Round(time.Millisecond))
		}
	}
}
