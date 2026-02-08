package config

import (
	"github.com/spf13/cobra"
)

// NewCmdConfig returns the config parent command.
func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage hcpt configuration",
	}

	cmd.AddCommand(newCmdConfigSet())

	return cmd
}
