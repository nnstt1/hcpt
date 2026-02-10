package workspace

import (
	"context"
	"fmt"
	"os"
	"strconv"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type wsListClientFactory func() (client.WorkspaceService, error)

func defaultWSListClientFactory() (client.WorkspaceService, error) {
	return client.NewClientWrapper()
}

func newCmdWorkspaceList() *cobra.Command {
	return newCmdWorkspaceListWith(defaultWSListClientFactory)
}

func newCmdWorkspaceListWith(clientFn wsListClientFactory) *cobra.Command {
	var search string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces in an organization",
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

func runWorkspaceList(svc client.WorkspaceService, org, search string) error {
	ctx := context.Background()
	opts := &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	}
	if search != "" {
		opts.Search = search
	}

	var allItems []*tfe.Workspace
	for {
		wsList, err := svc.ListWorkspaces(ctx, org, opts)
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}
		allItems = append(allItems, wsList.Items...)
		if wsList.Pagination == nil || wsList.NextPage == 0 {
			break
		}
		opts.PageNumber = wsList.NextPage
	}

	if viper.GetBool("json") {
		items := make([]workspaceJSON, 0, len(allItems))
		for _, ws := range allItems {
			items = append(items, toWorkspaceJSON(ws))
		}
		return output.PrintJSON(os.Stdout, items)
	}

	headers := []string{"NAME", "ID", "EXECUTION MODE", "TERRAFORM VERSION", "LOCKED", "AUTO APPLY", "UPDATED AT"}
	rows := make([][]string, 0, len(allItems))
	for _, ws := range allItems {
		rows = append(rows, []string{
			ws.Name,
			ws.ID,
			ws.ExecutionMode,
			ws.TerraformVersion,
			strconv.FormatBool(ws.Locked),
			strconv.FormatBool(ws.AutoApply),
			ws.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	output.Print(os.Stdout, headers, rows)
	return nil
}
