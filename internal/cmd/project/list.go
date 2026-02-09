package project

import (
	"context"
	"fmt"
	"os"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type projectJSON struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Description string `json:"description"`
}

type projectListClientFactory func() (client.ProjectService, error)

func defaultProjectListClientFactory() (client.ProjectService, error) {
	return client.NewClientWrapper()
}

func newCmdProjectList() *cobra.Command {
	return newCmdProjectListWith(defaultProjectListClientFactory)
}

func newCmdProjectListWith(clientFn projectListClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects in an organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runProjectList(svc, org)
		},
	}
	return cmd
}

func runProjectList(svc client.ProjectService, org string) error {
	ctx := context.Background()
	opts := &tfe.ProjectListOptions{
		ListOptions: tfe.ListOptions{PageSize: 100},
	}

	var allItems []*tfe.Project
	for {
		projList, err := svc.ListProjects(ctx, org, opts)
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}
		allItems = append(allItems, projList.Items...)
		if projList.Pagination == nil || projList.NextPage == 0 {
			break
		}
		opts.PageNumber = projList.NextPage
	}

	if viper.GetBool("json") {
		items := make([]projectJSON, 0, len(allItems))
		for _, p := range allItems {
			items = append(items, projectJSON{
				Name:        p.Name,
				ID:          p.ID,
				Description: p.Description,
			})
		}
		return output.PrintJSON(os.Stdout, items)
	}

	headers := []string{"NAME", "ID", "DESCRIPTION"}
	rows := make([][]string, 0, len(allItems))
	for _, p := range allItems {
		rows = append(rows, []string{
			p.Name,
			p.ID,
			p.Description,
		})
	}

	output.Print(os.Stdout, headers, rows)
	return nil
}
