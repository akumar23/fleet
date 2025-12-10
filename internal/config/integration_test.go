package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// TestKubeconfigLoaderIntegration tests the kubeconfig loader with real kubeconfig files
func TestKubeconfigLoaderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test kubeconfig
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	config := api.Config{
		Clusters: map[string]*api.Cluster{
			"dev-cluster": {
				Server:                "https://dev.example.com:6443",
				InsecureSkipTLSVerify: true,
			},
			"prod-cluster": {
				Server:                   "https://prod.example.com:6443",
				CertificateAuthorityData: []byte("fake-ca-data"),
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"dev-user": {
				Token: "dev-token",
			},
			"prod-user": {
				ClientCertificateData: []byte("fake-cert-data"),
				ClientKeyData:         []byte("fake-key-data"),
			},
		},
		Contexts: map[string]*api.Context{
			"dev": {
				Cluster:   "dev-cluster",
				AuthInfo:  "dev-user",
				Namespace: "development",
			},
			"prod": {
				Cluster:   "prod-cluster",
				AuthInfo:  "prod-user",
				Namespace: "production",
			},
		},
		CurrentContext: "dev",
	}

	if err := clientcmd.WriteToFile(config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	// Test loading the kubeconfig
	loader := NewKubeconfigLoader(kubeconfigPath)

	// Test GetContexts
	contexts, err := loader.GetContexts()
	if err != nil {
		t.Fatalf("failed to get contexts: %v", err)
	}

	if len(contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(contexts))
	}

	expectedContexts := map[string]bool{"dev": true, "prod": true}
	for _, ctx := range contexts {
		if !expectedContexts[ctx] {
			t.Errorf("unexpected context: %s", ctx)
		}
	}

	// Test GetCurrentContext
	currentCtx, err := loader.GetCurrentContext()
	if err != nil {
		t.Fatalf("failed to get current context: %v", err)
	}

	if currentCtx != "dev" {
		t.Errorf("expected current context 'dev', got %q", currentCtx)
	}

	// Test BuildClientConfig for each context
	for _, ctxName := range contexts {
		restConfig, err := loader.BuildClientConfig(ctxName)
		if err != nil {
			t.Errorf("failed to build config for context %s: %v", ctxName, err)
			continue
		}

		if restConfig == nil {
			t.Errorf("got nil rest config for context %s", ctxName)
			continue
		}

		// Verify the server URL
		expectedServer := config.Clusters[config.Contexts[ctxName].Cluster].Server
		if restConfig.Host != expectedServer {
			t.Errorf("context %s: expected server %s, got %s", ctxName, expectedServer, restConfig.Host)
		}
	}
}

