package run

import (
	"context"
	"encoding/json"
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
	ID                   string           `json:"id"`
	Status               string           `json:"status"`
	Message              string           `json:"message"`
	TerraformVersion     string           `json:"terraform_version"`
	HasChanges           bool             `json:"has_changes"`
	ResourceAdditions    int              `json:"resource_additions"`
	ResourceChanges      int              `json:"resource_changes"`
	ResourceDestructions int              `json:"resource_destructions"`
	CreatedAt            time.Time        `json:"created_at"`
	PlannedAt            *time.Time       `json:"planned_at,omitempty"`
	AppliedAt            *time.Time       `json:"applied_at,omitempty"`
	Changes              []resourceChange `json:"changes,omitempty"`
}

type resourceChange struct {
	Address string            `json:"address"`
	Type    string            `json:"type"`
	Actions []string          `json:"actions"`
	Changes map[string]change `json:"changes,omitempty"`
}

type change struct {
	Before interface{} `json:"before"`
	After  interface{} `json:"after"`
}

// runShowService combines RunService, WorkspaceService, and PlanService for run details.
type runShowService interface {
	client.RunService
	client.WorkspaceService
	client.PlanService
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
	var planJSON bool

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
				// Try to auto-detect repository from Git remote
				detectedRepo, err := client.DetectGitHubRepository()
				if err != nil {
					return err
				}
				repoFullName = detectedRepo
			}

			if repoFullName != "" && !strings.Contains(repoFullName, "/") {
				return fmt.Errorf("--repo must be in format 'owner/repo'")
			}

			if runID == "" && prNumber == 0 && workspaceName == "" {
				return fmt.Errorf("either run-id, --pr, or --workspace/-w is required")
			}

			if watch && planJSON {
				return fmt.Errorf("--plan-json cannot be used with --watch")
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

			return runRunShow(svc, runID, org, workspaceName, watch, planJSON)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "workspace name (get latest run)")
	cmd.Flags().BoolVarP(&watch, "watch", "W", false, "watch run status until completion")
	cmd.Flags().IntVarP(&prNumber, "pr", "p", 0, "GitHub pull request number")
	cmd.Flags().StringVarP(&repoFullName, "repo", "r", "", "GitHub repository (owner/repo)")
	cmd.Flags().BoolVar(&planJSON, "plan-json", false, "output plan JSON details")

	return cmd
}

func runRunShow(svc runShowService, runID string, org string, workspaceName string, watch bool, planJSON bool) error {
	return runRunShowWithInterval(svc, runID, org, workspaceName, watch, planJSON, 5*time.Second)
}

func runRunShowWithInterval(svc runShowService, runID string, org string, workspaceName string, watch bool, planJSON bool, pollInterval time.Duration) error {
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
		return fmt.Errorf("either run-id or --workspace/-w is required")
	}

	// watch モードの場合
	if watch {
		return watchRun(ctx, svc, runID, r, pollInterval)
	}

	// --plan-json が指定された場合
	if planJSON {
		// JSON モードでは displayPlanJSON が run 情報も含めて出力する
		return displayPlanJSON(ctx, svc, r)
	}

	// Plan がある場合は変更差分を取得
	var resourceChanges []resourceChange
	if r.Plan != nil && r.HasChanges {
		planJSONBytes, err := svc.ReadPlanJSONOutput(ctx, r.Plan.ID)
		if err == nil {
			resourceChanges, _ = extractResourceChanges(planJSONBytes)
		}
		// エラーは無視して、変更差分なしで表示
	}

	// 通常の displayRun を呼ぶ
	return displayRun(r, resourceChanges)
}

