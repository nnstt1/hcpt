package org

import (
	"github.com/spf13/cobra"
)

// NewCmdOrg returns the org parent command.
func NewCmdOrg() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage HCP Terraform organizations",
	}

	cmd.AddCommand(newCmdOrgList())
	cmd.AddCommand(newCmdOrgShow())

	return cmd
}
