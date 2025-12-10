// Package executor provides a concurrent execution engine for running tasks across multiple clusters.
//
// The package implements a worker pool pattern with bounded concurrency, graceful shutdown,
// context-aware cancellation, and comprehensive result aggregation capabilities.
//
// # Key Features
//
//   - Worker pool with configurable concurrency
//   - Context-aware task execution and cancellation
//   - Progress reporting callbacks
//   - Graceful shutdown with timeout support
//   - Result filtering and aggregation utilities
//   - Thread-safe operations with proper synchronization
//   - Zero goroutine leaks
//
// # Basic Usage
//
// Create a pool, submit tasks, and execute them:
//
//	pool := executor.NewPool(5, logger)
//
//	for _, cluster := range clusters {
//	    pool.Submit(executor.Task{
//	        ClusterName: cluster,
//	        Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
//	            // Perform cluster operation
//	            return result, nil
//	        },
//	    })
//	}
//
//	results := pool.Execute(context.Background())
//
// # Progress Reporting
//
// Track execution progress with a callback:
//
//	results := pool.ExecuteWithProgress(ctx, func(completed, total int) {
//	    fmt.Printf("Progress: %d/%d\n", completed, total)
//	})
//
// # Result Aggregation
//
// Filter and analyze results:
//
//	successful := executor.FilterSuccessful(results)
//	failed := executor.FilterFailed(results)
//	summary := executor.Summarize(results)
//	successRate := executor.SuccessRate(results)
//
// # Context Cancellation
//
// All operations respect context cancellation:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	results := pool.Execute(ctx)
//
// # Graceful Shutdown
//
// Shutdown the pool gracefully:
//
//	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	if err := pool.Shutdown(shutdownCtx); err != nil {
//	    log.Printf("shutdown error: %v", err)
//	}
//
// # Concurrency Guarantees
//
// The pool guarantees:
//   - Bounded concurrency (max N workers)
//   - No goroutine leaks
//   - Thread-safe submission and execution
//   - Proper cleanup on shutdown or context cancellation
//   - All submitted tasks are executed or explicitly cancelled
//
// # Performance Considerations
//
//   - Worker count should be tuned based on workload (CPU-bound vs I/O-bound)
//   - For I/O-bound tasks (API calls), higher worker counts are beneficial
//   - For CPU-bound tasks, worker count should match available CPU cores
//   - Results are collected in memory, so very large result sets may need streaming
//
// # Error Handling
//
// Task errors are captured in results and don't stop other tasks:
//
//	for _, r := range results {
//	    if r.Error != nil {
//	        log.Printf("cluster %s failed: %v", r.ClusterName, r.Error)
//	    }
//	}
//
// # Thread Safety
//
// All pool operations are thread-safe:
//   - Submit can be called concurrently (when pool is not running)
//   - Execute/ExecuteWithProgress are mutually exclusive (only one can run at a time)
//   - Shutdown can be called concurrently with Execute
//   - All accessors (TaskCount, WorkerCount, etc.) are thread-safe
package executor
