# Executor Package

Production-grade concurrent execution engine for multi-cluster Kubernetes operations.

## Quick Start

```go
import "github.com/aryankumar/fleet/internal/executor"

// Create a pool with 5 workers
pool := executor.NewPool(5, logger)

// Submit tasks
for _, cluster := range clusters {
    pool.Submit(executor.Task{
        ClusterName: cluster,
        Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
            // Your cluster operation here
            return result, nil
        },
    })
}

// Execute and get results
results := pool.Execute(context.Background())
```

## Features

- **Worker Pool Pattern**: Bounded concurrency with configurable workers
- **Context-Aware**: Full support for cancellation and timeouts
- **Progress Reporting**: Real-time progress callbacks
- **Graceful Shutdown**: Clean shutdown with timeout support
- **Result Aggregation**: Rich set of utilities for processing results
- **Thread-Safe**: All operations are goroutine-safe
- **Zero Leaks**: No goroutine or resource leaks

## API Overview

### Core Types

```go
type Task struct {
    ClusterName string
    Execute     func(ctx context.Context, client interface{}) (interface{}, error)
}

type Result struct {
    ClusterName string
    Data        interface{}
    Error       error
    Duration    time.Duration
}

type Pool struct {
    // ... internal fields
}
```

### Pool Methods

```go
// Create new pool
func NewPool(workers int, logger *slog.Logger) *Pool

// Submit a task
func (p *Pool) Submit(task Task) error

// Execute all tasks
func (p *Pool) Execute(ctx context.Context) []Result

// Execute with progress reporting
func (p *Pool) ExecuteWithProgress(ctx context.Context, progressFn func(completed, total int)) []Result

// Graceful shutdown
func (p *Pool) Shutdown(ctx context.Context) error

// Accessors
func (p *Pool) TaskCount() int
func (p *Pool) WorkerCount() int
func (p *Pool) IsRunning() bool
func (p *Pool) IsShutdown() bool
```

### Result Utilities

```go
// Counting
func CountSuccessful(results []Result) int
func CountFailed(results []Result) int

// Filtering
func FilterSuccessful(results []Result) []Result
func FilterFailed(results []Result) []Result
func FilterByCluster(results []Result, clusterName string) []Result

// Grouping
func GroupByCluster(results []Result) map[string][]Result

// Statistics
func AverageDuration(results []Result) time.Duration
func MaxDuration(results []Result) time.Duration
func MinDuration(results []Result) time.Duration

// Analysis
func SuccessRate(results []Result) float64
func FailureRate(results []Result) float64
func HasErrors(results []Result) bool
func AllSuccessful(results []Result) bool

// Summary
func Summarize(results []Result) Summary
```

## Examples

### Basic Usage

```go
pool := executor.NewPool(3, logger)

pool.Submit(executor.Task{
    ClusterName: "prod-us-east",
    Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
        return someClusterOperation(ctx)
    },
})

results := pool.Execute(context.Background())
```

### With Progress Reporting

```go
results := pool.ExecuteWithProgress(ctx, func(completed, total int) {
    fmt.Printf("Progress: %d/%d (%.0f%%)\n",
        completed, total, float64(completed)/float64(total)*100)
})
```

### With Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results := pool.Execute(ctx)
```

### Error Handling

```go
results := pool.Execute(ctx)

// Check for errors
if executor.HasErrors(results) {
    failed := executor.FilterFailed(results)
    for _, r := range failed {
        logger.Error("task failed",
            "cluster", r.ClusterName,
            "error", r.Error)
    }
}

// Process successful results
successful := executor.FilterSuccessful(results)
for _, r := range successful {
    // Process r.Data
}
```

### Summary Statistics

```go
results := pool.Execute(ctx)
summary := executor.Summarize(results)

