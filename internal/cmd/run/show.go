package run

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

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

type runShowClientFactory func() (client.RunService, error)

func defaultRunShowClientFactory() (client.RunService, error) {
	return client.NewClientWrapper()
}

func newCmdRunShow() *cobra.Command {
	return newCmdRunShowWith(defaultRunShowClientFactory)
}

func newCmdRunShowWith(clientFn runShowClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <run-id>",
		Short: "Show run details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runRunShow(svc, args[0])
		},
	}
	return cmd
}

func runRunShow(svc client.RunService, runID string) error {
	ctx := context.Background()
	r, err := svc.ReadRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to read run %q: %w", runID, err)
	}

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
