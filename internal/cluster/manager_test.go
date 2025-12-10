package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/aryankumar/fleet/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

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

func TestNewManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")

	manager := NewManager(loader, logger)

	if manager == nil {
		t.Fatal("expected manager, got nil")
	}

	if manager.loader != loader {
		t.Error("expected loader to be set")
	}

	if manager.logger == nil {
		t.Error("expected logger to be set")
	}

	if manager.clients == nil {
		t.Error("expected clients map to be initialized")
	}

	if manager.closed {
		t.Error("expected closed to be false")
	}
}

func TestNewManager_NilLogger(t *testing.T) {
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, nil)

	if manager.logger == nil {
		t.Error("expected default logger when nil is provided")
	}
}

func TestManager_Connect(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name         string
		clusterNames []string
		wantErr      bool
		setupFunc    func() *config.KubeconfigLoader
	}{
		{
			name:         "empty cluster names",
			clusterNames: []string{},
			wantErr:      true,
			setupFunc: func() *config.KubeconfigLoader {
				kubeconfigPath := createTestKubeconfig(t, []string{"cluster1"})
				return config.NewKubeconfigLoader(kubeconfigPath)
			},
		},
		{
			name:         "single cluster",
			clusterNames: []string{"cluster1"},
			wantErr:      false, // Will succeed creating client (connection not tested)
			setupFunc: func() *config.KubeconfigLoader {
				kubeconfigPath := createTestKubeconfig(t, []string{"cluster1"})
				return config.NewKubeconfigLoader(kubeconfigPath)
			},
		},
		{
			name:         "multiple clusters",
			clusterNames: []string{"cluster1", "cluster2", "cluster3"},
			wantErr:      false, // Will succeed creating clients
			setupFunc: func() *config.KubeconfigLoader {
				kubeconfigPath := createTestKubeconfig(t, []string{"cluster1", "cluster2", "cluster3"})
				return config.NewKubeconfigLoader(kubeconfigPath)
			},
		},
		{
			name:         "nonexistent cluster",
			clusterNames: []string{"nonexistent"},
			wantErr:      true,
			setupFunc: func() *config.KubeconfigLoader {
				kubeconfigPath := createTestKubeconfig(t, []string{"cluster1"})
				return config.NewKubeconfigLoader(kubeconfigPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := tt.setupFunc()
			manager := NewManager(loader, logger)
			ctx := context.Background()

			err := manager.Connect(ctx, tt.clusterNames)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestManager_Connect_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	kubeconfigPath := createTestKubeconfig(t, []string{"cluster1", "cluster2"})
	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := NewManager(loader, logger)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := manager.Connect(ctx, []string{"cluster1", "cluster2"})

	if err == nil {
		t.Error("expected error from cancelled context")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got: %v", err)
	}
}

func TestManager_ConnectAll(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	kubeconfigPath := createTestKubeconfig(t, []string{"cluster1", "cluster2", "cluster3"})
	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := NewManager(loader, logger)

	ctx := context.Background()
	err := manager.ConnectAll(ctx)

	// Should succeed in creating clients (actual connection isn't tested)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify all clusters are connected
	if manager.Count() != 3 {
		t.Errorf("expected 3 connected clusters, got %d", manager.Count())
	}
}

func TestManager_GetClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	// Manually add a fake client
	fakeClientset := fake.NewSimpleClientset()
	manager.clients["test-cluster"] = &Client{
		Name:      "test-cluster",
		Context:   "test-context",
		Clientset: fakeClientset,
	}

	tests := []struct {
		name        string
		clusterName string
		wantErr     bool
	}{
		{
			name:        "existing cluster",
			clusterName: "test-cluster",
			wantErr:     false,
		},
		{
			name:        "non-existent cluster",
			clusterName: "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := manager.GetClient(tt.clusterName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Fatal("expected client, got nil")
			}

			if client.Name != tt.clusterName {
				t.Errorf("expected name %s, got %s", tt.clusterName, client.Name)
			}
		})
	}
}

func TestManager_GetClient_Closed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	manager.Close()

	_, err := manager.GetClient("test-cluster")
	if err == nil {
		t.Error("expected error from closed manager")
	}

	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("expected 'closed' in error message, got: %v", err)
	}
}

