package drift

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

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
	var projects []string
	var excludeProjects []string

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
			return runDriftList(svc, org, all, projects, excludeProjects)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show all workspaces (default: drifted only)")
	cmd.Flags().StringArrayVar(&projects, "project", nil, "filter results by project name (can be repeated)")
	cmd.Flags().StringArrayVar(&excludeProjects, "exclude-project", nil, "exclude results from this project name (can be repeated)")

	return cmd
}

// verifyProjectsExist checks that a project with each given name exists in the organization.
// It returns an error listing every name that could not be found.
func verifyProjectsExist(ctx context.Context, svc client.ProjectService, org string, names []string) error {
	var missing []string
	for _, name := range names {
		projList, err := svc.ListProjects(ctx, org, &tfe.ProjectListOptions{Name: name})
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}
		found := false
		for _, p := range projList.Items {
			if p.Name == name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("projects not found in organization %q: %s", org, strings.Join(missing, ", "))
	}
	return nil
}

// filterByProject applies client-side include/exclude filtering by project name.
// The Explorer API's filter query parameters only honor a single value index
// per field+operator, so filtering by multiple project names must happen
// client-side rather than via server-side filter query parameters.
func filterByProject(items []client.ExplorerWorkspace, include, exclude []string) []client.ExplorerWorkspace {
	if len(include) == 0 && len(exclude) == 0 {
		return items
	}
	filtered := make([]client.ExplorerWorkspace, 0, len(items))
	for _, item := range items {
		if len(include) > 0 && !slices.Contains(include, item.ProjectName) {
			continue
		}
		if slices.Contains(exclude, item.ProjectName) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

type driftJSON struct {
	Workspace          string `json:"workspace"`
	Drifted            bool   `json:"drifted"`
	ResourcesDrifted   int    `json:"resources_drifted"`
	ResourcesUndrifted int    `json:"resources_undrifted"`
}

func runDriftList(svc driftListService, org string, all bool, projects []string, excludeProjects []string) error {
	ctx := context.Background()
	driftedOnly := !all

	for _, name := range projects {
		if slices.Contains(excludeProjects, name) {
			return fmt.Errorf("project %q specified in both --project and --exclude-project", name)
		}
	}

	if err := verifyProjectsExist(ctx, svc, org, projects); err != nil {
		return err
	}
	if err := verifyProjectsExist(ctx, svc, org, excludeProjects); err != nil {
		return err
	}

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

	allItems = filterByProject(allItems, projects, excludeProjects)

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
