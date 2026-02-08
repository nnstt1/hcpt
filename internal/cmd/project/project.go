package project

import (
	"github.com/spf13/cobra"
)

// NewCmdProject returns the project parent command.
func NewCmdProject() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage HCP Terraform projects",
	}

	cmd.AddCommand(newCmdProjectList())

	return cmd
}
