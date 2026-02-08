package config

import (
	"github.com/spf13/cobra"
)

// ValidKeys defines the set of valid configuration keys.
var ValidKeys = map[string]bool{
	"org":     true,
	"token":   true,
	"address": true,
}

// NewCmdConfig returns the config parent command.
func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage hcpt configuration",
	}

	cmd.AddCommand(newCmdConfigSet())
	cmd.AddCommand(newCmdConfigGet())
	cmd.AddCommand(newCmdConfigList())

	return cmd
}

// maskToken masks a token value, showing only the last 4 characters.
func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return "****..." + token[len(token)-4:]
}
