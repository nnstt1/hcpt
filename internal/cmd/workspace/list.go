package workspace

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	var runStatus string

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List workspaces in an organization",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return errOrgRequired
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runWorkspaceList(svc, org, search, runStatus)
		},
	}

	cmd.Flags().StringVar(&search, "search", "", "search workspaces by name")
	cmd.Flags().StringVar(&runStatus, "run-status", "", "filter by current run status (comma-separated, e.g. applied,errored)")

	return cmd
}

func runWorkspaceList(svc client.ExplorerService, org, search, runStatus string) error {
	ctx := context.Background()

	// For a single status, delegate filtering to the server; for multiple, fetch all and filter client-side.
	serverRunStatus := ""
	if runStatus != "" && !strings.Contains(runStatus, ",") {
		serverRunStatus = runStatus
	}

	var allItems []client.ExplorerWorkspace
	page := 1
	for {
		result, err := svc.ListExplorerWorkspaces(ctx, org, client.ExplorerListOptions{
			Search:    search,
			RunStatus: serverRunStatus,
			Page:      page,
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

	// Client-side filtering for multiple statuses.
	if runStatus != "" && strings.Contains(runStatus, ",") {
		statusSet := make(map[string]bool)
		for _, s := range strings.Split(runStatus, ",") {
			statusSet[strings.TrimSpace(s)] = true
		}
		filtered := make([]client.ExplorerWorkspace, 0, len(allItems))
		for _, ws := range allItems {
			if statusSet[ws.CurrentRunStatus] {
				filtered = append(filtered, ws)
			}
		}
		allItems = filtered
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
