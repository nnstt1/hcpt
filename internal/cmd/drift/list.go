package drift

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type driftListService interface {
	client.ExplorerService
}

type driftListClientFactory func() (driftListService, error)

func defaultDriftListClientFactory() (driftListService, error) {
	return client.NewClientWrapper()
}

func newCmdDriftList() *cobra.Command {
	return newCmdDriftListWith(defaultDriftListClientFactory)
}

func newCmdDriftListWith(clientFn driftListClientFactory) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List workspaces with drift status",
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
			return runDriftList(svc, org, all)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show all workspaces (default: drifted only)")

	return cmd
}

type driftJSON struct {
	Workspace          string `json:"workspace"`
	Drifted            bool   `json:"drifted"`
	ResourcesDrifted   int    `json:"resources_drifted"`
	ResourcesUndrifted int    `json:"resources_undrifted"`
}

func runDriftList(svc driftListService, org string, all bool) error {
	ctx := context.Background()
	driftedOnly := !all

	var allItems []client.ExplorerWorkspace
	page := 1
	for {
		result, err := svc.ListExplorerWorkspaces(ctx, org, client.ExplorerListOptions{
			DriftedOnly: driftedOnly,
			Page:        page,
		})
		if err != nil {
			return fmt.Errorf("failed to query explorer: %w", err)
		}
		allItems = append(allItems, result.Items...)
		if page >= result.TotalPages {
			break
		}
		page = result.NextPage
	}

	if viper.GetBool("json") {
		items := make([]driftJSON, 0, len(allItems))
		for _, w := range allItems {
			items = append(items, driftJSON{
				Workspace:          w.WorkspaceName,
				Drifted:            w.Drifted,
				ResourcesDrifted:   w.ResourcesDrifted,
				ResourcesUndrifted: w.ResourcesUndrifted,
			})
		}
		return output.PrintJSON(os.Stdout, items)
	}

	headers := []string{"WORKSPACE", "DRIFTED", "RESOURCES DRIFTED"}
	rows := make([][]string, 0, len(allItems))
	for _, w := range allItems {
		rows = append(rows, []string{
			w.WorkspaceName,
			strconv.FormatBool(w.Drifted),
			strconv.Itoa(w.ResourcesDrifted),
		})
	}

	output.Print(os.Stdout, headers, rows)
	return nil
}
