package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// KubeconfigLoader handles loading and merging kubeconfig files
type KubeconfigLoader struct {
	paths         []string
	explicitPath  string
	loadedConfig  *api.Config
	configLoaded  bool
}

// NewKubeconfigLoader creates a new kubeconfig loader
// It checks sources in the following order:
// 1. Explicit path (--kubeconfig flag)
// 2. KUBECONFIG environment variable (supports multiple paths separated by ':' on Unix or ';' on Windows)
// 3. Default ~/.kube/config
func NewKubeconfigLoader(explicitPath string) *KubeconfigLoader {
	loader := &KubeconfigLoader{
		explicitPath: explicitPath,
		paths:        make([]string, 0),
	}

	// Priority 1: Explicit path from flag
	if explicitPath != "" {
		if expandedPath, err := expandPath(explicitPath); err == nil {
			loader.paths = append(loader.paths, expandedPath)
		}
		return loader
	}

	// Priority 2: KUBECONFIG environment variable (can have multiple paths)
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		separator := ":"
		if os.PathSeparator == '\\' { // Windows
			separator = ";"
		}

		for _, path := range strings.Split(kubeconfigEnv, separator) {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			if expandedPath, err := expandPath(path); err == nil {
				loader.paths = append(loader.paths, expandedPath)
			}
		}
	}

	// Priority 3: Default location (~/.kube/config)
	if len(loader.paths) == 0 {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(home, ".kube", "config")
			loader.paths = append(loader.paths, defaultPath)
		}
	}

	return loader
}

// Load returns the merged kubeconfig from all sources
func (l *KubeconfigLoader) Load() (*api.Config, error) {
	if l.configLoaded && l.loadedConfig != nil {
		return l.loadedConfig, nil
	}

	if len(l.paths) == 0 {
		return nil, fmt.Errorf("no kubeconfig paths available")
	}

	// Use clientcmd to load and merge configs
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		Precedence: l.paths,
	}

	config, err := loadingRules.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	if config == nil {
		return nil, fmt.Errorf("kubeconfig is empty")
	}

	l.loadedConfig = config
	l.configLoaded = true

	return config, nil
}

// GetContexts returns all available context names
func (l *KubeconfigLoader) GetContexts() ([]string, error) {
	config, err := l.Load()
	if err != nil {
		return nil, err
	}

	contexts := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}

	return contexts, nil
}

// GetCurrentContext returns the current context name
func (l *KubeconfigLoader) GetCurrentContext() (string, error) {
	config, err := l.Load()
	if err != nil {
		return "", err
	}

	return config.CurrentContext, nil
}

// GetClusters returns detailed information about all available clusters
func (l *KubeconfigLoader) GetClusters() ([]ClusterInfo, error) {
	config, err := l.Load()
	if err != nil {
		return nil, err
	}

	currentContext := config.CurrentContext
	clusters := make([]ClusterInfo, 0, len(config.Contexts))

	for contextName, context := range config.Contexts {
		if context == nil {
			continue
		}

		cluster := config.Clusters[context.Cluster]
		if cluster == nil {
			continue
		}

		info := ClusterInfo{
			Name:      context.Cluster,
			Context:   contextName,
			Server:    cluster.Server,
			Namespace: context.Namespace,
			User:      context.AuthInfo,
			Current:   contextName == currentContext,
		}

		// Set default namespace if not specified
		if info.Namespace == "" {
			info.Namespace = "default"
		}

		clusters = append(clusters, info)
	}

	return clusters, nil
}

// GetClusterInfo returns information about a specific context
func (l *KubeconfigLoader) GetClusterInfo(contextName string) (*ClusterInfo, error) {
	config, err := l.Load()
	if err != nil {
		return nil, err
	}

	context, exists := config.Contexts[contextName]
	if !exists {
		return nil, fmt.Errorf("context %q not found in kubeconfig", contextName)
	}

	cluster := config.Clusters[context.Cluster]
	if cluster == nil {
		return nil, fmt.Errorf("cluster %q not found for context %q", context.Cluster, contextName)
	}

	info := &ClusterInfo{
		Name:      context.Cluster,
		Context:   contextName,
		Server:    cluster.Server,
		Namespace: context.Namespace,
		User:      context.AuthInfo,
		Current:   contextName == config.CurrentContext,
	}

	if info.Namespace == "" {
		info.Namespace = "default"
	}

	return info, nil
}

// BuildClientConfig creates a rest.Config for a specific context
func (l *KubeconfigLoader) BuildClientConfig(contextName string) (*rest.Config, error) {
	if len(l.paths) == 0 {
		return nil, fmt.Errorf("no kubeconfig paths available")
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{
		Precedence: l.paths,
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		configOverrides.CurrentContext = contextName
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create client config for context %q: %w", contextName, err)
	}

	return restConfig, nil
}

// GetPaths returns the kubeconfig paths being used
func (l *KubeconfigLoader) GetPaths() []string {
	return l.paths
}

// expandPath expands ~ to home directory and evaluates environment variables
func expandPath(path string) (string, error) {
	// Expand environment variables
	path = os.ExpandEnv(path)

	// Expand tilde
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Clean the path
	return filepath.Clean(path), nil
}
