package workspace

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

type wsDriftService interface {
	client.WorkspaceService
	client.AssessmentService
}

type wsDriftClientFactory func() (wsDriftService, error)

func defaultWSDriftClientFactory() (wsDriftService, error) {
	return client.NewClientWrapper()
}

func newCmdWorkspaceDrift() *cobra.Command {
	return newCmdWorkspaceDriftWith(defaultWSDriftClientFactory)
}

func newCmdWorkspaceDriftWith(clientFn wsDriftClientFactory) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "drift [name]",
		Short: "Show drift detection status for workspaces",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}

			if all {
				return runWorkspaceDriftAll(svc, org)
			}

			if len(args) == 0 {
				return fmt.Errorf("workspace name is required, or use --all flag to list all workspaces")
			}
			return runWorkspaceDrift(svc, org, args[0])
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show drift status for all workspaces")

	return cmd
}

type driftJSON struct {
	Workspace          string `json:"workspace"`
	Drifted            *bool  `json:"drifted"`
	ResourcesDrifted   *int   `json:"resources_drifted"`
	ResourcesUndrifted *int   `json:"resources_undrifted"`
	LastAssessment     string `json:"last_assessment"`
}

func runWorkspaceDrift(svc wsDriftService, org, name string) error {
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
		return output.PrintJSON(os.Stdout, toDriftJSON(ws, result))
	}

	pairs := buildDriftKeyValues(ws, result)
	output.PrintKeyValue(os.Stdout, pairs)
	return nil
}

func runWorkspaceDriftAll(svc wsDriftService, org string) error {
	ctx := context.Background()
	opts := &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	}

	var allWorkspaces []*tfe.Workspace
	for {
		wsList, err := svc.ListWorkspaces(ctx, org, opts)
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}
		allWorkspaces = append(allWorkspaces, wsList.Items...)
		if wsList.Pagination == nil || wsList.CurrentPage >= wsList.TotalPages {
			break
		}
		opts.PageNumber = wsList.NextPage
	}

	type wsResult struct {
		ws     *tfe.Workspace
		result *client.AssessmentResult
	}
	results := make([]wsResult, 0, len(allWorkspaces))

	for _, ws := range allWorkspaces {
		result, err := svc.ReadCurrentAssessment(ctx, ws.ID)
		if err != nil {
			return fmt.Errorf("failed to read assessment for workspace %q: %w", ws.Name, err)
		}
		results = append(results, wsResult{ws: ws, result: result})
	}

	if viper.GetBool("json") {
		items := make([]driftJSON, 0, len(results))
		for _, r := range results {
			items = append(items, toDriftJSON(r.ws, r.result))
		}
		return output.PrintJSON(os.Stdout, items)
	}

	headers := []string{"WORKSPACE", "DRIFTED", "RESOURCES DRIFTED", "LAST ASSESSMENT"}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		rows = append(rows, buildDriftRow(r.ws, r.result))
	}

	output.Print(os.Stdout, headers, rows)
	return nil
}

func buildDriftKeyValues(ws *tfe.Workspace, result *client.AssessmentResult) []output.KeyValue {
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

func buildDriftRow(ws *tfe.Workspace, result *client.AssessmentResult) []string {
	if result != nil {
		return []string{
			ws.Name,
			strconv.FormatBool(result.Drifted),
			strconv.Itoa(result.ResourcesDrifted),
			result.CreatedAt,
		}
	}
	return []string{
		ws.Name,
		"not ready",
		"-",
		"-",
	}
}

func toDriftJSON(ws *tfe.Workspace, result *client.AssessmentResult) driftJSON {
	d := driftJSON{
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
