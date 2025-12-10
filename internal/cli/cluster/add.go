package cluster

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newAddCmd creates the cluster add command
// This is a placeholder for Phase 2 - full implementation in later phases
func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a new cluster to fleet configuration",
		Long: `Add a new cluster to fleet configuration.

This command allows you to add cluster-specific configuration such as
aliases, labels, and enable/disable status.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet - coming in later phases")
		},
	}

	return cmd
}
