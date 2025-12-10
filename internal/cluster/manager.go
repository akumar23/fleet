package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/aryankumar/fleet/internal/config"
)

// Manager manages connections to multiple Kubernetes clusters
// It handles concurrent connection establishment, health checking, and graceful shutdown
type Manager struct {
	// clients is a map of cluster name to client
	clients map[string]*Client

	// mu protects concurrent access to the clients map
	// Using RWMutex for read-heavy access patterns
	mu sync.RWMutex

	// loader handles kubeconfig loading and parsing
	loader *config.KubeconfigLoader

	// logger for structured logging
	logger *slog.Logger

	// closed indicates if the manager has been closed
	closed bool
}

// NewManager creates a new cluster manager
func NewManager(loader *config.KubeconfigLoader, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		clients: make(map[string]*Client),
		loader:  loader,
		logger:  logger,
		closed:  false,
	}
}

// Connect establishes connections to the specified clusters
// It performs concurrent connection establishment for efficiency
// Returns an error if any connections fail, but continues trying all clusters
func (m *Manager) Connect(ctx context.Context, clusterNames []string) error {
	if len(clusterNames) == 0 {
		return fmt.Errorf("no cluster names provided")
	}

	m.logger.Info("connecting to clusters",
		"count", len(clusterNames),
		"clusters", clusterNames)

	// Use errgroup pattern for concurrent connection with error aggregation
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		errors []error
	)

	// Semaphore to limit concurrent connections (avoid overwhelming the system)
	sem := make(chan struct{}, 10)

	for _, name := range clusterNames {
		wg.Add(1)

		go func(clusterName string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				errors = append(errors, fmt.Errorf("cluster %s: %w", clusterName, ctx.Err()))
				mu.Unlock()
				return
			}

			// Check for context cancellation before starting work
			select {
			case <-ctx.Done():
				mu.Lock()
				errors = append(errors, fmt.Errorf("cluster %s: %w", clusterName, ctx.Err()))
				mu.Unlock()
				return
			default:
			}

			m.logger.Debug("connecting to cluster", "cluster", clusterName)

			// Build REST config for this cluster
			restConfig, err := m.loader.BuildClientConfig(clusterName)
			if err != nil {
				m.logger.Error("failed to build client config",
					"cluster", clusterName,
					"error", err)
				mu.Lock()
				errors = append(errors, fmt.Errorf("cluster %s: %w", clusterName, err))
				mu.Unlock()
				return
			}

			// Create the client
			client, err := NewClient(ctx, clusterName, clusterName, restConfig, m.logger)
			if err != nil {
				m.logger.Error("failed to create client",
					"cluster", clusterName,
					"error", err)
				mu.Lock()
				errors = append(errors, fmt.Errorf("cluster %s: %w", clusterName, err))
				mu.Unlock()
				return
			}

			// Store the client (thread-safe)
			m.mu.Lock()
			if m.closed {
				m.mu.Unlock()
				m.logger.Warn("manager is closed, skipping client storage",
					"cluster", clusterName)
				return
			}
			m.clients[clusterName] = client
			m.mu.Unlock()

			m.logger.Info("successfully connected to cluster",
				"cluster", clusterName,
				"server", restConfig.Host)
		}(name)
	}

	// Wait for all connections to complete
	wg.Wait()

	// Check if we have any errors
	if len(errors) > 0 {
		m.logger.Warn("some cluster connections failed",
			"total", len(clusterNames),
			"failed", len(errors),
			"succeeded", len(clusterNames)-len(errors))

		// Return aggregated error
		return fmt.Errorf("failed to connect to %d/%d clusters: %v", len(errors), len(clusterNames), errors)
	}

	m.logger.Info("successfully connected to all clusters", "count", len(clusterNames))
	return nil
}

// ConnectAll establishes connections to all available clusters in the kubeconfig
func (m *Manager) ConnectAll(ctx context.Context) error {
	m.logger.Debug("discovering all contexts from kubeconfig")

	contexts, err := m.loader.GetContexts()
	if err != nil {
		return fmt.Errorf("failed to get contexts: %w", err)
	}

	if len(contexts) == 0 {
		return fmt.Errorf("no contexts found in kubeconfig")
	}

	m.logger.Info("discovered contexts", "count", len(contexts))
	return m.Connect(ctx, contexts)
}

