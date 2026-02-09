package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/cmd/config"
	"github.com/nnstt1/hcpt/internal/cmd/drift"
	"github.com/nnstt1/hcpt/internal/cmd/org"
	"github.com/nnstt1/hcpt/internal/cmd/project"
	"github.com/nnstt1/hcpt/internal/cmd/run"
	"github.com/nnstt1/hcpt/internal/cmd/variable"
	"github.com/nnstt1/hcpt/internal/cmd/workspace"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "hcpt",
	Short: "CLI tool for HCP Terraform",
	Long:  "A CLI tool to retrieve HCP Terraform configurations and workspace information.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.hcpt.yaml)")
	rootCmd.PersistentFlags().String("org", "", "HCP Terraform organization name")
	rootCmd.PersistentFlags().Bool("json", false, "output in JSON format")

	_ = viper.BindPFlag("org", rootCmd.PersistentFlags().Lookup("org"))
	_ = viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))

	rootCmd.AddCommand(config.NewCmdConfig())
	rootCmd.AddCommand(drift.NewCmdDrift())
	rootCmd.AddCommand(org.NewCmdOrg())
	rootCmd.AddCommand(project.NewCmdProject())
	rootCmd.AddCommand(workspace.NewCmdWorkspace())
	rootCmd.AddCommand(run.NewCmdRun())
	rootCmd.AddCommand(variable.NewCmdVariable())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".hcpt")
	}

	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	// Map environment variables
	_ = viper.BindEnv("token", "TFE_TOKEN")
	_ = viper.BindEnv("address", "TFE_ADDRESS")

	// Set defaults
	viper.SetDefault("address", "https://app.terraform.io")

	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only print error if it's not a "file not found" error
			if cfgFile != "" {
				fmt.Fprintln(os.Stderr, "Error reading config file:", err)
			}
		}
	}
}
