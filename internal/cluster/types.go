package cluster

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Client represents a connection to a single Kubernetes cluster
type Client struct {
	// Name is a friendly identifier for the cluster
	Name string

	// Context is the kubeconfig context name
	Context string

	// Clientset is the Kubernetes client interface
	Clientset kubernetes.Interface

	// RestConfig is the underlying REST configuration
	RestConfig *rest.Config

	// Healthy indicates if the last health check passed
	Healthy bool
}

// HealthStatus represents the health status of a cluster
type HealthStatus struct {
	// ClusterName is the name of the cluster
	ClusterName string

	// Healthy indicates if the cluster is healthy
	Healthy bool

	// Error contains any health check error
	Error error

	// ServerVersion is the Kubernetes server version (if healthy)
	ServerVersion string
}
