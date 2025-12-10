package cluster_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aryankumar/fleet/internal/cluster"
	"github.com/aryankumar/fleet/internal/config"
)

// Example demonstrates basic usage of the cluster manager
func Example() {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a kubeconfig loader
	// This will use the default ~/.kube/config or KUBECONFIG env var
	loader := config.NewKubeconfigLoader("")

	// Create the cluster manager
	manager := cluster.NewManager(loader, logger)
	defer manager.Close()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to all clusters in the kubeconfig
	if err := manager.ConnectAll(ctx); err != nil {
		logger.Error("failed to connect to clusters", "error", err)
		return
	}

	fmt.Printf("Connected to %d clusters\n", manager.Count())

	// Get all connected clients
	clients := manager.GetAllClients()
	for _, client := range clients {
		fmt.Printf("Cluster: %s (Context: %s)\n", client.Name, client.Context)
	}

	// Perform health checks
	healthResults := manager.HealthCheck(ctx)
	for clusterName, err := range healthResults {
		if err != nil {
			fmt.Printf("Cluster %s: UNHEALTHY - %v\n", clusterName, err)
		} else {
			fmt.Printf("Cluster %s: HEALTHY\n", clusterName)
		}
	}

	// Get detailed health status
	statuses := manager.HealthCheckWithStatus(ctx)
	for _, status := range statuses {
		if status.Healthy {
			fmt.Printf("Cluster %s is healthy (version: %s)\n",
				status.ClusterName, status.ServerVersion)
		}
	}
}

// Example_specificClusters demonstrates connecting to specific clusters
func Example_specificClusters() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	loader := config.NewKubeconfigLoader("")
	manager := cluster.NewManager(loader, logger)
	defer manager.Close()

	ctx := context.Background()

	// Connect to specific clusters
	clusterNames := []string{"production-us-east", "production-eu-west"}
	if err := manager.Connect(ctx, clusterNames); err != nil {
		logger.Error("failed to connect", "error", err)
		return
	}

	// Use a specific client
	client, err := manager.GetClient("production-us-east")
	if err != nil {
		logger.Error("failed to get client", "error", err)
		return
	}

	// Now you can use client.Clientset for Kubernetes operations
	fmt.Printf("Got client for cluster: %s\n", client.Name)
}

// Example_contextCancellation demonstrates context cancellation
func Example_contextCancellation() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	loader := config.NewKubeconfigLoader("")
	manager := cluster.NewManager(loader, logger)
	defer manager.Close()

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// If connection takes too long, it will be cancelled
	if err := manager.ConnectAll(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Connection timed out")
		} else {
			fmt.Printf("Connection failed: %v\n", err)
		}
		return
	}

	fmt.Println("Successfully connected")
}

// Example_healthCheck demonstrates health checking
func Example_healthCheck() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	loader := config.NewKubeconfigLoader("")
	manager := cluster.NewManager(loader, logger)
	defer manager.Close()

	ctx := context.Background()

	// Assume clusters are already connected
	// (In real usage, you would call Connect or ConnectAll first)

	// Simple health check
	results := manager.HealthCheck(ctx)
	healthyCount := 0
	for _, err := range results {
		if err == nil {
			healthyCount++
		}
	}

	fmt.Printf("%d/%d clusters are healthy\n", healthyCount, len(results))

	// Detailed health check with server versions
	statuses := manager.HealthCheckWithStatus(ctx)
	for _, status := range statuses {
		if status.Healthy {
			fmt.Printf("%s: %s\n", status.ClusterName, status.ServerVersion)
		} else {
			fmt.Printf("%s: ERROR - %v\n", status.ClusterName, status.Error)
		}
	}
}
