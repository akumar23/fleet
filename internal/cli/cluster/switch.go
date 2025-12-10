package cluster

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newSwitchCmd creates the cluster switch command
// This is a placeholder for Phase 2 - full implementation in later phases
func newSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch CONTEXT",
		Short: "Switch to a different cluster context",
		Long: `Switch the current Kubernetes context to a different cluster.

This command updates the current-context in your kubeconfig file.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet - coming in later phases")
		},
	}

	return cmd
}