func displayRun(r *tfe.Run, resourceChanges []resourceChange) error {
	if viper.GetBool("json") {
		return output.PrintJSON(os.Stdout, toRunShowJSON(r, resourceChanges))
	}

	var additions, changes, destructions int
	if r.Plan != nil {
		additions = r.Plan.ResourceAdditions
		changes = r.Plan.ResourceChanges
		destructions = r.Plan.ResourceDestructions
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

	// 変更差分がある場合は表示
	if len(resourceChanges) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "")
		_, _ = fmt.Fprintln(os.Stdout, "Resource Changes:")
		for _, rc := range resourceChanges {
			action := strings.Join(rc.Actions, ", ")
			_, _ = fmt.Fprintf(os.Stdout, "- %s [%s]\n", rc.Address, action)

			// 属性の変更を表示
			if len(rc.Changes) > 0 {
				for attr, ch := range rc.Changes {
					beforeStr := formatValue(ch.Before)
					afterStr := formatValue(ch.After)
					_, _ = fmt.Fprintf(os.Stdout, "    %s: %s → %s\n", attr, beforeStr, afterStr)
				}
			}
		}
	}

	return nil
}

// extractResourceChanges extracts resource changes from plan JSON, excluding no-op changes.
func extractResourceChanges(planJSONBytes []byte) ([]resourceChange, error) {
	var planData struct {
		ResourceChanges []struct {
			Address string `json:"address"`
			Type    string `json:"type"`
			Change  struct {
				Actions []string               `json:"actions"`
				Before  map[string]interface{} `json:"before"`
				After   map[string]interface{} `json:"after"`
			} `json:"change"`
		} `json:"resource_changes"`
	}

	if err := json.Unmarshal(planJSONBytes, &planData); err != nil {
		return nil, err
	}

	var changes []resourceChange
	for _, rc := range planData.ResourceChanges {
		// Skip no-op changes
		if len(rc.Change.Actions) == 1 && rc.Change.Actions[0] == "no-op" {
			continue
		}

		// Extract attribute changes
		attrChanges := extractAttributeChanges(rc.Change.Before, rc.Change.After)

		changes = append(changes, resourceChange{
			Address: rc.Address,
			Type:    rc.Type,
			Actions: rc.Change.Actions,
			Changes: attrChanges,
		})
	}

	return changes, nil
}

// formatValue formats a value for display.
func formatValue(v interface{}) string {
	if v == nil {
		return "(null)"
	}

	switch val := v.(type) {
	case string:
		if val == "" {
			return "(empty)"
		}
		// Truncate long strings
		if len(val) > 100 {
			return val[:97] + "..."
		}
		return val
	case bool:
		return fmt.Sprintf("%t", val)
	case float64:
		// Check if it's an integer
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	default:
		str := fmt.Sprintf("%v", v)
		if len(str) > 100 {
			return str[:97] + "..."
		}
		return str
	}
}

// extractAttributeChanges compares before and after to find changed attributes.
func extractAttributeChanges(before, after map[string]interface{}) map[string]change {
	changes := make(map[string]change)

	// Attributes to skip (auto-generated or metadata fields)
	skipAttributes := map[string]bool{
		"id":                  true,
		"created_on":          true,
		"modified_on":         true,
		"updated_at":          true,
		"created_at":          true,
		"last_updated":        true,
		"timeouts":            true,
		"comment_modified_on": true,
		"tags_modified_on":    true,
		"self_link":           true,
		"fingerprint":         true,
		"etag":                true,
	}

	// Check all attributes in before and after
	allKeys := make(map[string]bool)
	for k := range before {
		allKeys[k] = true
	}
	for k := range after {
		allKeys[k] = true
	}

	for key := range allKeys {
		// Skip attributes in skip list
		if skipAttributes[key] {
			continue
		}

		beforeVal := before[key]
		afterVal := after[key]

		// Skip if both are nil
		if beforeVal == nil && afterVal == nil {
			continue
		}

		// Skip if after is nil (computed/unknown value)
		if afterVal == nil {
			continue
		}

		// Skip if values are equal
		if fmt.Sprintf("%v", beforeVal) == fmt.Sprintf("%v", afterVal) {
			continue
		}

		// Skip complex nested objects (maps, arrays) for simplicity
		if _, ok := beforeVal.(map[string]interface{}); ok {
			continue
		}
		if _, ok := afterVal.(map[string]interface{}); ok {
			continue
		}
		if _, ok := beforeVal.([]interface{}); ok {
			continue
		}
		if _, ok := afterVal.([]interface{}); ok {
			continue
		}

		changes[key] = change{
			Before: beforeVal,
			After:  afterVal,
		}
	}

	return changes
}

