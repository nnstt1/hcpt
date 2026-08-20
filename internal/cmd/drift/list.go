package drift

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

type driftListService interface {
	client.ExplorerService
	client.ProjectService
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
	var project string

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List workspaces with drift status",
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
			return runDriftList(svc, org, all, project)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show all workspaces (default: drifted only)")
	cmd.Flags().StringVar(&project, "project", "", "filter results by project name")

	return cmd
}

// verifyProjectExists checks that a project with the given name exists in the organization.
func verifyProjectExists(ctx context.Context, svc client.ProjectService, org, name string) error {
	projList, err := svc.ListProjects(ctx, org, &tfe.ProjectListOptions{Name: name})
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}
	for _, p := range projList.Items {
		if p.Name == name {
			return nil
		}
	}
	return fmt.Errorf("project %q not found in organization %q", name, org)
}

type driftJSON struct {
	Workspace          string `json:"workspace"`
	Drifted            bool   `json:"drifted"`
	ResourcesDrifted   int    `json:"resources_drifted"`
	ResourcesUndrifted int    `json:"resources_undrifted"`
}

func runDriftList(svc driftListService, org string, all bool, project string) error {
	ctx := context.Background()
	driftedOnly := !all

	if project != "" {
		if err := verifyProjectExists(ctx, svc, org, project); err != nil {
			return err
		}
	}

	var allItems []client.ExplorerWorkspace
	page := 1
	for {
		result, err := svc.ListExplorerWorkspaces(ctx, org, client.ExplorerListOptions{
			DriftedOnly: driftedOnly,
			ProjectName: project,
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
