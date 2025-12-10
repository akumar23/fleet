package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

func TestNewClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name        string
		clusterName string
		contextName string
		restConfig  *rest.Config
		wantErr     bool
	}{
		{
			name:        "valid config",
			clusterName: "test-cluster",
			contextName: "test-context",
			restConfig: &rest.Config{
				Host: "https://localhost:6443",
			},
			wantErr: false,
		},
		{
			name:        "nil rest config",
			clusterName: "test-cluster",
			contextName: "test-context",
			restConfig:  nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewClient(ctx, tt.clusterName, tt.contextName, tt.restConfig, logger)

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

			if client.Context != tt.contextName {
				t.Errorf("expected context %s, got %s", tt.contextName, client.Context)
			}

			if client.Healthy {
				t.Error("expected Healthy to be false initially")
			}
		})
	}
}

func TestClient_HealthCheck(t *testing.T) {
	_ = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name      string
		setupFunc func() *Client
		wantErr   bool
	}{
		{
			name: "successful health check",
			setupFunc: func() *Client {
				fakeClient := fake.NewSimpleClientset()
				// Set up fake discovery to return a version
				fakeDiscovery, ok := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
				if ok {
					fakeDiscovery.FakedServerVersion = &version.Info{
						Major:        "1",
						Minor:        "28",
						GitVersion:   "v1.28.0",
						GitCommit:    "abc123",
						GitTreeState: "clean",
						BuildDate:    "2023-01-01T00:00:00Z",
						GoVersion:    "go1.21.0",
						Compiler:     "gc",
						Platform:     "linux/amd64",
					}
				}

				return &Client{
					Name:      "test-cluster",
					Context:   "test-context",
					Clientset: fakeClient,
					RestConfig: &rest.Config{
						Host: "https://localhost:6443",
					},
					Healthy: false,
				}
			},
			wantErr: false,
		},
		{
			name: "failed health check",
			setupFunc: func() *Client {
				fakeClient := fake.NewSimpleClientset()
				// Make discovery fail
				fakeClient.Discovery().(*fakediscovery.FakeDiscovery).PrependReactor("get", "version",
					func(action k8stesting.Action) (bool, runtime.Object, error) {
						return true, nil, fmt.Errorf("connection refused")
					})

				return &Client{
					Name:      "test-cluster",
					Context:   "test-context",
					Clientset: fakeClient,
					RestConfig: &rest.Config{
						Host: "https://localhost:6443",
					},
					Healthy: false,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupFunc()
			ctx := context.Background()

			err := client.HealthCheck(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if client.Healthy {
					t.Error("expected Healthy to be false after failed check")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !client.Healthy {
				t.Error("expected Healthy to be true after successful check")
			}
		})
	}
}

func TestClient_HealthCheck_ContextCancellation(t *testing.T) {
	_ = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	fakeClient := fake.NewSimpleClientset()

	client := &Client{
		Name:      "test-cluster",
		Context:   "test-context",
		Clientset: fakeClient,
		RestConfig: &rest.Config{
			Host: "https://localhost:6443",
		},
		Healthy: false,
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.HealthCheck(ctx)

	if err == nil {
		t.Error("expected error from cancelled context")
	}

	if client.Healthy {
		t.Error("expected Healthy to be false after cancelled context")
	}
}

func TestClient_HealthCheck_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	_ = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	fakeClient := fake.NewSimpleClientset()

	// Make discovery hang
	fakeDiscovery := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.PrependReactor("get", "version",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			time.Sleep(15 * time.Second) // Longer than health check timeout
			return true, nil, nil
		})

	client := &Client{
		Name:      "test-cluster",
		Context:   "test-context",
		Clientset: fakeClient,
		RestConfig: &rest.Config{
			Host: "https://localhost:6443",
		},
		Healthy: false,
	}

	ctx := context.Background()
	err := client.HealthCheck(ctx)

	if err == nil {
		t.Error("expected timeout error")
	}

	if client.Healthy {
		t.Error("expected Healthy to be false after timeout")
	}
}

func TestClient_GetServerVersion(t *testing.T) {
	_ = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	fakeClient := fake.NewSimpleClientset()
	fakeDiscovery := fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscovery.FakedServerVersion = &version.Info{
		Major:      "1",
		Minor:      "28",
		GitVersion: "v1.28.0",
	}

	client := &Client{
		Name:      "test-cluster",
		Context:   "test-context",
		Clientset: fakeClient,
		RestConfig: &rest.Config{
			Host: "https://localhost:6443",
		},
	}

	ctx := context.Background()
	version, err := client.GetServerVersion(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if version == "" {
		t.Error("expected non-empty version")
	}

	// Should contain the version number
	if !contains(version, "1.28") && !contains(version, "v1.28.0") {
		t.Errorf("expected version to contain '1.28', got %s", version)
	}
}

func TestClient_IsHealthy(t *testing.T) {
	client := &Client{
		Name:    "test-cluster",
		Context: "test-context",
		Healthy: true,
	}

	if !client.IsHealthy() {
		t.Error("expected IsHealthy to return true")
	}

	client.Healthy = false
	if client.IsHealthy() {
		t.Error("expected IsHealthy to return false")
	}
}

func TestClient_String(t *testing.T) {
	client := &Client{
		Name:    "test-cluster",
		Context: "test-context",
		Healthy: true,
	}

	str := client.String()

	if str == "" {
		t.Error("expected non-empty string")
	}

	if !contains(str, "test-cluster") {
		t.Errorf("expected string to contain cluster name, got %s", str)
	}

	if !contains(str, "test-context") {
		t.Errorf("expected string to contain context name, got %s", str)
	}

	if !contains(str, "true") {
		t.Errorf("expected string to contain health status, got %s", str)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
