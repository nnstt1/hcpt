package org

import (
	"context"
	"fmt"
	"os"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type orgJSON struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type orgListClientFactory func() (client.OrganizationService, error)

func defaultOrgListClientFactory() (client.OrganizationService, error) {
	return client.NewClientWrapper()
}

func newCmdOrgList() *cobra.Command {
	return newCmdOrgListWith(defaultOrgListClientFactory)
}

func newCmdOrgListWith(clientFn orgListClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List organizations",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runOrgList(svc)
		},
	}
	return cmd
}

func runOrgList(svc client.OrganizationService) error {
	ctx := context.Background()
	orgList, err := svc.ListOrganizations(ctx, &tfe.OrganizationListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}

	if viper.GetBool("json") {
		items := make([]orgJSON, 0, len(orgList.Items))
		for _, o := range orgList.Items {
			items = append(items, orgJSON{
				Name:      o.Name,
				Email:     o.Email,
				CreatedAt: o.CreatedAt,
			})
		}
		return output.PrintJSON(os.Stdout, items)
	}

	headers := []string{"NAME", "EMAIL", "CREATED AT"}
	rows := make([][]string, 0, len(orgList.Items))
	for _, org := range orgList.Items {
		rows = append(rows, []string{
			org.Name,
			org.Email,
			org.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	output.Print(os.Stdout, headers, rows)
	return nil
}
