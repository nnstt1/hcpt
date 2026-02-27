package variable

import (
	"context"
	"fmt"
	"os"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
	"github.com/nnstt1/hcpt/internal/parser"
	"github.com/nnstt1/hcpt/internal/prompt"
)

type variableImportService interface {
	client.WorkspaceService
	client.VariableService
}

type variableImportClientFactory func() (variableImportService, error)

func defaultVariableImportClientFactory() (variableImportService, error) {
	return client.NewClientWrapper()
}

func newCmdVariableImport() *cobra.Command {
	return newCmdVariableImportWith(defaultVariableImportClientFactory)
}

func newCmdVariableImportWith(clientFn variableImportClientFactory) *cobra.Command {
	var (
		workspaceName string
		category      string
		sensitive     bool
		overwrite     bool
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:          "import <file>",
		Short:        "Import variables from a .tfvars or .tfvars.json file",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]
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

			categoryType := tfe.CategoryTerraform
			if category == "env" {
				categoryType = tfe.CategoryEnv
			}

			return runVariableImport(svc, org, workspaceName, filename, categoryType, sensitive, overwrite, dryRun)
		},
	}

	cmd.Flags().StringVarP(&workspaceName, "workspace", "w", "", "workspace name (required)")
	cmd.Flags().StringVar(&category, "category", "terraform", "variable category (terraform or env)")
	cmd.Flags().BoolVar(&sensitive, "sensitive", false, "mark all imported variables as sensitive")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "auto-overwrite existing variables without prompting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without actually creating/updating variables")

	return cmd
}

func runVariableImport(svc variableImportService, org, workspaceName, filename string,
	category tfe.CategoryType, sensitive, overwrite, dryRun bool) error {
	ctx := context.Background()

	// 1. ファイル存在チェック
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filename)
	}

	// 2. ファイルパース
	vars, err := parser.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	if len(vars) == 0 {
		fmt.Println("No variables found in file")
		return nil
	}

	// 3. ワークスペース取得
	ws, err := svc.ReadWorkspace(ctx, org, workspaceName)
	if err != nil {
		return fmt.Errorf("failed to read workspace %q: %w", workspaceName, err)
	}

	// 4. 既存変数取得（重複チェック用）
	varList, err := svc.ListVariables(ctx, ws.ID, &tfe.VariableListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list variables: %w", err)
	}

	existingVars := make(map[string]*tfe.Variable)
	for _, v := range varList.Items {
		if v.Category == category {
			existingVars[v.Key] = v
		}
	}

	// 5. 変数を順次処理
	createdCount := 0
	updatedCount := 0
	skippedCount := 0

	for i, variable := range vars {
		fmt.Printf("[%d/%d] Setting variable: %s...\n", i+1, len(vars), variable.Key)

		existing, exists := existingVars[variable.Key]

		// 重複チェック
		if exists && !overwrite && !dryRun {
			ok, err := prompt.Confirm(fmt.Sprintf("Variable %q already exists. Overwrite?", variable.Key))
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}
			if !ok {
				fmt.Printf("  Skipped variable %q\n", variable.Key)
				skippedCount++
				continue
			}
		}

		// Dry-run チェック
		if dryRun {
			if exists {
				fmt.Printf("  [DRY RUN] Would update variable %q\n", variable.Key)
			} else {
				fmt.Printf("  [DRY RUN] Would create variable %q\n", variable.Key)
			}
			continue
		}

		// Update or Create
		if exists {
			opts := tfe.VariableUpdateOptions{
				Key:       &variable.Key,
				Value:     &variable.Value,
				Sensitive: &sensitive,
				HCL:       &variable.IsHCL,
			}
			_, err := svc.UpdateVariable(ctx, ws.ID, existing.ID, opts)
			if err != nil {
				return fmt.Errorf("failed to update variable %q: %w", variable.Key, err)
			}
			fmt.Printf("  ✓ Updated variable %q\n", variable.Key)
			updatedCount++
		} else {
			opts := tfe.VariableCreateOptions{
				Key:       &variable.Key,
				Value:     &variable.Value,
				Category:  &category,
				Sensitive: &sensitive,
				HCL:       &variable.IsHCL,
			}
			_, err := svc.CreateVariable(ctx, ws.ID, opts)
			if err != nil {
				return fmt.Errorf("failed to create variable %q: %w", variable.Key, err)
			}
			fmt.Printf("  ✓ Created variable %q\n", variable.Key)
			createdCount++
		}
	}

	// 6. サマリー表示
	if !dryRun {
		fmt.Printf("\nSuccessfully imported %d variable(s) to workspace %q\n", createdCount+updatedCount, workspaceName)
		if createdCount > 0 {
			fmt.Printf("  Created: %d\n", createdCount)
		}
		if updatedCount > 0 {
			fmt.Printf("  Updated: %d\n", updatedCount)
		}
		if skippedCount > 0 {
			fmt.Printf("  Skipped: %d\n", skippedCount)
		}
	}

	return nil
}
