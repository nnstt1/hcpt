package drift

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

type driftShowService interface {
	client.WorkspaceService
	client.AssessmentService
}

type driftShowClientFactory func() (driftShowService, error)

func defaultDriftShowClientFactory() (driftShowService, error) {
	return client.NewClientWrapper()
}

func newCmdDriftShow() *cobra.Command {
	return newCmdDriftShowWith(defaultDriftShowClientFactory)
}

func newCmdDriftShowWith(clientFn driftShowClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <workspace>",
		Short: "Show drift detection detail for a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runDriftShow(svc, org, args[0])
		},
	}
	return cmd
}

type driftShowJSON struct {
	Workspace          string `json:"workspace"`
	Drifted            *bool  `json:"drifted"`
	ResourcesDrifted   *int   `json:"resources_drifted"`
	ResourcesUndrifted *int   `json:"resources_undrifted"`
	LastAssessment     string `json:"last_assessment"`
}

func runDriftShow(svc driftShowService, org, name string) error {
	ctx := context.Background()
	ws, err := svc.ReadWorkspace(ctx, org, name)
	if err != nil {
		return fmt.Errorf("failed to read workspace %q: %w", name, err)
	}

	result, err := svc.ReadCurrentAssessment(ctx, ws.ID)
	if err != nil {
		return fmt.Errorf("failed to read assessment for workspace %q: %w", name, err)
	}

	if viper.GetBool("json") {
		return output.PrintJSON(os.Stdout, toDriftShowJSON(ws, result))
	}

	pairs := buildDriftShowKeyValues(ws, result)
	output.PrintKeyValue(os.Stdout, pairs)
	return nil
}

func toDriftShowJSON(ws *tfe.Workspace, result *client.AssessmentResult) driftShowJSON {
	d := driftShowJSON{
		Workspace: ws.Name,
	}
	if result != nil {
		d.Drifted = &result.Drifted
		d.ResourcesDrifted = &result.ResourcesDrifted
		d.ResourcesUndrifted = &result.ResourcesUndrifted
		d.LastAssessment = result.CreatedAt
	}
	return d
}

func buildDriftShowKeyValues(ws *tfe.Workspace, result *client.AssessmentResult) []output.KeyValue {
	pairs := []output.KeyValue{
		{Key: "Workspace", Value: ws.Name},
	}

	if result != nil {
		pairs = append(pairs,
			output.KeyValue{Key: "Drifted", Value: strconv.FormatBool(result.Drifted)},
			output.KeyValue{Key: "Resources Drifted", Value: strconv.Itoa(result.ResourcesDrifted)},
			output.KeyValue{Key: "Resources Undrifted", Value: strconv.Itoa(result.ResourcesUndrifted)},
			output.KeyValue{Key: "Last Assessment", Value: result.CreatedAt},
		)
	} else {
		pairs = append(pairs,
			output.KeyValue{Key: "Drifted", Value: "not ready"},
			output.KeyValue{Key: "Resources Drifted", Value: "-"},
			output.KeyValue{Key: "Resources Undrifted", Value: "-"},
			output.KeyValue{Key: "Last Assessment", Value: "-"},
		)
	}

	return pairs
}
