package variable

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

type variableJSON struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Category  string `json:"category"`
	Sensitive bool   `json:"sensitive"`
	HCL       bool   `json:"hcl"`
}

type variableListService interface {
	client.WorkspaceService
	client.VariableService
}

type variableListClientFactory func() (variableListService, error)

func defaultVariableListClientFactory() (variableListService, error) {
	return client.NewClientWrapper()
}

func newCmdVariableList() *cobra.Command {
	return newCmdVariableListWith(defaultVariableListClientFactory)
}

func newCmdVariableListWith(clientFn variableListClientFactory) *cobra.Command {
	var workspaceName string

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List variables for a workspace",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}
			if workspaceName == "" {
				return fmt.Errorf("workspace is required: use --workspace/-w flag")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runVariableList(svc, org, workspaceName)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "workspace name (required)")

	return cmd
}

func runVariableList(svc variableListService, org, workspaceName string) error {
	ctx := context.Background()

	ws, err := svc.ReadWorkspace(ctx, org, workspaceName)
	if err != nil {
		return fmt.Errorf("failed to read workspace %q: %w", workspaceName, err)
	}

	opts := &tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{PageSize: 100},
	}

	var allItems []*tfe.Variable
	for {
		varList, err := svc.ListVariables(ctx, ws.ID, opts)
		if err != nil {
			return fmt.Errorf("failed to list variables: %w", err)
		}
		allItems = append(allItems, varList.Items...)
		if varList.Pagination == nil || varList.NextPage == 0 {
			break
		}
		opts.PageNumber = varList.NextPage
	}

	if viper.GetBool("json") {
		items := make([]variableJSON, 0, len(allItems))
		for _, v := range allItems {
			value := v.Value
			if v.Sensitive {
				value = "(sensitive)"
			}
			items = append(items, variableJSON{
				Key:       v.Key,
				Value:     value,
				Category:  string(v.Category),
				Sensitive: v.Sensitive,
				HCL:       v.HCL,
			})
		}
		return output.PrintJSON(os.Stdout, items)
	}

	headers := []string{"KEY", "VALUE", "CATEGORY", "SENSITIVE", "HCL"}
	rows := make([][]string, 0, len(allItems))
	for _, v := range allItems {
		value := v.Value
		if v.Sensitive {
			value = "(sensitive)"
		}
		rows = append(rows, []string{
			v.Key,
			value,
			string(v.Category),
			strconv.FormatBool(v.Sensitive),
			strconv.FormatBool(v.HCL),
		})
	}

	output.Print(os.Stdout, headers, rows)
	return nil
}
