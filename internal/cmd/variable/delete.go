package variable

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

type variableDeleteService interface {
	client.WorkspaceService
	client.VariableService
}

type variableDeleteClientFactory func() (variableDeleteService, error)

func defaultVariableDeleteClientFactory() (variableDeleteService, error) {
	return client.NewClientWrapper()
}

func newCmdVariableDelete() *cobra.Command {
	return newCmdVariableDeleteWith(defaultVariableDeleteClientFactory)
}

func newCmdVariableDeleteWith(clientFn variableDeleteClientFactory) *cobra.Command {
	var (
		workspaceName string
		category      string
	)

	cmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a workspace variable",
		Args:  cobra.ExactArgs(1),
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
			return runVariableDelete(svc, org, workspaceName, args[0], cat)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "workspace name (required)")
	cmd.Flags().StringVar(&category, "category", "terraform", "variable category (terraform or env)")

	return cmd
}

func runVariableDelete(svc variableDeleteService, org, workspaceName, key string, category tfe.CategoryType) error {
	ctx := context.Background()

	ws, err := svc.ReadWorkspace(ctx, org, workspaceName)
	if err != nil {
		return fmt.Errorf("failed to read workspace %q: %w", workspaceName, err)
	}

	varList, err := svc.ListVariables(ctx, ws.ID, &tfe.VariableListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list variables: %w", err)
	}

	for _, v := range varList.Items {
		if v.Key == key && v.Category == category {
			if err := svc.DeleteVariable(ctx, ws.ID, v.ID); err != nil {
				return fmt.Errorf("failed to delete variable %q: %w", key, err)
			}
			fmt.Printf("Deleted variable %q\n", key)
			return nil
		}
	}

	return fmt.Errorf("variable %q not found in workspace %q", key, workspaceName)
}
