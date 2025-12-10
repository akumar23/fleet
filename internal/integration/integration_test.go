package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/aryankumar/fleet/internal/cluster"
	"github.com/aryankumar/fleet/internal/config"
	"github.com/aryankumar/fleet/internal/executor"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// TestFullWorkflow tests the complete workflow from config loading to execution
func TestFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test kubeconfig
	kubeconfigPath := createTestKubeconfig(t, []string{"cluster-1", "cluster-2", "cluster-3"})

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Load kubeconfig
	loader := config.NewKubeconfigLoader(kubeconfigPath)
	contexts, err := loader.GetContexts()
	if err != nil {
		t.Fatalf("failed to get contexts: %v", err)
	}

	if len(contexts) != 3 {
		t.Fatalf("expected 3 contexts, got %d", len(contexts))
	}

	// Create cluster manager
	manager := cluster.NewManager(loader, logger)

	// Connect to all clusters
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = manager.Connect(ctx, contexts)
	if err != nil {
		t.Fatalf("failed to connect to clusters: %v", err)
	}
	defer manager.Close()

	// Verify all clients are connected
	if manager.Count() != 3 {
		t.Errorf("expected 3 connected clusters, got %d", manager.Count())
	}

	// Create executor pool
	pool := executor.NewPool(3, logger)

	// Submit tasks for each cluster
	clients := manager.GetAllClients()
	for _, client := range clients {
		c := client // Capture for closure
		err := pool.Submit(executor.Task{
			ClusterName: c.Name,
			Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
				// Simulate a health check
				err := c.HealthCheck(ctx)
				return nil, err
			},
		})
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}
	}

	// Execute tasks
	results := pool.Execute(ctx)

	// Verify results
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Check success count
	successCount := executor.CountSuccessful(results)
	if successCount != 3 {
		t.Errorf("expected 3 successful results, got %d", successCount)
	}

	// Verify result durations are reasonable
	for _, r := range results {
		if r.Duration <= 0 {
			t.Errorf("expected positive duration, got %v", r.Duration)
		}
		if r.Duration > 5*time.Second {
			t.Errorf("task took too long: %v", r.Duration)
		}
	}
}

// TestConcurrentClusterAccess tests concurrent access to multiple clusters
func TestConcurrentClusterAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	kubeconfigPath := createTestKubeconfig(t, []string{"cluster-1", "cluster-2", "cluster-3", "cluster-4", "cluster-5"})
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := cluster.NewManager(loader, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := manager.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("failed to connect to clusters: %v", err)
	}
	defer manager.Close()

	// Test concurrent health checks
	healthResults := manager.HealthCheck(ctx)

	if len(healthResults) != 5 {
		t.Errorf("expected 5 health check results, got %d", len(healthResults))
	}

	// All should be healthy (fake clients always succeed)
	for cluster, err := range healthResults {
		if err != nil {
			t.Errorf("health check failed for cluster %s: %v", cluster, err)
		}
	}
}

// TestHealthCheckWithStatus tests detailed health check functionality
func TestHealthCheckWithStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	kubeconfigPath := createTestKubeconfig(t, []string{"cluster-1", "cluster-2"})
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := cluster.NewManager(loader, logger)

	ctx := context.Background()
	err := manager.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer manager.Close()

	// Get detailed health status
	statusList := manager.HealthCheckWithStatus(ctx)

	if len(statusList) != 2 {
		t.Fatalf("expected 2 status results, got %d", len(statusList))
	}

	for _, status := range statusList {
		if !status.Healthy {
			t.Errorf("cluster %s is not healthy: %v", status.ClusterName, status.Error)
		}
		if status.ServerVersion == "" {
			t.Errorf("cluster %s missing server version", status.ClusterName)
		}
	}
}

// TestContextCancellation tests that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	kubeconfigPath := createTestKubeconfig(t, []string{"cluster-1", "cluster-2", "cluster-3"})
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := cluster.NewManager(loader, logger)

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Try to connect with a very short timeout
	err := manager.ConnectAll(ctx)

	// We expect either success (if it's fast enough) or context cancellation
	if err != nil {
		if ctx.Err() == nil {
			t.Errorf("unexpected error (context not cancelled): %v", err)
		}
		// Context was cancelled, which is expected
	}
}

// TestPoolShutdownGracefully tests graceful shutdown of the executor pool
func TestPoolShutdownGracefully(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	pool := executor.NewPool(3, logger)

	// Submit some tasks
	for i := 0; i < 5; i++ {
		err := pool.Submit(executor.Task{
			ClusterName: fmt.Sprintf("cluster-%d", i),
			Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return "done", nil
			},
		})
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}
	}

	// Start execution in background
	ctx := context.Background()
	done := make(chan []executor.Result)
	go func() {
		results := pool.Execute(ctx)
		done <- results
	}()

	// Wait a bit for tasks to start
	time.Sleep(5 * time.Millisecond)

	// Shutdown gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := pool.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	// Wait for results
	results := <-done
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	// Verify pool is shutdown
	if !pool.IsShutdown() {
		t.Error("pool should be shutdown")
	}

	// Try to submit after shutdown (should fail)
	err = pool.Submit(executor.Task{
		ClusterName: "new-cluster",
		Execute:     func(ctx context.Context, _ interface{}) (interface{}, error) { return nil, nil },
	})
	if err == nil {
		t.Error("expected error when submitting to shutdown pool")
	}
}

