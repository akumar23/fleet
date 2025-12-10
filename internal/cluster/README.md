# Cluster Package

The `cluster` package provides thread-safe management of connections to multiple Kubernetes clusters with comprehensive health checking and context-aware cancellation.

## Overview

This package implements Phase 3 of the Fleet CLI project, providing:
- Concurrent connection establishment to multiple Kubernetes clusters
- Thread-safe client access with RWMutex for read-heavy patterns
- Context-aware health checking with configurable timeouts
- Graceful shutdown and cleanup
- Full support for all kubeconfig authentication methods

## Components

### Types (`types.go`)

- **`Client`**: Represents a connection to a single Kubernetes cluster
  - Contains clientset, REST config, and health status
  - Thread-safe when accessed through Manager

- **`HealthStatus`**: Detailed health check results
  - Cluster name, health status, error, and server version

### Client Wrapper (`client.go`)

- **`NewClient()`**: Creates a client from a REST config
- **`HealthCheck()`**: Performs health check with 10s timeout
- **`GetServerVersion()`**: Retrieves Kubernetes server version
- **`IsHealthy()`**: Returns current health status

### Connection Manager (`manager.go`)

- **`NewManager()`**: Creates a new cluster manager
- **`Connect()`**: Connects to specific clusters concurrently (max 10 at once)
- **`ConnectAll()`**: Connects to all clusters in kubeconfig
- **`GetClient()`**: Retrieves a specific cluster client (thread-safe)
- **`GetAllClients()`**: Returns all connected clients (snapshot)
- **`GetClientNames()`**: Returns list of cluster names
- **`HasClient()`**: Checks if cluster is connected
- **`Count()`**: Returns number of connected clusters
- **`HealthCheck()`**: Concurrent health checks on all clusters
- **`HealthCheckWithStatus()`**: Detailed health status with versions
- **`Close()`**: Graceful shutdown with cleanup
- **`IsClosed()`**: Check if manager is closed

## Usage Example

```go
package main

import (
    "context"
    "log/slog"
    "os"
    
    "github.com/aryankumar/fleet/internal/cluster"
    "github.com/aryankumar/fleet/internal/config"
)

func main() {
    // Setup
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    loader := config.NewKubeconfigLoader("")
    manager := cluster.NewManager(loader, logger)
    defer manager.Close()
    
    // Connect to clusters
    ctx := context.Background()
    if err := manager.ConnectAll(ctx); err != nil {
        logger.Error("connection failed", "error", err)
        return
    }
    
    // Health check
    results := manager.HealthCheck(ctx)
    for name, err := range results {
        if err == nil {
            logger.Info("cluster healthy", "cluster", name)
        }
    }
    
    // Use a client
    client, _ := manager.GetClient("my-cluster")
    pods, _ := client.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
}
```

## Testing

The package includes comprehensive tests with 87% coverage:

```bash
# Run all tests
go test -v -race ./internal/cluster/...

# Run tests in short mode (skip slow tests)
go test -v -race -short ./internal/cluster/...

# Generate coverage report
go test -coverprofile=coverage.out ./internal/cluster/...
go tool cover -html=coverage.out
```

## Architecture Decisions

1. **RWMutex for Client Map**: Optimized for read-heavy access patterns
2. **Semaphore for Connection Limiting**: Prevents overwhelming the system (max 10 concurrent)
3. **Snapshot Pattern**: GetAllClients returns a copy to prevent external modification
4. **Context Timeouts**: 10-second timeout for health checks
5. **Error Aggregation**: Collects all errors from concurrent operations

## Thread Safety

All public methods are thread-safe:
- Read operations use RLock
- Write operations use Lock
- GetAllClients returns a snapshot (copy)
- Closed state prevents use-after-close

## Performance

- **Connection**: O(n) with parallelism limited to 10
- **GetClient**: O(1) with RLock
- **GetAllClients**: O(n) with RLock and copy
- **HealthCheck**: O(1) per cluster (all parallel)

## Dependencies

- `k8s.io/client-go/kubernetes` - Kubernetes client
- `k8s.io/client-go/rest` - REST configuration
- `github.com/aryankumar/fleet/internal/config` - Kubeconfig loading
- `log/slog` - Structured logging

## Future Integration

This package is designed to integrate with:
- Concurrent execution engine (Phase 4)
- CLI commands for multi-cluster operations (Phase 5+)
- Output formatters for result display (Phase 5)
