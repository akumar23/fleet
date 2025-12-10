package cluster

import (
	"github.com/spf13/cobra"
)

// NewClusterCmd creates the cluster management command
func NewClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage Kubernetes clusters",
		Long: `Manage Kubernetes clusters in your kubeconfig.

This command provides subcommands for listing, adding, removing,
and switching between Kubernetes cluster contexts.`,
	}

	// Add subcommands
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newSwitchCmd())

	return cmd
}
