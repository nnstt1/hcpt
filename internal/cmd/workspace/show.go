package workspace

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/output"
)

type wsShowClientFactory func() (client.WorkspaceService, error)

func defaultWSShowClientFactory() (client.WorkspaceService, error) {
	return client.NewClientWrapper()
}

func newCmdWorkspaceShow() *cobra.Command {
	return newCmdWorkspaceShowWith(defaultWSShowClientFactory)
}

func newCmdWorkspaceShowWith(clientFn wsShowClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show workspace details",
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
			return runWorkspaceShow(svc, org, args[0])
		},
	}
	return cmd
}

func runWorkspaceShow(svc client.WorkspaceService, org, name string) error {
	ctx := context.Background()
	ws, err := svc.ReadWorkspace(ctx, org, name)
	if err != nil {
		return fmt.Errorf("failed to read workspace %q: %w", name, err)
	}

	if viper.GetBool("json") {
		return output.PrintJSON(os.Stdout, toWorkspaceJSON(ws))
	}

	pairs := []output.KeyValue{
		{Key: "Name", Value: ws.Name},
		{Key: "ID", Value: ws.ID},
		{Key: "Description", Value: ws.Description},
		{Key: "Execution Mode", Value: ws.ExecutionMode},
		{Key: "Terraform Version", Value: ws.TerraformVersion},
		{Key: "Locked", Value: strconv.FormatBool(ws.Locked)},
		{Key: "Auto Apply", Value: strconv.FormatBool(ws.AutoApply)},
		{Key: "Working Directory", Value: ws.WorkingDirectory},
		{Key: "Resource Count", Value: strconv.Itoa(ws.ResourceCount)},
		{Key: "Created At", Value: ws.CreatedAt.Format("2006-01-02 15:04:05")},
		{Key: "Updated At", Value: ws.UpdatedAt.Format("2006-01-02 15:04:05")},
	}

	output.PrintKeyValue(os.Stdout, pairs)
	return nil
}
