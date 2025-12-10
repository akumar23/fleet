package config

import "time"

// FleetConfig represents the fleet configuration file structure
type FleetConfig struct {
	// DefaultContext is the default kubeconfig context to use
	DefaultContext string `yaml:"defaultContext,omitempty" json:"defaultContext,omitempty"`

	// Clusters is a map of cluster aliases to their configurations
	Clusters map[string]ClusterConfig `yaml:"clusters,omitempty" json:"clusters,omitempty"`

	// Defaults contains default settings for operations
	Defaults DefaultsConfig `yaml:"defaults,omitempty" json:"defaults,omitempty"`
}

// ClusterConfig represents configuration for a single cluster
type ClusterConfig struct {
	// Context is the kubeconfig context name
	Context string `yaml:"context" json:"context"`

	// Alias is a friendly name for the cluster
	Alias string `yaml:"alias,omitempty" json:"alias,omitempty"`

	// Labels for organizing clusters
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`

	// Enabled indicates if this cluster should be included in operations
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// DefaultsConfig contains default configuration values
type DefaultsConfig struct {
	// Timeout for API operations
	Timeout time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// Parallel is the number of concurrent operations
	Parallel int `yaml:"parallel,omitempty" json:"parallel,omitempty"`

	// OutputFormat is the default output format (table, json, yaml)
	OutputFormat string `yaml:"outputFormat,omitempty" json:"outputFormat,omitempty"`

	// NoColor disables colored output
	NoColor bool `yaml:"noColor,omitempty" json:"noColor,omitempty"`
}

// ClusterInfo represents information about a cluster from kubeconfig
type ClusterInfo struct {
	// Name is the cluster name from kubeconfig
	Name string `json:"name"`

	// Context is the context name
	Context string `json:"context"`

	// Server is the API server URL
	Server string `json:"server"`

	// Namespace is the default namespace
	Namespace string `json:"namespace"`

	// User is the user for authentication
	User string `json:"user"`

	// Current indicates if this is the current context
	Current bool `json:"current"`

	// Alias is a friendly name from fleet config
	Alias string `json:"alias,omitempty"`

	// Labels from fleet config
	Labels map[string]string `json:"labels,omitempty"`
}