func TestManager_GetAllClients(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	// Add multiple fake clients
	for i := 1; i <= 3; i++ {
		fakeClientset := fake.NewSimpleClientset()
		clusterName := fmt.Sprintf("cluster%d", i)
		manager.clients[clusterName] = &Client{
			Name:      clusterName,
			Context:   clusterName,
			Clientset: fakeClientset,
		}
	}

	clients := manager.GetAllClients()

	if len(clients) != 3 {
		t.Errorf("expected 3 clients, got %d", len(clients))
	}

	// Verify we got all cluster names
	names := make(map[string]bool)
	for _, client := range clients {
		names[client.Name] = true
	}

	for i := 1; i <= 3; i++ {
		clusterName := fmt.Sprintf("cluster%d", i)
		if !names[clusterName] {
			t.Errorf("expected to find cluster %s", clusterName)
		}
	}
}

func TestManager_GetClientNames(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	// Add fake clients
	expectedNames := []string{"cluster1", "cluster2", "cluster3"}
	for _, name := range expectedNames {
		fakeClientset := fake.NewSimpleClientset()
		manager.clients[name] = &Client{
			Name:      name,
			Context:   name,
			Clientset: fakeClientset,
		}
	}

	names := manager.GetClientNames()

	if len(names) != len(expectedNames) {
		t.Errorf("expected %d names, got %d", len(expectedNames), len(names))
	}
}

func TestManager_HasClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	fakeClientset := fake.NewSimpleClientset()
	manager.clients["test-cluster"] = &Client{
		Name:      "test-cluster",
		Context:   "test-context",
		Clientset: fakeClientset,
	}

	if !manager.HasClient("test-cluster") {
		t.Error("expected HasClient to return true")
	}

	if manager.HasClient("nonexistent") {
		t.Error("expected HasClient to return false")
	}
}

func TestManager_Count(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	if manager.Count() != 0 {
		t.Errorf("expected count 0, got %d", manager.Count())
	}

	// Add clients
	for i := 1; i <= 5; i++ {
		fakeClientset := fake.NewSimpleClientset()
		clusterName := fmt.Sprintf("cluster%d", i)
		manager.clients[clusterName] = &Client{
			Name:      clusterName,
			Context:   clusterName,
			Clientset: fakeClientset,
		}
	}

	if manager.Count() != 5 {
		t.Errorf("expected count 5, got %d", manager.Count())
	}
}

func TestManager_HealthCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	// Add fake clients with different health statuses
	for i := 1; i <= 3; i++ {
		fakeClientset := fake.NewSimpleClientset()
		clusterName := fmt.Sprintf("cluster%d", i)

		// Set up fake discovery
		fakeDiscovery := fakeClientset.Discovery().(*fakediscovery.FakeDiscovery)
		if i <= 2 {
			// First two clusters are healthy
			fakeDiscovery.FakedServerVersion = &version.Info{
				Major:      "1",
				Minor:      "28",
				GitVersion: fmt.Sprintf("v1.28.%d", i),
			}
		} else {
			// Third cluster fails
			fakeDiscovery.PrependReactor("get", "version",
				func(action k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("connection refused")
				})
		}

		manager.clients[clusterName] = &Client{
			Name:      clusterName,
			Context:   clusterName,
			Clientset: fakeClientset,
		}
	}

	ctx := context.Background()
	results := manager.HealthCheck(ctx)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Check that first two passed
	for i := 1; i <= 2; i++ {
		clusterName := fmt.Sprintf("cluster%d", i)
		if err, ok := results[clusterName]; ok {
			if err != nil {
				t.Errorf("expected cluster%d to be healthy, got error: %v", i, err)
			}
		} else {
			t.Errorf("expected result for cluster%d", i)
		}
	}

	// Check that third failed
	if err, ok := results["cluster3"]; ok {
		if err == nil {
			t.Error("expected cluster3 to fail health check")
		}
	} else {
		t.Error("expected result for cluster3")
	}
}

func TestManager_HealthCheck_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	// Add a fake client
	fakeClientset := fake.NewSimpleClientset()
	manager.clients["test-cluster"] = &Client{
		Name:      "test-cluster",
		Context:   "test-context",
		Clientset: fakeClientset,
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := manager.HealthCheck(ctx)

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if err, ok := results["test-cluster"]; ok {
		if err == nil {
			t.Error("expected error from cancelled context")
		}
	}
}

