package workspace

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type wsListClientFactory func() (client.ExplorerService, error)

func defaultWSListClientFactory() (client.ExplorerService, error) {
	return client.NewClientWrapper()
}

func newCmdWorkspaceList() *cobra.Command {
	return newCmdWorkspaceListWith(defaultWSListClientFactory)
}

func newCmdWorkspaceListWith(clientFn wsListClientFactory) *cobra.Command {
	var search string

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List workspaces in an organization",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runWorkspaceList(svc, org, search)
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "search workspaces by name")

	return cmd
}

func runWorkspaceList(svc client.ExplorerService, org, search string) error {
	ctx := context.Background()

	var allItems []client.ExplorerWorkspace
	page := 1
	for {
		result, err := svc.ListExplorerWorkspaces(ctx, org, client.ExplorerListOptions{
			Search: search,
			Page:   page,
		})
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}
		allItems = append(allItems, result.Items...)
		if page >= result.TotalPages {
			break
		}
		page = result.NextPage
	}

	if viper.GetBool("json") {
		items := make([]workspaceListJSON, 0, len(allItems))
		for _, ws := range allItems {
			items = append(items, toWorkspaceListJSON(ws))
		}
		return output.PrintJSON(os.Stdout, items)
	}

	headers := []string{"NAME", "ID", "PROJECT", "TERRAFORM VERSION", "CURRENT RUN", "UPDATED AT"}
	rows := make([][]string, 0, len(allItems))
	for _, ws := range allItems {
		rows = append(rows, []string{
			ws.WorkspaceName,
			ws.WorkspaceID,
			ws.ProjectName,
			ws.TerraformVersion,
			ws.CurrentRunStatus,
			ws.UpdatedAt,
		})
	}

	output.Print(os.Stdout, headers, rows)
	return nil
}
