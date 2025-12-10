package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewClient creates a new cluster client from a REST config
// This performs the actual connection to the Kubernetes API server
func NewClient(ctx context.Context, name string, contextName string, restConfig *rest.Config, logger *slog.Logger) (*Client, error) {
	if restConfig == nil {
		return nil, fmt.Errorf("rest config cannot be nil")
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	client := &Client{
		Name:       name,
		Context:    contextName,
		Clientset:  clientset,
		RestConfig: restConfig,
		Healthy:    false, // Will be set by health check
	}

	logger.Debug("created cluster client",
		"cluster", name,
		"context", contextName,
		"server", restConfig.Host)

	return client, nil
}

// HealthCheck performs a health check on the cluster by pinging the API server
// It uses the Discovery API to get the server version, which is a lightweight operation
func (c *Client) HealthCheck(ctx context.Context) error {
	// Create a context with timeout to prevent hanging on unresponsive clusters
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Channel to receive the result
	type result struct {
		version string
		err     error
	}
	resultCh := make(chan result, 1)

	// Run health check in goroutine to respect context
	go func() {
		version, err := c.Clientset.Discovery().ServerVersion()
		if err != nil {
			resultCh <- result{err: err}
			return
		}
		resultCh <- result{version: version.String(), err: nil}
	}()

	// Wait for either result or context cancellation
	select {
	case <-healthCtx.Done():
		c.Healthy = false
		return fmt.Errorf("health check timeout: %w", healthCtx.Err())
	case res := <-resultCh:
		if res.err != nil {
			c.Healthy = false
			return fmt.Errorf("failed to get server version: %w", res.err)
		}
		c.Healthy = true
		return nil
	}
}

// GetServerVersion returns the Kubernetes server version
func (c *Client) GetServerVersion(ctx context.Context) (string, error) {
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type result struct {
		version string
		err     error
	}
	resultCh := make(chan result, 1)

	go func() {
		version, err := c.Clientset.Discovery().ServerVersion()
		if err != nil {
			resultCh <- result{err: err}
			return
		}
		resultCh <- result{version: version.String(), err: nil}
	}()

	select {
	case <-versionCtx.Done():
		return "", fmt.Errorf("get server version timeout: %w", versionCtx.Err())
	case res := <-resultCh:
		return res.version, res.err
	}
}

// IsHealthy returns the current health status
func (c *Client) IsHealthy() bool {
	return c.Healthy
}

// String returns a string representation of the client
func (c *Client) String() string {
	return fmt.Sprintf("Client{Name: %s, Context: %s, Healthy: %v}", c.Name, c.Context, c.Healthy)
}