// TestKubeconfigLoaderWithMergedConfigs tests loading from multiple kubeconfig files
func TestKubeconfigLoaderWithMergedConfigs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create first kubeconfig
	config1Path := filepath.Join(tmpDir, "config1")
	config1 := api.Config{
		Clusters: map[string]*api.Cluster{
			"cluster-1": {
				Server:                "https://cluster1.example.com:6443",
				InsecureSkipTLSVerify: true,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"user-1": {Token: "token-1"},
		},
		Contexts: map[string]*api.Context{
			"context-1": {
				Cluster:  "cluster-1",
				AuthInfo: "user-1",
			},
		},
		CurrentContext: "context-1",
	}

	if err := clientcmd.WriteToFile(config1, config1Path); err != nil {
		t.Fatalf("failed to write config1: %v", err)
	}

	// Create second kubeconfig
	config2Path := filepath.Join(tmpDir, "config2")
	config2 := api.Config{
		Clusters: map[string]*api.Cluster{
			"cluster-2": {
				Server:                "https://cluster2.example.com:6443",
				InsecureSkipTLSVerify: true,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"user-2": {Token: "token-2"},
		},
		Contexts: map[string]*api.Context{
			"context-2": {
				Cluster:  "cluster-2",
				AuthInfo: "user-2",
			},
		},
	}

	if err := clientcmd.WriteToFile(config2, config2Path); err != nil {
		t.Fatalf("failed to write config2: %v", err)
	}

	// Load with merged configs
	mergedPath := fmt.Sprintf("%s:%s", config1Path, config2Path)
	loader := NewKubeconfigLoader(mergedPath)

	contexts, err := loader.GetContexts()
	if err != nil {
		t.Fatalf("failed to get contexts: %v", err)
	}

	if len(contexts) != 2 {
		t.Errorf("expected 2 contexts from merged configs, got %d", len(contexts))
	}

	// Verify both contexts are present
	contextMap := make(map[string]bool)
	for _, ctx := range contexts {
		contextMap[ctx] = true
	}

	if !contextMap["context-1"] || !contextMap["context-2"] {
		t.Errorf("expected both context-1 and context-2, got %v", contexts)
	}
}

// TestKubeconfigLoaderErrorCases tests error handling in kubeconfig loading
func TestKubeconfigLoaderErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) string
		context       string
		expectError   bool
		errorContains string
	}{
		{
			name: "nonexistent file",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/to/kubeconfig"
			},
			context:       "any-context",
			expectError:   true,
			errorContains: "does not exist",
		},
		{
			name: "invalid context",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "config")
				config := api.Config{
					Clusters: map[string]*api.Cluster{
						"cluster-1": {Server: "https://example.com"},
					},
					AuthInfos: map[string]*api.AuthInfo{
						"user-1": {Token: "token"},
					},
					Contexts: map[string]*api.Context{
						"valid-context": {Cluster: "cluster-1", AuthInfo: "user-1"},
					},
					CurrentContext: "valid-context",
				}
				clientcmd.WriteToFile(config, path)
				return path
			},
			context:       "invalid-context",
			expectError:   true,
			errorContains: "context",
		},
		{
			name: "empty kubeconfig",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "config")
				config := api.Config{}
				clientcmd.WriteToFile(config, path)
				return path
			},
			context:     "any",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeconfigPath := tt.setup(t)
			loader := NewKubeconfigLoader(kubeconfigPath)

			_, err := loader.BuildClientConfig(tt.context)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorContains)
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestKubeconfigLoaderDefaultPath tests loading from default kubeconfig location
func TestKubeconfigLoaderDefaultPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Create temporary home directory
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	// Create .kube directory
	kubeDir := filepath.Join(tmpHome, ".kube")
	if err := os.MkdirAll(kubeDir, 0755); err != nil {
		t.Fatalf("failed to create .kube directory: %v", err)
	}

	// Create a kubeconfig in the default location
	kubeconfigPath := filepath.Join(kubeDir, "config")
	config := api.Config{
		Clusters: map[string]*api.Cluster{
			"default-cluster": {
				Server:                "https://default.example.com:6443",
				InsecureSkipTLSVerify: true,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"default-user": {Token: "default-token"},
		},
		Contexts: map[string]*api.Context{
			"default": {
				Cluster:  "default-cluster",
				AuthInfo: "default-user",
			},
		},
		CurrentContext: "default",
	}

	if err := clientcmd.WriteToFile(config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	// Create loader with empty path (should use default)
	loader := NewKubeconfigLoader("")

	contexts, err := loader.GetContexts()
	if err != nil {
		t.Fatalf("failed to get contexts from default location: %v", err)
	}

	if len(contexts) != 1 || contexts[0] != "default" {
		t.Errorf("expected context 'default', got %v", contexts)
	}
}

// TestKubeconfigLoaderConcurrentAccess tests concurrent access to kubeconfig loader
func TestKubeconfigLoaderConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create kubeconfig with multiple contexts
	config := api.Config{
		Clusters:  make(map[string]*api.Cluster),
		AuthInfos: make(map[string]*api.AuthInfo),
		Contexts:  make(map[string]*api.Context),
	}

	for i := 1; i <= 10; i++ {
		clusterName := fmt.Sprintf("cluster-%d", i)
		config.Clusters[clusterName] = &api.Cluster{
			Server:                fmt.Sprintf("https://cluster%d.example.com:6443", i),
			InsecureSkipTLSVerify: true,
		}
		config.AuthInfos[clusterName] = &api.AuthInfo{
			Token: fmt.Sprintf("token-%d", i),
		}
		config.Contexts[clusterName] = &api.Context{
			Cluster:  clusterName,
			AuthInfo: clusterName,
		}
	}
	config.CurrentContext = "cluster-1"

	if err := clientcmd.WriteToFile(config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	loader := NewKubeconfigLoader(kubeconfigPath)

	// Concurrent access
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Get contexts
			contexts, err := loader.GetContexts()
			if err != nil {
				t.Errorf("goroutine %d: failed to get contexts: %v", id, err)
				return
			}
			if len(contexts) != 10 {
				t.Errorf("goroutine %d: expected 10 contexts, got %d", id, len(contexts))
			}

			// Build client config for a random context
			contextIdx := id % 10
			contextName := fmt.Sprintf("cluster-%d", contextIdx+1)
			_, err = loader.BuildClientConfig(contextName)
			if err != nil {
				t.Errorf("goroutine %d: failed to build config for %s: %v", id, contextName, err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())
}
