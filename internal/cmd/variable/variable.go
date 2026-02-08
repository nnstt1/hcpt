package variable

import (
	"github.com/spf13/cobra"
)

// NewCmdVariable returns the variable parent command.
func NewCmdVariable() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "variable",
		Aliases: []string{"var"},
		Short:   "Manage HCP Terraform workspace variables",
	}

	cmd.AddCommand(newCmdVariableList())
	cmd.AddCommand(newCmdVariableSet())
	cmd.AddCommand(newCmdVariableDelete())

	return cmd
}
