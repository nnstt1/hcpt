package run

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type runShowJSON struct {
	ID                   string     `json:"id"`
	Status               string     `json:"status"`
	Message              string     `json:"message"`
	TerraformVersion     string     `json:"terraform_version"`
	HasChanges           bool       `json:"has_changes"`
	ResourceAdditions    int        `json:"resource_additions"`
	ResourceChanges      int        `json:"resource_changes"`
	ResourceDestructions int        `json:"resource_destructions"`
	CreatedAt            time.Time  `json:"created_at"`
	PlannedAt            *time.Time `json:"planned_at,omitempty"`
	AppliedAt            *time.Time `json:"applied_at,omitempty"`
}

// runShowService combines RunService and WorkspaceService for workspace ID resolution.
type runShowService interface {
	client.RunService
	client.WorkspaceService
}

type runShowClientFactory func() (runShowService, error)

func defaultRunShowClientFactory() (runShowService, error) {
	return client.NewClientWrapper()
}

func newCmdRunShow() *cobra.Command {
	return newCmdRunShowWith(defaultRunShowClientFactory)
}

func newCmdRunShowWith(clientFn runShowClientFactory) *cobra.Command {
	var workspaceName string
	var watch bool
	var prNumber int
	var repoFullName string

	cmd := &cobra.Command{
		Use:          "show [run-id]",
		Short:        "Show run details",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var runID string
			if len(args) > 0 {
				runID = args[0]
			}

			org := viper.GetString("org")
			if org == "" && workspaceName != "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}

			// Validate flag combinations
			if runID != "" && prNumber > 0 {
				return fmt.Errorf("cannot specify both run-id and --pr")
			}

			if prNumber > 0 && repoFullName == "" {
				return fmt.Errorf("--repo is required when using --pr")
			}

			if repoFullName != "" && !strings.Contains(repoFullName, "/") {
				return fmt.Errorf("--repo must be in format 'owner/repo'")
			}

			if runID == "" && prNumber == 0 && workspaceName == "" {
				return fmt.Errorf("either run-id, --pr, or --workspace is required")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}

			// If --pr is specified, get run-id from GitHub
			if prNumber > 0 {
				ghClient, err := client.NewGitHubClientWrapper()
				if err != nil {
					return err
				}

				// Parse owner/repo
				parts := strings.Split(repoFullName, "/")
				owner, repo := parts[0], parts[1]

				ctx := context.Background()
				runID, err = ghClient.GetRunIDFromPR(ctx, owner, repo, prNumber, workspaceName)
				if err != nil {
					return err
				}
			}

			return runRunShow(svc, runID, org, workspaceName, watch)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "workspace name (get latest run)")
	cmd.Flags().BoolVarP(&watch, "watch", "W", false, "watch run status until completion")
	cmd.Flags().IntVarP(&prNumber, "pr", "p", 0, "GitHub pull request number")
	cmd.Flags().StringVarP(&repoFullName, "repo", "r", "", "GitHub repository (owner/repo)")

	return cmd
}

func runRunShow(svc runShowService, runID string, org string, workspaceName string, watch bool) error {
	return runRunShowWithInterval(svc, runID, org, workspaceName, watch, 5*time.Second)
}

func runRunShowWithInterval(svc runShowService, runID string, org string, workspaceName string, watch bool, pollInterval time.Duration) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var r *tfe.Run
	var err error

	// If run-id is specified, use it directly
	if runID != "" {
		r, err = svc.ReadRun(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to read run %q: %w", runID, err)
		}
	} else if workspaceName != "" {
		// Get latest run for workspace
		ws, err := svc.ReadWorkspace(ctx, org, workspaceName)
		if err != nil {
			return fmt.Errorf("failed to read workspace %q: %w", workspaceName, err)
		}

		runList, err := svc.ListRuns(ctx, ws.ID, &tfe.RunListOptions{
			ListOptions: tfe.ListOptions{PageSize: 1},
			Operation:   "plan_and_apply,plan_only,refresh_only,destroy,empty_apply,save_plan",
		})
		if err != nil {
			return fmt.Errorf("failed to list runs: %w", err)
		}

		if len(runList.Items) == 0 {
			return fmt.Errorf("no runs found for workspace %q", workspaceName)
		}

		// ListRuns では Plan が含まれないため、ReadRun で詳細を取得
		runID = runList.Items[0].ID
		r, err = svc.ReadRun(ctx, runID)
		if err != nil {
			return fmt.Errorf("failed to read run %q: %w", runID, err)
		}
	} else {
		return fmt.Errorf("either run-id or --workspace is required")
	}

	// watch モードの場合
	if watch {
		return watchRun(ctx, svc, runID, r, pollInterval)
	}

	return displayRun(r)
}

