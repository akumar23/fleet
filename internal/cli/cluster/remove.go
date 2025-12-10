package cluster

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newRemoveCmd creates the cluster remove command
// This is a placeholder for Phase 2 - full implementation in later phases
func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove NAME",
		Short: "Remove a cluster from fleet configuration",
		Long: `Remove a cluster from fleet configuration.

This command removes cluster-specific configuration from the fleet config file.
It does not modify your kubeconfig.`,
		Aliases: []string{"rm", "delete"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet - coming in later phases")
		},
	}

	return cmd
}
