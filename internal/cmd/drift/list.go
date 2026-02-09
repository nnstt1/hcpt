package drift

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type driftService interface {
	client.WorkspaceService
	client.AssessmentService
}

type driftClientFactory func() (driftService, error)

func defaultDriftClientFactory() (driftService, error) {
	return client.NewClientWrapper()
}

func newCmdDriftList() *cobra.Command {
	return newCmdDriftListWith(defaultDriftClientFactory)
}

func newCmdDriftListWith(clientFn driftClientFactory) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces with drift status",
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runDriftList(svc, org, all)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show all workspaces (default: drifted only)")

	return cmd
}

type driftJSON struct {
	Workspace          string `json:"workspace"`
	Drifted            *bool  `json:"drifted"`
	ResourcesDrifted   *int   `json:"resources_drifted"`
	ResourcesUndrifted *int   `json:"resources_undrifted"`
	LastAssessment     string `json:"last_assessment"`
}

const (
	maxConcurrency = 20
	apiRateLimit   = 25 // requests per second (API limit is 30, keep headroom)
	apiRateBurst   = 5
)

func runDriftList(svc driftService, org string, all bool) error {
	ctx := context.Background()
	opts := &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	}

	// Collect all workspaces first
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

	// Fetch assessment results concurrently with rate limiting
	type wsResult struct {
		ws     *tfe.Workspace
		result *client.AssessmentResult
	}
	indexed := make([]wsResult, len(allWorkspaces))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)
	limiter := rate.NewLimiter(rate.Limit(apiRateLimit), apiRateBurst)

	var mu sync.Mutex

	for i, ws := range allWorkspaces {
		indexed[i].ws = ws
		g.Go(func() error {
			if err := limiter.Wait(ctx); err != nil {
				return err
			}
			result, err := svc.ReadCurrentAssessment(ctx, ws.ID)
			if err != nil {
				return fmt.Errorf("failed to read assessment for workspace %q: %w", ws.Name, err)
			}
			mu.Lock()
			indexed[i].result = result
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// Filter results
	var results []wsResult
	for _, r := range indexed {
		if all || (r.result != nil && r.result.Drifted) {
			results = append(results, r)
		}
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