// GetClient returns the client for a specific cluster
// Returns an error if the cluster is not connected
func (m *Manager) GetClient(name string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, fmt.Errorf("manager is closed")
	}

	client, ok := m.clients[name]
	if !ok {
		return nil, fmt.Errorf("cluster %q not connected", name)
	}

	return client, nil
}

// GetAllClients returns all connected clients
// Returns a copy of the client slice to prevent external modification
func (m *Manager) GetAllClients() []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}

	return clients
}

// GetClientNames returns all connected cluster names
func (m *Manager) GetClientNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}

	return names
}

// HasClient returns true if the cluster is connected
func (m *Manager) HasClient(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.clients[name]
	return ok
}

// Count returns the number of connected clusters
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.clients)
}

// HealthCheck performs health checks on all connected clusters concurrently
// Returns a map of cluster name to health status
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	m.logger.Debug("starting health checks")

	m.mu.RLock()
	// Create a snapshot of clients to avoid holding the lock during checks
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.mu.RUnlock()

	if len(clients) == 0 {
		m.logger.Warn("no clients to health check")
		return make(map[string]error)
	}

	// Concurrent health checks
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make(map[string]error)
	)

	for _, client := range clients {
		wg.Add(1)

		go func(c *Client) {
			defer wg.Done()

			// Check for context cancellation
			select {
			case <-ctx.Done():
				mu.Lock()
				results[c.Name] = ctx.Err()
				mu.Unlock()
				return
			default:
			}

			m.logger.Debug("health checking cluster", "cluster", c.Name)

			err := c.HealthCheck(ctx)

			mu.Lock()
			results[c.Name] = err
			mu.Unlock()

			if err != nil {
				m.logger.Warn("health check failed",
					"cluster", c.Name,
					"error", err)
			} else {
				m.logger.Debug("health check passed", "cluster", c.Name)
			}
		}(client)
	}

	wg.Wait()

	m.logger.Info("health checks completed",
		"total", len(results),
		"healthy", countHealthy(results))

	return results
}

// HealthCheckWithStatus performs health checks and returns detailed status
func (m *Manager) HealthCheckWithStatus(ctx context.Context) []HealthStatus {
	m.logger.Debug("starting detailed health checks")

	m.mu.RLock()
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.mu.RUnlock()

	if len(clients) == 0 {
		return []HealthStatus{}
	}

	// Concurrent health checks with detailed status
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = make([]HealthStatus, 0, len(clients))
	)

	for _, client := range clients {
		wg.Add(1)

		go func(c *Client) {
			defer wg.Done()

			status := HealthStatus{
				ClusterName: c.Name,
			}

			// Check for context cancellation
			select {
			case <-ctx.Done():
				status.Healthy = false
				status.Error = ctx.Err()
				mu.Lock()
				results = append(results, status)
				mu.Unlock()
				return
			default:
			}

			// Perform health check
			err := c.HealthCheck(ctx)
			status.Error = err
			status.Healthy = err == nil

			// Get server version if healthy
			if err == nil {
				if version, vErr := c.GetServerVersion(ctx); vErr == nil {
					status.ServerVersion = version
				}
			}

			mu.Lock()
			results = append(results, status)
			mu.Unlock()
		}(client)
	}

	wg.Wait()

	return results
}

// Close gracefully closes all cluster connections
// This clears the client map and marks the manager as closed
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		m.logger.Debug("manager already closed")
		return
	}

	m.logger.Info("closing cluster manager", "clients", len(m.clients))

	// Clear all clients
	// Note: kubernetes.Clientset doesn't have an explicit Close method
	// The underlying HTTP client will be garbage collected
	m.clients = make(map[string]*Client)
	m.closed = true

	m.logger.Debug("cluster manager closed")
}

// IsClosed returns true if the manager has been closed
func (m *Manager) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// countHealthy counts the number of healthy clusters in the results
func countHealthy(results map[string]error) int {
	count := 0
	for _, err := range results {
		if err == nil {
			count++
		}
	}
	return count
}
