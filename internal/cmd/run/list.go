package run

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type runJSON struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	PlanOnly   bool      `json:"plan_only"`
	HasChanges bool      `json:"has_changes"`
	CreatedAt  time.Time `json:"created_at"`
}

// runListService combines RunService and WorkspaceService for workspace ID resolution.
type runListService interface {
	client.RunService
	client.WorkspaceService
}

type runListClientFactory func() (runListService, error)

func defaultRunListClientFactory() (runListService, error) {
	return client.NewClientWrapper()
}

func newCmdRunList() *cobra.Command {
	return newCmdRunListWith(defaultRunListClientFactory)
}

func newCmdRunListWith(clientFn runListClientFactory) *cobra.Command {
	var workspaceName string
	var status string

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List runs for a workspace",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}
			if workspaceName == "" {
				return fmt.Errorf("workspace is required: use --workspace/-w flag")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runRunList(svc, org, workspaceName, status)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "workspace name (required)")
	cmd.Flags().StringVar(&status, "status", "", "filter by run status (comma-separated, e.g. applied,errored)")

	return cmd
}

func runRunList(svc runListService, org, workspaceName, status string) error {
	ctx := context.Background()

	// Resolve workspace name to ID
	ws, err := svc.ReadWorkspace(ctx, org, workspaceName)
	if err != nil {
		return fmt.Errorf("failed to read workspace %q: %w", workspaceName, err)
	}

	opts := &tfe.RunListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
		// plan_only is excluded by default, so explicitly specify all operation types
		Operation: "plan_and_apply,plan_only,refresh_only,destroy,empty_apply,save_plan",
		Status:    status,
	}

	var allItems []*tfe.Run
	for {
		runList, err := svc.ListRuns(ctx, ws.ID, opts)
		if err != nil {
			return fmt.Errorf("failed to list runs: %w", err)
		}
		allItems = append(allItems, runList.Items...)
		if runList.Pagination == nil || runList.NextPage == 0 {
			break
		}
		opts.PageNumber = runList.NextPage
	}

	if viper.GetBool("json") {
		items := make([]runJSON, 0, len(allItems))
		for _, r := range allItems {
			items = append(items, runJSON{
				ID:         r.ID,
				Status:     string(r.Status),
				Message:    r.Message,
				PlanOnly:   r.PlanOnly,
				HasChanges: r.HasChanges,
				CreatedAt:  r.CreatedAt,
			})
		}
		return output.PrintJSON(os.Stdout, items)
	}

	headers := []string{"ID", "STATUS", "MESSAGE", "PLAN ONLY", "HAS CHANGES", "CREATED AT"}
	rows := make([][]string, 0, len(allItems))
	for _, r := range allItems {
		rows = append(rows, []string{
			r.ID,
			string(r.Status),
			truncate(r.Message, 50),
			strconv.FormatBool(r.PlanOnly),
			strconv.FormatBool(r.HasChanges),
			r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	output.Print(os.Stdout, headers, rows)
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
