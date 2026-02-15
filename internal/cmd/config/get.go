package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newCmdConfigGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value.

Available keys:
  org       HCP Terraform organization name
  token     API token (masked for security)
  address   HCP Terraform API address`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(args[0])
		},
	}
	return cmd
}

func runConfigGet(key string) error {
	if !ValidKeys[key] {
		return fmt.Errorf("unknown config key %q (valid keys: org, token, address)", key)
	}

	value := viper.GetString(key)
	if key == "token" && value != "" {
		value = maskToken(value)
	}

	fmt.Println(value)
	return nil
}
