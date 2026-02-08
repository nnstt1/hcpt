package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

func newCmdConfigSet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value in ~/.hcpt.yaml",
		Long: `Set a configuration value in the hcpt config file.

Available keys:
  org       HCP Terraform organization name
  token     API token
  address   HCP Terraform API address`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(args[0], args[1])
		},
	}
	return cmd
}

func runConfigSet(key, value string) error {
	if !ValidKeys[key] {
		return fmt.Errorf("unknown config key %q (valid keys: org, token, address)", key)
	}

	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".hcpt.yaml")
	}

	existing := make(map[string]string)
	data, err := os.ReadFile(configPath)
	if err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}

	existing[key] = value

	out, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, out, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Set %q to %q in %s\n", key, value, configPath)
	return nil
}
