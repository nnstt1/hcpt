package run

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

// runLogsService combines the services needed to fetch apply logs.
type runLogsService interface {
	client.RunService
	client.WorkspaceService
	client.ApplyService
}

type runLogsClientFactory func() (runLogsService, error)

func defaultRunLogsClientFactory() (runLogsService, error) {
	return client.NewClientWrapper()
}

func newCmdRunLogs() *cobra.Command {
	return newCmdRunLogsWith(defaultRunLogsClientFactory)
}

func newCmdRunLogsWith(clientFn runLogsClientFactory) *cobra.Command {
	var workspaceName string
	var errorOnly bool

	cmd := &cobra.Command{
		Use:          "logs [run-id]",
		Short:        "Show apply logs for a run",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var runID string
			if len(args) > 0 {
				runID = args[0]
			}

			if runID == "" && workspaceName == "" {
				return fmt.Errorf("either run-id or --workspace/-w is required")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}

			return runRunLogs(svc, runID, viper.GetString("org"), workspaceName, errorOnly)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace name (uses latest run)")
	cmd.Flags().BoolVar(&errorOnly, "error-only", false, "Show only error-level log lines")

	return cmd
}

func runRunLogs(svc runLogsService, runID, org, workspaceName string, errorOnly bool) error {
	ctx := context.Background()

	// If no run ID given, get the latest run from the workspace
	if runID == "" {
		if org == "" {
			return fmt.Errorf("organization is required when using --workspace/-w (set via --org or config)")
		}

		ws, err := svc.ReadWorkspace(ctx, org, workspaceName)
		if err != nil {
			return fmt.Errorf("failed to read workspace: %w", err)
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

		runID = runList.Items[0].ID
	}

	// Read the run including the apply
	r, err := svc.ReadRunWithApply(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to read run: %w", err)
	}

	if r.Apply == nil {
		return fmt.Errorf("run %q does not have an apply (status: %s)", runID, r.Status)
	}

	logs, err := svc.ReadApplyLogs(ctx, r.Apply.ID)
	if err != nil {
		return fmt.Errorf("failed to read apply logs: %w", err)
	}

	return printLogs(os.Stdout, logs, errorOnly)
}

func printLogs(w io.Writer, r io.Reader, errorOnly bool) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if errorOnly && !isErrorLine(line) {
			continue
		}
		fmt.Fprintln(w, line)
	}
	return scanner.Err()
}

func isErrorLine(line string) bool {
	return strings.Contains(line, `"@level":"error"`)
}