// toRunShowJSON converts a tfe.Run to runShowJSON structure.
func toRunShowJSON(r *tfe.Run, resourceChanges []resourceChange) runShowJSON {
	var additions, changes, destructions int
	if r.Plan != nil {
		additions = r.Plan.ResourceAdditions
		changes = r.Plan.ResourceChanges
		destructions = r.Plan.ResourceDestructions
	}

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
		Changes:              resourceChanges,
	}
	if r.StatusTimestamps != nil {
		if !r.StatusTimestamps.PlannedAt.IsZero() {
			j.PlannedAt = &r.StatusTimestamps.PlannedAt
		}
		if !r.StatusTimestamps.AppliedAt.IsZero() {
			j.AppliedAt = &r.StatusTimestamps.AppliedAt
		}
	}
	return j
}

// displayPlanJSON outputs the plan JSON details.
func displayPlanJSON(ctx context.Context, svc runShowService, r *tfe.Run) error {
	if r.Plan == nil {
		return fmt.Errorf("this run does not have a plan")
	}

	planJSONBytes, err := svc.ReadPlanJSONOutput(ctx, r.Plan.ID)
	if err != nil {
		return fmt.Errorf("failed to read plan JSON: %w", err)
	}

	// 変更差分を抽出
	resourceChanges, _ := extractResourceChanges(planJSONBytes)

	// --json フラグの有無で出力形式を変える
	if viper.GetBool("json") {
		// JSON モード: {"run": {...}, "plan_json": {...}} 形式
		var planData map[string]interface{}
		if err := json.Unmarshal(planJSONBytes, &planData); err != nil {
			return fmt.Errorf("failed to parse plan JSON: %w", err)
		}

		combined := map[string]interface{}{
			"run":       toRunShowJSON(r, resourceChanges),
			"plan_json": planData,
		}
		return output.PrintJSON(os.Stdout, combined)
	}

	// テーブルモード: まず run 情報を表示してから区切り線と Plan JSON を出力
	if err := displayRun(r, resourceChanges); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(os.Stdout, "---")
	_, _ = fmt.Fprintln(os.Stdout, string(planJSONBytes))
	return nil
}

// watchRun polls the run status until it reaches a terminal state.
func watchRun(ctx context.Context, svc runShowService, runID string, initialRun *tfe.Run, pollInterval time.Duration) error {
	// watch モードでは変更差分を取得しない（ポーリングごとに取得するのは非効率）
	var resourceChanges []resourceChange

	// すでに終了ステータスなら初回表示のみで終了
	if isTerminalStatus(initialRun.Status) {
		// 終了ステータスの場合のみ変更差分を取得
		if initialRun.Plan != nil && initialRun.HasChanges {
			planJSONBytes, err := svc.ReadPlanJSONOutput(ctx, initialRun.Plan.ID)
			if err == nil {
				resourceChanges, _ = extractResourceChanges(planJSONBytes)
			}
		}
		return displayRun(initialRun, resourceChanges)
	}

	// JSON モード以外の場合、初回表示と区切り線
	if !viper.GetBool("json") {
		if err := displayRun(initialRun, nil); err != nil {
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
				// 終了ステータスになったら変更差分を取得
				var finalChanges []resourceChange
				if r.Plan != nil && r.HasChanges {
					planJSONBytes, err := svc.ReadPlanJSONOutput(ctx, r.Plan.ID)
					if err == nil {
						finalChanges, _ = extractResourceChanges(planJSONBytes)
					}
				}
				if !viper.GetBool("json") {
					_, _ = fmt.Fprintln(os.Stdout, "---")
				}
				return displayRun(r, finalChanges)
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
