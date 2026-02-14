package run

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type runShowJSON struct {
	ID               string    `json:"id"`
	Status           string    `json:"status"`
	Message          string    `json:"message"`
	TerraformVersion string    `json:"terraform_version"`
	HasChanges       bool      `json:"has_changes"`
	IsDestroy        bool      `json:"is_destroy"`
	CreatedAt        time.Time `json:"created_at"`
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

	cmd := &cobra.Command{
		Use:   "show [run-id]",
		Short: "Show run details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var runID string
			if len(args) > 0 {
				runID = args[0]
			}

			org := viper.GetString("org")
			if org == "" && workspaceName != "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}

			if runID == "" && workspaceName == "" {
				return fmt.Errorf("either run-id or --workspace is required")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runRunShow(svc, runID, org, workspaceName, watch)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "workspace name (get latest run)")
	cmd.Flags().BoolVarP(&watch, "watch", "W", false, "watch run status until completion")

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
		})
		if err != nil {
			return fmt.Errorf("failed to list runs: %w", err)
		}

		if len(runList.Items) == 0 {
			return fmt.Errorf("no runs found for workspace %q", workspaceName)
		}

		r = runList.Items[0]
		// workspace から取得した場合は run ID を保存
		runID = r.ID
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
	if viper.GetBool("json") {
		return output.PrintJSON(os.Stdout, runShowJSON{
			ID:               r.ID,
			Status:           string(r.Status),
			Message:          r.Message,
			TerraformVersion: r.TerraformVersion,
			HasChanges:       r.HasChanges,
			IsDestroy:        r.IsDestroy,
			CreatedAt:        r.CreatedAt,
		})
	}

	pairs := []output.KeyValue{
		{Key: "ID", Value: r.ID},
		{Key: "Status", Value: string(r.Status)},
		{Key: "Message", Value: r.Message},
		{Key: "Terraform Version", Value: r.TerraformVersion},
		{Key: "Has Changes", Value: strconv.FormatBool(r.HasChanges)},
		{Key: "Is Destroy", Value: strconv.FormatBool(r.IsDestroy)},
		{Key: "Created At", Value: r.CreatedAt.Format("2006-01-02 15:04:05")},
	}

	if r.StatusTimestamps != nil {
		if !r.StatusTimestamps.PlanQueueableAt.IsZero() {
			pairs = append(pairs, output.KeyValue{Key: "Plan Queueable At", Value: r.StatusTimestamps.PlanQueueableAt.Format("2006-01-02 15:04:05")})
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

	currentStatus := initialRun.Status

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

			// ステータスが変化した場合のみ出力
			if r.Status != currentStatus {
				if !viper.GetBool("json") {
					_, _ = fmt.Fprintln(os.Stdout, formatStatusUpdate(time.Now(), r.Status))
				}
				currentStatus = r.Status
			}

			// 終了ステータスに到達したら終了
			if isTerminalStatus(r.Status) {
				// JSON モードの場合は最終結果を出力
				if viper.GetBool("json") {
					return displayRun(r)
				}
				return nil
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
