package project

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

type projectShowService interface {
	client.ProjectService
	client.WorkspaceService
}

type projectShowJSON struct {
	Name                 string              `json:"name"`
	ID                   string              `json:"id"`
	Description          string              `json:"description"`
	DefaultExecutionMode string              `json:"default_execution_mode"`
	Workspaces           []showWorkspaceJSON `json:"workspaces"`
}

type showWorkspaceJSON struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type projectShowClientFactory func() (projectShowService, error)

func defaultProjectShowClientFactory() (projectShowService, error) {
	return client.NewClientWrapper()
}

func newCmdProjectShow() *cobra.Command {
	return newCmdProjectShowWith(defaultProjectShowClientFactory)
}

func newCmdProjectShowWith(clientFn projectShowClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "show <name>",
		Short:        "Show project details",
		Args:         cobra.ExactArgs(1),
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
			return runProjectShow(svc, org, args[0])
		},
	}
	return cmd
}

func runProjectShow(svc projectShowService, org, name string) error {
	ctx := context.Background()

	// Find project by name
	projList, err := svc.ListProjects(ctx, org, &tfe.ProjectListOptions{
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	var found *tfe.Project
	for _, p := range projList.Items {
		if p.Name == name {
			found = p
			break
		}
	}
	if found == nil {
		return fmt.Errorf("project %q not found in organization %q", name, org)
	}

	// List workspaces in this project
	workspaces, err := listProjectWorkspaces(ctx, svc, org, found.ID)
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if viper.GetBool("json") {
		wsItems := make([]showWorkspaceJSON, 0, len(workspaces))
		for _, ws := range workspaces {
			wsItems = append(wsItems, showWorkspaceJSON{
				Name: ws.Name,
				ID:   ws.ID,
			})
		}
		return output.PrintJSON(os.Stdout, projectShowJSON{
			Name:                 found.Name,
			ID:                   found.ID,
			Description:          found.Description,
			DefaultExecutionMode: found.DefaultExecutionMode,
			Workspaces:           wsItems,
		})
	}

	pairs := []output.KeyValue{
		{Key: "Name", Value: found.Name},
		{Key: "ID", Value: found.ID},
		{Key: "Description", Value: found.Description},
		{Key: "Default Execution Mode", Value: found.DefaultExecutionMode},
		{Key: "Workspaces", Value: strconv.Itoa(len(workspaces))},
	}

	output.PrintKeyValue(os.Stdout, pairs)

	if len(workspaces) > 0 {
		fmt.Fprintln(os.Stderr)
		headers := []string{"NAME", "ID"}
		rows := make([][]string, 0, len(workspaces))
		for _, ws := range workspaces {
			rows = append(rows, []string{ws.Name, ws.ID})
		}
		output.Print(os.Stdout, headers, rows)
	}

	return nil
}

func listProjectWorkspaces(ctx context.Context, svc client.WorkspaceService, org, projectID string) ([]*tfe.Workspace, error) {
	opts := &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{PageSize: 100},
		ProjectID:   projectID,
	}

	var all []*tfe.Workspace
	for {
		wsList, err := svc.ListWorkspaces(ctx, org, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, wsList.Items...)
		if wsList.Pagination == nil || wsList.NextPage == 0 {
			break
		}
		opts.PageNumber = wsList.NextPage
	}
	return all, nil
}