func displayRun(r *tfe.Run) error {
	var additions, changes, destructions int
	if r.Plan != nil {
		additions = r.Plan.ResourceAdditions
		changes = r.Plan.ResourceChanges
		destructions = r.Plan.ResourceDestructions
	}

	if viper.GetBool("json") {
		j := runShowJSON{
			ID:                   r.ID,
			Status:               string(r.Status),
			Message:              r.Message,
			TerraformVersion:     r.TerraformVersion,
			HasChanges:           r.HasChanges,
			ResourceAdditions:    additions,
			ResourceChanges:      changes,
			ResourceDestructions: destructions,
			CreatedAt:            r.CreatedAt,
		}
		if r.StatusTimestamps != nil {
			if !r.StatusTimestamps.PlannedAt.IsZero() {
				j.PlannedAt = &r.StatusTimestamps.PlannedAt
			}
			if !r.StatusTimestamps.AppliedAt.IsZero() {
				j.AppliedAt = &r.StatusTimestamps.AppliedAt
			}
		}
		return output.PrintJSON(os.Stdout, j)
	}

	hasChanges := strconv.FormatBool(r.HasChanges)
	planChanges := fmt.Sprintf("+%d ~%d -%d", additions, changes, destructions)
	if !isTerminalStatus(r.Status) {
		hasChanges = "-"
		planChanges = "-"
	}

	pairs := []output.KeyValue{
		{Key: "ID", Value: r.ID},
		{Key: "Status", Value: string(r.Status)},
		{Key: "Message", Value: r.Message},
		{Key: "Terraform Version", Value: r.TerraformVersion},
		{Key: "Has Changes", Value: hasChanges},
		{Key: "Plan Changes", Value: planChanges},
		{Key: "Created At", Value: r.CreatedAt.Format("2006-01-02 15:04:05")},
	}

	if r.StatusTimestamps != nil {
		if !r.StatusTimestamps.PlannedAt.IsZero() {
			pairs = append(pairs, output.KeyValue{Key: "Planned At", Value: r.StatusTimestamps.PlannedAt.Format("2006-01-02 15:04:05")})
		}
		if !r.StatusTimestamps.AppliedAt.IsZero() {
			pairs = append(pairs, output.KeyValue{Key: "Applied At", Value: r.StatusTimestamps.AppliedAt.Format("2006-01-02 15:04:05")})
		}
	}

	output.PrintKeyValue(os.Stdout, pairs)
	return nil
}

// watchRun polls the run status until it reaches a terminal state.
func watchRun(ctx context.Context, svc runShowService, runID string, initialRun *tfe.Run, pollInterval time.Duration) error {
	// すでに終了ステータスなら初回表示のみで終了
	if isTerminalStatus(initialRun.Status) {
		return displayRun(initialRun)
	}

	// JSON モード以外の場合、初回表示と区切り線
	if !viper.GetBool("json") {
		if err := displayRun(initialRun); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(os.Stdout, "---")
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			r, err := svc.ReadRun(ctx, runID)
			if err != nil {
				// 一時的なエラーの可能性があるため、警告を出力して継続
				fmt.Fprintf(os.Stderr, "Warning: failed to read run: %v\n", err)
				continue
			}

			// ポーリングごとにステータスを出力
			if !viper.GetBool("json") {
				_, _ = fmt.Fprintln(os.Stdout, formatStatusUpdate(time.Now(), r.Status))
			}

			// 終了ステータスに到達したら最終結果を表示して終了
			if isTerminalStatus(r.Status) {
				if !viper.GetBool("json") {
					_, _ = fmt.Fprintln(os.Stdout, "---")
				}
				return displayRun(r)
			}
		}
	}
}

// isTerminalStatus returns true if the status is a terminal state.
func isTerminalStatus(status tfe.RunStatus) bool {
	switch status {
	case tfe.RunApplied,
		tfe.RunErrored,
		tfe.RunCanceled,
		tfe.RunDiscarded,
		tfe.RunPlannedAndFinished,
		tfe.RunPlannedAndSaved:
		return true
	default:
		return false
	}
}

// formatStatusUpdate formats a status update line with timestamp.
func formatStatusUpdate(timestamp time.Time, status tfe.RunStatus) string {
	return fmt.Sprintf("%s  Status: %s", timestamp.Format("2006-01-02 15:04:05"), status)
}
