# Get Commands Implementation

This package implements the `get` commands for the Fleet multi-cluster Kubernetes management CLI.

## Overview

The get commands allow users to query Kubernetes resources across multiple clusters concurrently, with consistent output formatting and filtering options.

## Commands

### Parent Command: `fleet get`

The parent command provides a unified interface for all resource retrieval operations.

### Subcommands

1. **`fleet get pods`** - Get pods across clusters
   - Flags:
     - `-n, --namespace`: Filter by namespace
     - `-A, --all-namespaces`: Query all namespaces
     - `-l, --selector`: Label selector to filter pods
   - Output columns: CLUSTER, NAMESPACE, NAME, READY, STATUS, RESTARTS, AGE

2. **`fleet get nodes`** - Get nodes across clusters
   - Output columns: CLUSTER, NAME, STATUS, ROLES, AGE, VERSION

3. **`fleet get deployments`** - Get deployments across clusters
   - Flags:
     - `-n, --namespace`: Filter by namespace
     - `-A, --all-namespaces`: Query all namespaces
   - Output columns: CLUSTER, NAMESPACE, NAME, READY, UP-TO-DATE, AVAILABLE, AGE

4. **`fleet get services`** - Get services across clusters
   - Flags:
     - `-n, --namespace`: Filter by namespace
     - `-A, --all-namespaces`: Query all namespaces
   - Output columns: CLUSTER, NAMESPACE, NAME, TYPE, CLUSTER-IP, EXTERNAL-IP, PORT(S), AGE

5. **`fleet get namespaces`** - Get namespaces across clusters
   - Output columns: CLUSTER, NAME, STATUS, AGE

## Architecture

### Concurrent Execution

All get commands use the `executor.Pool` for concurrent execution across multiple clusters:

```go
// Create executor pool with configured parallelism
pool := executor.NewPool(parallelism, logger)

// Submit tasks for each cluster
for _, client := range clients {
    task := executor.Task{
        ClusterName: clusterName,
        Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
            return getResource(ctx, clientset, namespace, clusterName)
        },
    }
    pool.Submit(task)
}

// Execute with timeout
results := pool.Execute(execCtx)
```

### Error Handling

- **Partial Failures**: Commands continue execution even if some clusters fail
- **Error Reporting**: Failed clusters are logged with detailed error messages
- **Graceful Degradation**: Results from successful clusters are displayed

### Context-Aware Cancellation

All operations respect context cancellation:
- Global timeout from `--timeout` flag
- User interruption (Ctrl+C)
- Programmatic cancellation

### Output Formatting

Three output formats are supported:

1. **Table** (default) - kubectl-style tabular output with colors
2. **JSON** - Machine-readable JSON format
3. **YAML** - Human-readable YAML format

Format is controlled by the `--output` flag:
```bash
fleet get pods --output json
fleet get nodes -o yaml
```

## Implementation Details

### Resource Retrieval Pattern

Each get command follows this pattern:

1. **Load Configuration**: Load kubeconfig and determine target clusters
2. **Create Cluster Manager**: Connect to all target clusters
3. **Create Executor Pool**: Set up concurrent task execution
4. **Submit Tasks**: Create a task for each cluster
5. **Execute**: Run tasks concurrently with timeout
6. **Format Results**: Display results using the selected formatter

### Data Structures

Each resource has a dedicated info structure for display:

```go
type PodInfo struct {
    Cluster   string
    Namespace string
    Name      string
    Ready     string
    Status    string
    Restarts  int32
    Age       string
}
```

### Helper Functions

- `calculateAge()` - Converts creation time to human-readable age (e.g., "2h", "3d")
- `calculateReadyStatus()` - Computes ready/total containers ratio
- `getNodeRoles()` - Extracts node roles from labels
- `getServicePorts()` - Formats service ports as a string

## Testing

Comprehensive tests cover:

- **Unit Tests**: Individual function testing with fake clientsets
- **Integration Tests**: Multi-cluster scenarios with executor pool
- **Context Cancellation**: Proper handling of timeouts and cancellation
- **Partial Failures**: Handling of mixed success/failure scenarios
- **Table Formatting**: Output formatting verification
- **Benchmarks**: Performance testing

Run tests:
```bash
# Run all tests
go test ./internal/cli/get/...

# Run with verbose output
go test -v ./internal/cli/get/...

# Run with race detector
go test -race ./internal/cli/get/...

# Run benchmarks
go test -bench=. -benchmem ./internal/cli/get/...
```

## Performance

Benchmarks show:
- `BenchmarkGetPods`: ~106µs per operation with 100 pods
- `BenchmarkCalculateAge`: ~29ns per operation

## Usage Examples

### Get all pods across all clusters
```bash
fleet get pods
```

### Get pods in kube-system namespace
```bash
fleet get pods -n kube-system
```

### Get all pods across all namespaces
```bash
fleet get pods -A
```

### Get pods with label selector
```bash
fleet get pods -l app=nginx
```

### Get pods from specific clusters
```bash
fleet get pods --clusters prod-east,prod-west
```

### Get deployments in JSON format
```bash
fleet get deployments -o json
```

### Get nodes with increased parallelism
```bash
fleet get nodes --parallel 10
```

### Get services with custom timeout
```bash
fleet get services --timeout 1m
```

## Files

- `get.go` - Parent command registration
- `pods.go` - Pod retrieval implementation
- `nodes.go` - Node retrieval implementation
- `deployments.go` - Deployment retrieval implementation
- `services.go` - Service retrieval implementation
- `namespaces.go` - Namespace retrieval implementation
- `get_test.go` - Comprehensive test suite

## Dependencies

- `k8s.io/client-go` - Kubernetes client library
- `k8s.io/api` - Kubernetes API types
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/aryankumar/fleet/internal/cluster` - Cluster management
- `github.com/aryankumar/fleet/internal/executor` - Concurrent execution
- `github.com/aryankumar/fleet/internal/output` - Output formatting
- `github.com/aryankumar/fleet/internal/config` - Configuration loading

## Best Practices

1. **Context Propagation**: All API calls use context for cancellation
2. **Error Wrapping**: Errors are wrapped with context using `fmt.Errorf`
3. **Structured Logging**: Uses `slog` for structured logging
4. **Resource Cleanup**: Proper cleanup with `defer mgr.Close()`
5. **No Goroutine Leaks**: All goroutines are properly managed by executor pool
6. **Idiomatic Go**: Follows Go best practices and conventions
7. **Table-Driven Tests**: Uses table-driven test patterns

## Future Enhancements

Potential improvements:
- Watch mode for real-time updates
- Field selectors support
- Custom output columns
- Sorting options
- Pagination for large result sets
- Caching for improved performance
- Export to file capabilities
