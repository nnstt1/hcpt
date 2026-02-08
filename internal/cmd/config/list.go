package config

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/output"
)

type configJSON struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func newCmdConfigList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList()
		},
	}
	return cmd
}

func runConfigList() error {
	keys := make([]string, 0, len(ValidKeys))
	for k := range ValidKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if viper.GetBool("json") {
		items := make([]configJSON, 0, len(keys))
		for _, key := range keys {
			value := viper.GetString(key)
			if key == "token" && value != "" {
				value = maskToken(value)
			}
			items = append(items, configJSON{
				Key:   key,
				Value: value,
			})
		}
		return output.PrintJSON(os.Stdout, items)
	}

	pairs := make([]output.KeyValue, 0, len(keys))
	for _, key := range keys {
		value := viper.GetString(key)
		if key == "token" && value != "" {
			value = maskToken(value)
		}
		pairs = append(pairs, output.KeyValue{
			Key:   key,
			Value: value,
		})
	}

	if len(pairs) == 0 {
		fmt.Println("No configuration values set.")
		return nil
	}

	output.PrintKeyValue(os.Stdout, pairs)
	return nil
}
