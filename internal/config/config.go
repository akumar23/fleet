package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultConfigName = ".fleet"
	defaultConfigDir  = ".fleet"
)

// Manager handles fleet configuration
type Manager struct {
	configPath string
	config     *FleetConfig
	viper      *viper.Viper
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
		viper:      viper.New(),
		config:     &FleetConfig{},
	}
}

// Load loads the fleet configuration from file
func (m *Manager) Load() (*FleetConfig, error) {
	// Set up config file path
	if m.configPath != "" {
		m.viper.SetConfigFile(m.configPath)
	} else {
		// Try multiple locations
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		// Check ~/.fleet/config.yaml
		m.viper.AddConfigPath(filepath.Join(home, defaultConfigDir))
		// Check ~/.fleet.yaml
		m.viper.AddConfigPath(home)
		m.viper.SetConfigName(defaultConfigName)
		m.viper.SetConfigType("yaml")
	}

	// Set environment variable support
	m.viper.SetEnvPrefix("FLEET")
	m.viper.AutomaticEnv()

	// Initialize config to ensure defaults are set even for empty configs
	m.config = &FleetConfig{}

	// Read config file
	if err := m.viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use defaults
		// Check for both ConfigFileNotFoundError and os.IsNotExist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// File doesn't exist, apply defaults and return
		m.applyDefaults()
		return m.config, nil
	}

	// Unmarshal into config struct
	if err := m.viper.Unmarshal(m.config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults
	m.applyDefaults()

	return m.config, nil
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	if m.configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(home, defaultConfigDir)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		m.configPath = filepath.Join(configDir, "config.yaml")
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config to file
	if err := m.viper.WriteConfigAs(m.configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *FleetConfig {
	return m.config
}

// GetClusterConfig returns configuration for a specific cluster
func (m *Manager) GetClusterConfig(name string) (*ClusterConfig, bool) {
	if m.config.Clusters == nil {
		return nil, false
	}

	cluster, ok := m.config.Clusters[name]
	return &cluster, ok
}

// SetClusterConfig sets or updates configuration for a cluster
func (m *Manager) SetClusterConfig(name string, config ClusterConfig) {
	if m.config.Clusters == nil {
		m.config.Clusters = make(map[string]ClusterConfig)
	}

	m.config.Clusters[name] = config
	m.viper.Set("clusters", m.config.Clusters)
}

// RemoveClusterConfig removes configuration for a cluster
func (m *Manager) RemoveClusterConfig(name string) {
	if m.config.Clusters == nil {
		return
	}

	delete(m.config.Clusters, name)
	m.viper.Set("clusters", m.config.Clusters)
}

// GetEnabledClusters returns a list of enabled cluster names
func (m *Manager) GetEnabledClusters() []string {
	if m.config.Clusters == nil {
		return nil
	}

	enabled := make([]string, 0)
	for name, cluster := range m.config.Clusters {
		if cluster.Enabled {
			enabled = append(enabled, name)
		}
	}

	return enabled
}

// GetClustersByLabel returns clusters matching the given labels
func (m *Manager) GetClustersByLabel(labels map[string]string) []string {
	if m.config.Clusters == nil {
		return nil
	}

	matching := make([]string, 0)
	for name, cluster := range m.config.Clusters {
		if !cluster.Enabled {
			continue
		}

		if matchesLabels(cluster.Labels, labels) {
			matching = append(matching, name)
		}
	}

	return matching
}

// applyDefaults sets default values for configuration
func (m *Manager) applyDefaults() {
	if m.config == nil {
		return
	}

	// Set default timeout
	if m.config.Defaults.Timeout == 0 {
		m.config.Defaults.Timeout = 30 * time.Second
	}

	// Set default parallel workers
	if m.config.Defaults.Parallel == 0 {
		m.config.Defaults.Parallel = 5
	}

	// Set default output format
	if m.config.Defaults.OutputFormat == "" {
		m.config.Defaults.OutputFormat = "table"
	}

	// Enable all clusters by default if not specified
	for name, cluster := range m.config.Clusters {
		// If enabled field is not explicitly set, default to true
		// This is a bit tricky with zero values, but we assume if someone
		// adds a cluster to config, they want it enabled
		if cluster.Alias == "" {
			cluster.Alias = name
		}
		m.config.Clusters[name] = cluster
	}
}

// matchesLabels checks if cluster labels match the required labels
func matchesLabels(clusterLabels, requiredLabels map[string]string) bool {
	if len(requiredLabels) == 0 {
		return true
	}

	for key, value := range requiredLabels {
		clusterValue, exists := clusterLabels[key]
		if !exists || clusterValue != value {
			return false
		}
	}

	return true
}

// MergeClusterInfo merges fleet config data into cluster info from kubeconfig
func (m *Manager) MergeClusterInfo(clusters []ClusterInfo) []ClusterInfo {
	if m.config.Clusters == nil {
		return clusters
	}

	for i := range clusters {
		// Try to find config by context name
		if cfg, ok := m.config.Clusters[clusters[i].Context]; ok {
			clusters[i].Alias = cfg.Alias
			clusters[i].Labels = cfg.Labels
		}
	}

	return clusters
}
