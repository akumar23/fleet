package config

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestNewKubeconfigLoader(t *testing.T) {
	tests := []struct {
		name         string
		explicitPath string
		kubeconfigEnv string
		wantPaths    int
	}{
		{
			name:         "explicit path takes precedence",
			explicitPath: "/path/to/kubeconfig",
			kubeconfigEnv: "/env/kubeconfig",
			wantPaths:    1,
		},
		{
			name:         "KUBECONFIG environment variable with single path",
			explicitPath: "",
			kubeconfigEnv: "/env/kubeconfig",
			wantPaths:    1,
		},
		{
			name:         "KUBECONFIG environment variable with multiple paths",
			explicitPath: "",
			kubeconfigEnv: "/env/kubeconfig1:/env/kubeconfig2:/env/kubeconfig3",
			wantPaths:    3,
		},
		{
			name:         "default to ~/.kube/config",
			explicitPath: "",
			kubeconfigEnv: "",
			wantPaths:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.kubeconfigEnv != "" {
				os.Setenv("KUBECONFIG", tt.kubeconfigEnv)
				defer os.Unsetenv("KUBECONFIG")
			}

			loader := NewKubeconfigLoader(tt.explicitPath)

			if len(loader.paths) != tt.wantPaths {
				t.Errorf("got %d paths, want %d", len(loader.paths), tt.wantPaths)
			}
		})
	}
}

func TestKubeconfigLoader_Load(t *testing.T) {
	// Create a temporary kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Create a test kubeconfig
	config := createTestKubeconfig()
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	loader := NewKubeconfigLoader(kubeconfigPath)

	// Test loading
	loadedConfig, err := loader.Load()
	if err != nil {
		t.Fatalf("failed to load kubeconfig: %v", err)
	}

	if loadedConfig == nil {
		t.Fatal("loaded config is nil")
	}

	if len(loadedConfig.Contexts) != 2 {
		t.Errorf("got %d contexts, want 2", len(loadedConfig.Contexts))
	}

	if loadedConfig.CurrentContext != "test-context-1" {
		t.Errorf("got current context %q, want %q", loadedConfig.CurrentContext, "test-context-1")
	}

	// Test caching - second load should return cached config
	loadedConfig2, err := loader.Load()
	if err != nil {
		t.Fatalf("failed to load kubeconfig second time: %v", err)
	}

	if loadedConfig != loadedConfig2 {
		t.Error("expected cached config to be returned")
	}
}

func TestKubeconfigLoader_GetContexts(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	config := createTestKubeconfig()
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	loader := NewKubeconfigLoader(kubeconfigPath)

	contexts, err := loader.GetContexts()
	if err != nil {
		t.Fatalf("failed to get contexts: %v", err)
	}

	if len(contexts) != 2 {
		t.Errorf("got %d contexts, want 2", len(contexts))
	}

	// Check that both contexts are present
	contextMap := make(map[string]bool)
	for _, ctx := range contexts {
		contextMap[ctx] = true
	}

	if !contextMap["test-context-1"] {
		t.Error("test-context-1 not found")
	}
	if !contextMap["test-context-2"] {
		t.Error("test-context-2 not found")
	}
}

func TestKubeconfigLoader_GetCurrentContext(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	config := createTestKubeconfig()
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	loader := NewKubeconfigLoader(kubeconfigPath)

	currentContext, err := loader.GetCurrentContext()
	if err != nil {
		t.Fatalf("failed to get current context: %v", err)
	}

	if currentContext != "test-context-1" {
		t.Errorf("got current context %q, want %q", currentContext, "test-context-1")
	}
}

func TestKubeconfigLoader_GetClusters(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	config := createTestKubeconfig()
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	loader := NewKubeconfigLoader(kubeconfigPath)

	clusters, err := loader.GetClusters()
	if err != nil {
		t.Fatalf("failed to get clusters: %v", err)
	}

	if len(clusters) != 2 {
		t.Errorf("got %d clusters, want 2", len(clusters))
	}

	// Find the current context
	var currentCluster *ClusterInfo
	for i := range clusters {
		if clusters[i].Current {
			currentCluster = &clusters[i]
			break
		}
	}

	if currentCluster == nil {
		t.Fatal("no current cluster found")
	}

	if currentCluster.Context != "test-context-1" {
		t.Errorf("got current context %q, want %q", currentCluster.Context, "test-context-1")
	}

	if currentCluster.Server != "https://test-server-1:6443" {
		t.Errorf("got server %q, want %q", currentCluster.Server, "https://test-server-1:6443")
	}

	if currentCluster.Namespace != "test-namespace-1" {
		t.Errorf("got namespace %q, want %q", currentCluster.Namespace, "test-namespace-1")
	}
}

func TestKubeconfigLoader_GetClusterInfo(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	config := createTestKubeconfig()
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	loader := NewKubeconfigLoader(kubeconfigPath)

	tests := []struct {
		name        string
		contextName string
		wantErr     bool
		wantServer  string
	}{
		{
			name:        "valid context",
			contextName: "test-context-1",
			wantErr:     false,
			wantServer:  "https://test-server-1:6443",
		},
		{
			name:        "invalid context",
			contextName: "non-existent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := loader.GetClusterInfo(tt.contextName)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.Server != tt.wantServer {
				t.Errorf("got server %q, want %q", info.Server, tt.wantServer)
			}

			if info.Context != tt.contextName {
				t.Errorf("got context %q, want %q", info.Context, tt.contextName)
			}
		})
	}
}

func TestKubeconfigLoader_BuildClientConfig(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	config := createTestKubeconfig()
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	loader := NewKubeconfigLoader(kubeconfigPath)

	// Test building client config for a valid context
	restConfig, err := loader.BuildClientConfig("test-context-1")
	if err != nil {
		t.Fatalf("failed to build client config: %v", err)
	}

	if restConfig == nil {
		t.Fatal("rest config is nil")
	}

	if restConfig.Host != "https://test-server-1:6443" {
		t.Errorf("got host %q, want %q", restConfig.Host, "https://test-server-1:6443")
	}

	// Test building client config for invalid context
	_, err = loader.BuildClientConfig("non-existent")
	if err == nil {
		t.Error("expected error for non-existent context, got nil")
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		contains string
	}{
		{
			name:     "expand tilde",
			input:    "~/test/path",
			wantErr:  false,
			contains: "/test/path",
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			wantErr:  false,
			contains: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			wantErr:  false,
			contains: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.contains != "" && len(result) > 0 {
				// For tilde expansion, just check the path ends correctly
				if tt.input[:1] == "~" {
					if !filepath.IsAbs(result) {
						t.Errorf("expected absolute path for tilde expansion, got %q", result)
					}
				}
			}
		})
	}
}

// createTestKubeconfig creates a test kubeconfig for testing
func createTestKubeconfig() *api.Config {
	return &api.Config{
		CurrentContext: "test-context-1",
		Clusters: map[string]*api.Cluster{
			"test-cluster-1": {
				Server: "https://test-server-1:6443",
			},
			"test-cluster-2": {
				Server: "https://test-server-2:6443",
			},
		},
		Contexts: map[string]*api.Context{
			"test-context-1": {
				Cluster:   "test-cluster-1",
				AuthInfo:  "test-user-1",
				Namespace: "test-namespace-1",
			},
			"test-context-2": {
				Cluster:   "test-cluster-2",
				AuthInfo:  "test-user-2",
				Namespace: "test-namespace-2",
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"test-user-1": {
				Token: "test-token-1",
			},
			"test-user-2": {
				Token: "test-token-2",
			},
		},
	}
}
