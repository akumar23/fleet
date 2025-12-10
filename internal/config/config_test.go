package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_Load(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		wantErr        bool
		wantClusters   int
		wantTimeout    time.Duration
		wantParallel   int
	}{
		{
			name: "valid config with clusters",
			configContent: `
defaultContext: test-context
clusters:
  test-cluster-1:
    context: test-context-1
    alias: prod
    enabled: true
    labels:
      env: production
      region: us-east
  test-cluster-2:
    context: test-context-2
    alias: staging
    enabled: true
defaults:
  timeout: 60s
  parallel: 10
  outputFormat: json
`,
			wantErr:      false,
			wantClusters: 2,
			wantTimeout:  60 * time.Second,
			wantParallel: 10,
		},
		{
			name: "minimal config with defaults",
			configContent: `
clusters:
  test-cluster:
    context: test-context
    enabled: true
`,
			wantErr:      false,
			wantClusters: 1,
			wantTimeout:  30 * time.Second, // default
			wantParallel: 5,                // default
		},
		{
			name:          "empty config",
			configContent: "",
			wantErr:       false,
			wantClusters:  0,
			wantTimeout:   30 * time.Second,
			wantParallel:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".fleet.yaml")

			if tt.configContent != "" {
				if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
					t.Fatalf("failed to write test config: %v", err)
				}
			}

			manager := NewManager(configPath)
			config, err := manager.Load()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			// For empty config, we don't write a file, so Load() will create empty config
			// The error is acceptable if file doesn't exist
			if err != nil && tt.configContent != "" {
				t.Fatalf("unexpected error: %v", err)
			}

			// GetConfig should always return a valid config object
			config = manager.GetConfig()
			if config == nil {
				t.Fatal("config is nil")
			}

			if len(config.Clusters) != tt.wantClusters {
				t.Errorf("got %d clusters, want %d", len(config.Clusters), tt.wantClusters)
			}

			if config.Defaults.Timeout != tt.wantTimeout {
				t.Errorf("got timeout %v, want %v", config.Defaults.Timeout, tt.wantTimeout)
			}

			if config.Defaults.Parallel != tt.wantParallel {
				t.Errorf("got parallel %d, want %d", config.Defaults.Parallel, tt.wantParallel)
			}
		})
	}
}