func TestManager_HealthCheckWithStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	// Add fake clients
	for i := 1; i <= 2; i++ {
		fakeClientset := fake.NewSimpleClientset()
		clusterName := fmt.Sprintf("cluster%d", i)

		fakeDiscovery := fakeClientset.Discovery().(*fakediscovery.FakeDiscovery)
		fakeDiscovery.FakedServerVersion = &version.Info{
			Major:      "1",
			Minor:      "28",
			GitVersion: fmt.Sprintf("v1.28.%d", i),
		}

		manager.clients[clusterName] = &Client{
			Name:      clusterName,
			Context:   clusterName,
			Clientset: fakeClientset,
		}
	}

	ctx := context.Background()
	statuses := manager.HealthCheckWithStatus(ctx)

	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}

	for _, status := range statuses {
		if !status.Healthy {
			t.Errorf("expected cluster %s to be healthy", status.ClusterName)
		}

		if status.ServerVersion == "" {
			t.Errorf("expected server version for cluster %s", status.ClusterName)
		}

		if status.Error != nil {
			t.Errorf("expected no error for cluster %s, got: %v", status.ClusterName, status.Error)
		}
	}
}

func TestManager_Close(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	// Add some clients
	for i := 1; i <= 3; i++ {
		fakeClientset := fake.NewSimpleClientset()
		clusterName := fmt.Sprintf("cluster%d", i)
		manager.clients[clusterName] = &Client{
			Name:      clusterName,
			Context:   clusterName,
			Clientset: fakeClientset,
		}
	}

	if manager.Count() != 3 {
		t.Errorf("expected 3 clients before close, got %d", manager.Count())
	}

	manager.Close()

	if manager.Count() != 0 {
		t.Errorf("expected 0 clients after close, got %d", manager.Count())
	}

	if !manager.IsClosed() {
		t.Error("expected IsClosed to return true")
	}

	// Test double close
	manager.Close() // Should not panic
}

func TestManager_IsClosed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	if manager.IsClosed() {
		t.Error("expected IsClosed to return false initially")
	}

	manager.Close()

	if !manager.IsClosed() {
		t.Error("expected IsClosed to return true after Close")
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent access test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader("")
	manager := NewManager(loader, logger)

	// Add some initial clients
	for i := 1; i <= 5; i++ {
		fakeClientset := fake.NewSimpleClientset()
		clusterName := fmt.Sprintf("cluster%d", i)
		manager.clients[clusterName] = &Client{
			Name:      clusterName,
			Context:   clusterName,
			Clientset: fakeClientset,
		}
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			clusterName := fmt.Sprintf("cluster%d", (id%5)+1)
			_, err := manager.GetClient(clusterName)
			if err != nil {
				errCh <- fmt.Errorf("read %d: %w", id, err)
			}
		}(i)
	}

	// Concurrent GetAllClients
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			clients := manager.GetAllClients()
			if len(clients) == 0 {
				errCh <- fmt.Errorf("getAll %d: got 0 clients", id)
			}
		}(i)
	}

	// Concurrent Count
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.Count()
		}()
	}

	wg.Wait()
	close(errCh)

	// Check for errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("concurrent access errors: %v", errors)
	}
}

func TestCountHealthy(t *testing.T) {
	tests := []struct {
		name     string
		results  map[string]error
		expected int
	}{
		{
			name:     "all healthy",
			results:  map[string]error{"c1": nil, "c2": nil, "c3": nil},
			expected: 3,
		},
		{
			name:     "some unhealthy",
			results:  map[string]error{"c1": nil, "c2": fmt.Errorf("error"), "c3": nil},
			expected: 2,
		},
		{
			name:     "all unhealthy",
			results:  map[string]error{"c1": fmt.Errorf("error"), "c2": fmt.Errorf("error")},
			expected: 0,
		},
		{
			name:     "empty results",
			results:  map[string]error{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countHealthy(tt.results)
			if count != tt.expected {
				t.Errorf("expected %d healthy, got %d", tt.expected, count)
			}
		})
	}
}
