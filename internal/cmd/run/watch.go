package run

import (
	"context"
	"fmt"
	"os"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"
)

// watchRun polls the run status until it reaches a terminal state.
func watchRun(ctx context.Context, svc runShowService, runID string, initialRun *tfe.Run, pollInterval time.Duration) error {
	// In watch mode, skip fetching resource changes (inefficient to fetch on every poll)
	var resourceChanges []resourceChange

	// If already in terminal status, display only the initial output and exit
	if isTerminalStatus(initialRun.Status) {
		// Fetch resource changes only when terminal status is reached
		if initialRun.Plan != nil && initialRun.HasChanges {
			planJSONBytes, err := svc.ReadPlanJSONOutput(ctx, initialRun.Plan.ID)
			if err == nil {
				resourceChanges, _ = extractResourceChanges(planJSONBytes)
			}
		}
		return displayRun(initialRun, resourceChanges)
	}

	// In non-JSON mode, display initial output and separator
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
				// Possibly a transient error; print warning and continue
				fmt.Fprintf(os.Stderr, "Warning: failed to read run: %v\n", err)
				continue
			}

			// Output status on each poll
			if !viper.GetBool("json") {
				_, _ = fmt.Fprintln(os.Stdout, formatStatusUpdate(time.Now(), r.Status))
			}

			// When terminal status is reached, display final result and exit
			if isTerminalStatus(r.Status) {
				// Fetch resource changes when terminal status is reached
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
