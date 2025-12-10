package get

import (
	"github.com/spf13/cobra"
)

// NewGetCmd creates the get parent command
// This command aggregates all get subcommands (pods, nodes, deployments, services, namespaces)
func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get resources across multiple clusters",
		Long: `Get Kubernetes resources across all connected clusters.

Supports querying pods, nodes, deployments, services, and namespaces
with filtering options and formatted output.`,
		Example: `  # Get all pods across all clusters
  fleet get pods

  # Get pods in a specific namespace
  fleet get pods -n kube-system

  # Get pods with label selector
  fleet get pods -l app=nginx

  # Get deployments in JSON format
  fleet get deployments -o json

  # Get nodes from specific clusters
  fleet get nodes --clusters prod-east,prod-west`,
	}

	// Register all subcommands
	cmd.AddCommand(newGetPodsCmd())
	cmd.AddCommand(newGetNodesCmd())
	cmd.AddCommand(newGetDeploymentsCmd())
	cmd.AddCommand(newGetServicesCmd())
	cmd.AddCommand(newGetNamespacesCmd())

	return cmd
}