func TestManager_GetClusterConfig(t *testing.T) {
	configContent := `
clusters:
  test-cluster:
    context: test-context
    alias: test
    enabled: true
    labels:
      env: test
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".fleet.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	manager := NewManager(configPath)
	if _, err := manager.Load(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tests := []struct {
		name        string
		clusterName string
		wantFound   bool
		wantAlias   string
	}{
		{
			name:        "existing cluster",
			clusterName: "test-cluster",
			wantFound:   true,
			wantAlias:   "test",
		},
		{
			name:        "non-existent cluster",
			clusterName: "non-existent",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster, found := manager.GetClusterConfig(tt.clusterName)

			if found != tt.wantFound {
				t.Errorf("got found=%v, want %v", found, tt.wantFound)
			}

			if tt.wantFound {
				if cluster.Alias != tt.wantAlias {
					t.Errorf("got alias %q, want %q", cluster.Alias, tt.wantAlias)
				}
			}
		})
	}
}

func TestManager_SetClusterConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".fleet.yaml")

	manager := NewManager(configPath)
	if _, err := manager.Load(); err != nil {
		// Empty config is fine
	}

	// Set a new cluster config
	newCluster := ClusterConfig{
		Context: "new-context",
		Alias:   "new-alias",
		Enabled: true,
		Labels: map[string]string{
			"env": "dev",
		},
	}

	manager.SetClusterConfig("new-cluster", newCluster)

	// Verify it was set
	cluster, found := manager.GetClusterConfig("new-cluster")
	if !found {
		t.Fatal("cluster not found after setting")
	}

	if cluster.Alias != "new-alias" {
		t.Errorf("got alias %q, want %q", cluster.Alias, "new-alias")
	}

	if cluster.Context != "new-context" {
		t.Errorf("got context %q, want %q", cluster.Context, "new-context")
	}
}

func TestManager_RemoveClusterConfig(t *testing.T) {
	configContent := `
clusters:
  test-cluster:
    context: test-context
    enabled: true
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".fleet.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	manager := NewManager(configPath)
	if _, err := manager.Load(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify cluster exists
	_, found := manager.GetClusterConfig("test-cluster")
	if !found {
		t.Fatal("test cluster not found before removal")
	}

	// Remove cluster
	manager.RemoveClusterConfig("test-cluster")

	// Verify cluster is gone
	_, found = manager.GetClusterConfig("test-cluster")
	if found {
		t.Error("cluster still exists after removal")
	}
}

func TestManager_GetEnabledClusters(t *testing.T) {
	configContent := `
clusters:
  cluster-1:
    context: context-1
    enabled: true
  cluster-2:
    context: context-2
    enabled: false
  cluster-3:
    context: context-3
    enabled: true
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".fleet.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	manager := NewManager(configPath)
	if _, err := manager.Load(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	enabled := manager.GetEnabledClusters()

	if len(enabled) != 2 {
		t.Errorf("got %d enabled clusters, want 2", len(enabled))
	}

	// Check that cluster-2 is not in the list
	for _, name := range enabled {
		if name == "cluster-2" {
			t.Error("cluster-2 should not be enabled")
		}
	}
}

func TestManager_GetClustersByLabel(t *testing.T) {
	configContent := `
clusters:
  cluster-1:
    context: context-1
    enabled: true
    labels:
      env: production
      region: us-east
  cluster-2:
    context: context-2
    enabled: true
    labels:
      env: staging
      region: us-west
  cluster-3:
    context: context-3
    enabled: true
    labels:
      env: production
      region: us-west
  cluster-4:
    context: context-4
    enabled: false
    labels:
      env: production
      region: us-east
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".fleet.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	manager := NewManager(configPath)
	if _, err := manager.Load(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tests := []struct {
		name       string
		labels     map[string]string
		wantCount  int
		wantNames  []string
	}{
		{
			name: "match by env=production",
			labels: map[string]string{
				"env": "production",
			},
			wantCount: 2, // cluster-1 and cluster-3 (cluster-4 is disabled)
		},
		{
			name: "match by env and region",
			labels: map[string]string{
				"env":    "production",
				"region": "us-east",
			},
			wantCount: 1, // only cluster-1
			wantNames: []string{"cluster-1"},
		},
		{
			name: "match all (empty labels)",
			labels: map[string]string{},
			wantCount: 3, // all enabled clusters
		},
		{
			name: "no matches",
			labels: map[string]string{
				"env": "development",
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusters := manager.GetClustersByLabel(tt.labels)

			if len(clusters) != tt.wantCount {
				t.Errorf("got %d clusters, want %d", len(clusters), tt.wantCount)
			}

			if tt.wantNames != nil {
				clusterMap := make(map[string]bool)
				for _, name := range clusters {
					clusterMap[name] = true
				}

				for _, wantName := range tt.wantNames {
					if !clusterMap[wantName] {
						t.Errorf("cluster %q not found in results", wantName)
					}
				}
			}
		})
	}
}

func TestManager_MergeClusterInfo(t *testing.T) {
	configContent := `
clusters:
  test-context-1:
    context: test-context-1
    alias: prod-cluster
    enabled: true
    labels:
      env: production
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".fleet.yaml")

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	manager := NewManager(configPath)
	if _, err := manager.Load(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Create test cluster info from kubeconfig
	clusterInfo := []ClusterInfo{
		{
			Name:      "test-cluster-1",
			Context:   "test-context-1",
			Server:    "https://test-server:6443",
			Namespace: "default",
			User:      "test-user",
			Current:   true,
		},
		{
			Name:      "test-cluster-2",
			Context:   "test-context-2",
			Server:    "https://test-server-2:6443",
			Namespace: "default",
			User:      "test-user-2",
			Current:   false,
		},
	}

	// Merge with fleet config
	merged := manager.MergeClusterInfo(clusterInfo)

	if len(merged) != 2 {
		t.Errorf("got %d clusters, want 2", len(merged))
	}

	// Check that first cluster got the alias and labels
	if merged[0].Alias != "prod-cluster" {
		t.Errorf("got alias %q, want %q", merged[0].Alias, "prod-cluster")
	}

	if merged[0].Labels == nil || merged[0].Labels["env"] != "production" {
		t.Error("labels not merged correctly")
	}

	// Second cluster should not have alias or labels
	if merged[1].Alias != "" {
		t.Errorf("unexpected alias %q for cluster without config", merged[1].Alias)
	}

	if merged[1].Labels != nil && len(merged[1].Labels) > 0 {
		t.Error("unexpected labels for cluster without config")
	}
}

func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	manager := NewManager(configPath)

	// Set some configuration
	manager.SetClusterConfig("test-cluster", ClusterConfig{
		Context: "test-context",
		Alias:   "test",
		Enabled: true,
		Labels: map[string]string{
			"env": "test",
		},
	})

	// Save the configuration
	if err := manager.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load it back and verify
	manager2 := NewManager(configPath)
	config, err := manager2.Load()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(config.Clusters) != 1 {
		t.Errorf("got %d clusters, want 1", len(config.Clusters))
	}

	cluster, found := config.Clusters["test-cluster"]
	if !found {
		t.Fatal("test-cluster not found in saved config")
	}

	if cluster.Alias != "test" {
		t.Errorf("got alias %q, want %q", cluster.Alias, "test")
	}
}
