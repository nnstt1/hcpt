package variable

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

type variableSetService interface {
	client.WorkspaceService
	client.VariableService
}

type variableSetClientFactory func() (variableSetService, error)

func defaultVariableSetClientFactory() (variableSetService, error) {
	return client.NewClientWrapper()
}

func newCmdVariableSet() *cobra.Command {
	return newCmdVariableSetWith(defaultVariableSetClientFactory)
}

func newCmdVariableSetWith(clientFn variableSetClientFactory) *cobra.Command {
	var (
		workspaceName string
		category      string
		sensitive     bool
		hcl           bool
		description   string
	)

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Create or update a workspace variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}
			if workspaceName == "" {
				return fmt.Errorf("workspace is required: use --workspace (-w) flag")
			}

			cat := tfe.CategoryTerraform
			if category == "env" {
				cat = tfe.CategoryEnv
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runVariableSet(svc, org, workspaceName, args[0], args[1], cat, sensitive, hcl, description)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "workspace name (required)")
	cmd.Flags().StringVar(&category, "category", "terraform", "variable category (terraform or env)")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark variable as sensitive")
	cmd.Flags().BoolVar(&hcl, "hcl", false, "mark variable value as HCL")
	cmd.Flags().StringVar(&description, "description", "", "variable description")

	return cmd
}

func runVariableSet(svc variableSetService, org, workspaceName, key, value string, category tfe.CategoryType, sensitive, hcl bool, description string) error {
	ctx := context.Background()

	ws, err := svc.ReadWorkspace(ctx, org, workspaceName)
	if err != nil {
		return fmt.Errorf("failed to read workspace %q: %w", workspaceName, err)
	}

	// Search for existing variable with same key and category
	varList, err := svc.ListVariables(ctx, ws.ID, &tfe.VariableListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list variables: %w", err)
	}

	for _, v := range varList.Items {
		if v.Key == key && v.Category == category {
			// Update existing variable
			opts := tfe.VariableUpdateOptions{
				Key:         &key,
				Value:       &value,
				Sensitive:   &sensitive,
				HCL:         &hcl,
				Description: &description,
			}
			_, err := svc.UpdateVariable(ctx, ws.ID, v.ID, opts)
			if err != nil {
				return fmt.Errorf("failed to update variable %q: %w", key, err)
			}
			fmt.Printf("Updated variable %q\n", key)
			return nil
		}
	}

	// Create new variable
	opts := tfe.VariableCreateOptions{
		Key:         &key,
		Value:       &value,
		Category:    &category,
		Sensitive:   &sensitive,
		HCL:         &hcl,
		Description: &description,
	}
	_, err = svc.CreateVariable(ctx, ws.ID, opts)
	if err != nil {
		return fmt.Errorf("failed to create variable %q: %w", key, err)
	}
	fmt.Printf("Created variable %q\n", key)
	return nil
}