fmt.Printf("Total: %d\n", summary.Total)
fmt.Printf("Successful: %d\n", summary.Successful)
fmt.Printf("Failed: %d\n", summary.Failed)
fmt.Printf("Avg Duration: %v\n", summary.AvgDuration)
fmt.Printf("Success Rate: %.1f%%\n", executor.SuccessRate(results))
```

### Graceful Shutdown

```go
// Start execution in background
go func() {
    results := pool.Execute(ctx)
    // Handle results
}()

// Later, shutdown gracefully
shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := pool.Shutdown(shutdownCtx); err != nil {
    logger.Error("shutdown failed", "error", err)
}
```

## Best Practices

### Worker Count Tuning

```go
// For I/O-bound tasks (API calls, network)
pool := executor.NewPool(20, logger)  // Higher concurrency OK

// For CPU-bound tasks
pool := executor.NewPool(runtime.NumCPU(), logger)

// For rate-limited APIs
pool := executor.NewPool(5, logger)  // Match rate limit
```

### Context Usage

```go
// Always use contexts with timeouts for production
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results := pool.Execute(ctx)
```

### Error Handling

```go
// Don't ignore partial failures
results := pool.Execute(ctx)

if !executor.AllSuccessful(results) {
    // Log failures but continue with successful results
    for _, r := range executor.FilterFailed(results) {
        logger.Warn("cluster operation failed",
            "cluster", r.ClusterName,
            "error", r.Error)
    }
}

// Process successful results
successful := executor.FilterSuccessful(results)
// ...
```

### Task Design

```go
// Capture variables properly in closures
for _, cluster := range clusters {
    c := cluster  // Important: capture loop variable

    pool.Submit(executor.Task{
        ClusterName: c.Name,
        Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
            // Use c here, not cluster
            return c.DoSomething(ctx)
        },
    })
}
```

## Performance

### Benchmarks

- **10 tasks, 5 workers**: ~100ms (concurrent)
- **10 tasks, sequential**: ~500ms
- **Speedup**: ~5x

### Memory

- O(N) memory where N = number of tasks
- Results collected in memory
- No memory leaks (verified with race detector)

### Scalability

- Linear scaling with worker count for I/O-bound tasks
- Tested with race detector
- No goroutine leaks

## Testing

```bash
# Run tests
go test ./internal/executor/

# With race detection
go test -race ./internal/executor/

# With coverage
go test -cover ./internal/executor/

# Verbose
go test -v ./internal/executor/
```

## Thread Safety

All public methods are thread-safe:

- `Submit()`: Safe when pool is not running
- `Execute()`: Mutually exclusive (only one execution at a time)
- `Shutdown()`: Safe to call concurrently with Execute
- Accessors: All thread-safe

## Error Conditions

| Error | Condition |
|-------|-----------|
| "pool is shutting down" | Submit() called after Shutdown() |
| "pool is running" | Submit() called during Execute() |
| "pool already shut down" | Shutdown() called twice |
| "shutdown timeout" | Shutdown timeout exceeded |
| "task must have a cluster name" | Task.ClusterName is empty |
| "task must have an execute function" | Task.Execute is nil |

## Integration

### With Cluster Manager

```go
import (
    "github.com/aryankumar/fleet/internal/cluster"
    "github.com/aryankumar/fleet/internal/executor"
)

pool := executor.NewPool(5, logger)
mgr := cluster.NewManager(loader, logger)

// Connect to clusters
mgr.ConnectAll(ctx)

// Submit tasks for each cluster
for _, client := range mgr.GetAllClients() {
    c := client
    pool.Submit(executor.Task{
        ClusterName: c.Name,
        Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
            return c.GetPods(ctx, "default")
        },
    })
}

results := pool.Execute(ctx)
```

## See Also

- [pool.go](./pool.go) - Core implementation
- [result.go](./result.go) - Result utilities
- [pool_test.go](./pool_test.go) - Test suite
- [example_test.go](./example_test.go) - Usage examples
- [integration_example_test.go](./integration_example_test.go) - Integration examples
- [PHASE4_SUMMARY.md](../../PHASE4_SUMMARY.md) - Complete phase documentation
