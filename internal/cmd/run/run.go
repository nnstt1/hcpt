package run

import (
	"github.com/spf13/cobra"
)

// NewCmdRun returns the run parent command.
func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Manage HCP Terraform runs",
	}

	cmd.AddCommand(newCmdRunList())

	return cmd
}
