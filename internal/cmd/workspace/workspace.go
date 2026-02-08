package workspace

import (
	"github.com/spf13/cobra"
)

// NewCmdWorkspace returns the workspace parent command.
func NewCmdWorkspace() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Manage HCP Terraform workspaces",
	}

	cmd.AddCommand(newCmdWorkspaceList())
	cmd.AddCommand(newCmdWorkspaceShow())

	return cmd
}