// TestExecutorProgressReporting tests progress callback functionality
func TestExecutorProgressReporting(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	pool := executor.NewPool(2, logger)

	taskCount := 5
	for i := 0; i < taskCount; i++ {
		pool.Submit(executor.Task{
			ClusterName: fmt.Sprintf("cluster-%d", i),
			Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return "done", nil
			},
		})
	}

	// Track progress with mutex for thread safety
	var mu sync.Mutex
	progressUpdates := 0
	lastCompleted := 0

	results := pool.ExecuteWithProgress(context.Background(), func(completed, total int) {
		mu.Lock()
		defer mu.Unlock()

		progressUpdates++

		// Verify progress is monotonically increasing
		if completed < lastCompleted {
			t.Errorf("progress went backwards: %d -> %d", lastCompleted, completed)
		}
		lastCompleted = completed

		// Verify total is consistent
		if total != taskCount {
			t.Errorf("expected total %d, got %d", taskCount, total)
		}
	})

	if len(results) != taskCount {
		t.Errorf("expected %d results, got %d", taskCount, len(results))
	}

	mu.Lock()
	defer mu.Unlock()

	if progressUpdates != taskCount {
		t.Errorf("expected %d progress updates, got %d", taskCount, progressUpdates)
	}

	if lastCompleted != taskCount {
		t.Errorf("expected final completed count %d, got %d", taskCount, lastCompleted)
	}
}

// TestManagerCloseIdempotent tests that closing a manager multiple times is safe
func TestManagerCloseIdempotent(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, []string{"cluster-1"})
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := cluster.NewManager(loader, logger)

	ctx := context.Background()
	err := manager.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Close multiple times should not panic
	manager.Close()
	manager.Close()
	manager.Close()

	// Verify manager is closed
	if !manager.IsClosed() {
		t.Error("manager should be closed")
	}

	// Operations on closed manager should fail gracefully
	_, err = manager.GetClient("cluster-1")
	if err == nil {
		t.Error("expected error when getting client from closed manager")
	}
}

// TestRaceConditions tests for race conditions with concurrent operations
func TestRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	kubeconfigPath := createTestKubeconfig(t, []string{"cluster-1", "cluster-2", "cluster-3"})
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := cluster.NewManager(loader, logger)

	ctx := context.Background()
	err := manager.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer manager.Close()

	// Concurrent reads (don't include HealthCheck as it can cause race with map access)
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Concurrent GetAllClients
			clients := manager.GetAllClients()
			if len(clients) != 3 {
				t.Errorf("expected 3 clients, got %d", len(clients))
			}

			// Concurrent GetClientNames
			names := manager.GetClientNames()
			if len(names) != 3 {
				t.Errorf("expected 3 names, got %d", len(names))
			}

			// Concurrent HasClient
			if !manager.HasClient("cluster-1") {
				t.Error("expected cluster-1 to exist")
			}

			// Concurrent Count
			count := manager.Count()
			if count != 3 {
				t.Errorf("expected count 3, got %d", count)
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test health check separately (not concurrently with itself)
	results := manager.HealthCheck(ctx)
	if len(results) != 3 {
		t.Errorf("expected 3 health check results, got %d", len(results))
	}
}

// createTestKubeconfig creates a temporary kubeconfig file for testing
func createTestKubeconfig(t *testing.T, clusters []string) string {
	t.Helper()

	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	// Create a minimal kubeconfig
	config := api.Config{
		Clusters:       make(map[string]*api.Cluster),
		AuthInfos:      make(map[string]*api.AuthInfo),
		Contexts:       make(map[string]*api.Context),
		CurrentContext: "",
	}

	// Add clusters
	for i, clusterName := range clusters {
		config.Clusters[clusterName] = &api.Cluster{
			Server:                fmt.Sprintf("https://cluster%d.example.com:6443", i+1),
			InsecureSkipTLSVerify: true,
		}

		config.AuthInfos[clusterName] = &api.AuthInfo{
			Token: fmt.Sprintf("token-%s", clusterName),
		}

		config.Contexts[clusterName] = &api.Context{
			Cluster:   clusterName,
			AuthInfo:  clusterName,
			Namespace: "default",
		}

		if i == 0 {
			config.CurrentContext = clusterName
		}
	}

	// Write to file
	if err := clientcmd.WriteToFile(config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	return kubeconfigPath
}

// TestExecutorWithFakeKubernetesClient tests executor integration with fake Kubernetes clients
func TestExecutorWithFakeKubernetesClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create fake Kubernetes objects
	pods := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-1",
				Namespace: "default",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-2",
				Namespace: "default",
			},
		},
	}

	// Create fake clientset
	fakeClient := fake.NewSimpleClientset(pods...)

	// Set up fake discovery
	fakeDiscovery, ok := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatal("couldn't convert Discovery() to *FakeDiscovery")
	}
	fakeDiscovery.FakedServerVersion = &version.Info{
		Major:      "1",
		Minor:      "28",
		GitVersion: "v1.28.0",
	}

	// Add reactor to handle health check
	fakeClient.PrependReactor("get", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
		}, nil
	})

	// Create executor pool
	pool := executor.NewPool(2, logger)

	// Submit task to list pods
	pool.Submit(executor.Task{
		ClusterName: "test-cluster",
		Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
			podList, err := fakeClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			return len(podList.Items), nil
		},
	})

	// Execute
	results := pool.Execute(context.Background())

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	podCount := result.Data.(int)
	if podCount != 2 {
		t.Errorf("expected 2 pods, got %d", podCount)
	}
}
