package org

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

type orgShowJSON struct {
	Name         string           `json:"name"`
	Email        string           `json:"email"`
	Plan         string           `json:"plan"`
	CreatedAt    string           `json:"created_at"`
	Entitlements entitlementsJSON `json:"entitlements"`
}

type entitlementsJSON struct {
	Agents                bool `json:"agents"`
	AuditLogging          bool `json:"audit_logging"`
	CostEstimation        bool `json:"cost_estimation"`
	Operations            bool `json:"operations"`
	PrivateModuleRegistry bool `json:"private_module_registry"`
	RunTasks              bool `json:"run_tasks"`
	SSO                   bool `json:"sso"`
	Sentinel              bool `json:"sentinel"`
	StateStorage          bool `json:"state_storage"`
	Teams                 bool `json:"teams"`
	VCSIntegrations       bool `json:"vcs_integrations"`
}

type orgShowService interface {
	client.OrganizationService
	client.SubscriptionService
}

type orgShowClientFactory func() (orgShowService, error)

func defaultOrgShowClientFactory() (orgShowService, error) {
	return client.NewClientWrapper()
}

func newCmdOrgShow() *cobra.Command {
	return newCmdOrgShowWith(defaultOrgShowClientFactory)
}

func newCmdOrgShowWith(clientFn orgShowClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show organization details",
		RunE: func(cmd *cobra.Command, args []string) error {
			org := viper.GetString("org")
			if org == "" {
				return fmt.Errorf("organization is required: use --org flag, TFE_ORG env, or set 'org' in config file")
			}

			svc, err := clientFn()
			if err != nil {
				return err
			}
			return runOrgShow(svc, org)
		},
	}
	return cmd
}

func runOrgShow(svc orgShowService, orgName string) error {
	ctx := context.Background()

	org, err := svc.ReadOrganization(ctx, orgName)
	if err != nil {
		return fmt.Errorf("failed to read organization %q: %w", orgName, err)
	}

	sub, err := svc.ReadSubscription(ctx, orgName)
	if err != nil {
		return fmt.Errorf("failed to read subscription for %q: %w", orgName, err)
	}

	entitlements, err := svc.ReadEntitlements(ctx, orgName)
	if err != nil {
		return fmt.Errorf("failed to read entitlements for %q: %w", orgName, err)
	}

	if viper.GetBool("json") {
		return output.PrintJSON(os.Stdout, orgShowJSON{
			Name:      org.Name,
			Email:     org.Email,
			Plan:      sub.PlanName,
			CreatedAt: org.CreatedAt.Format("2006-01-02 15:04:05"),
			Entitlements: entitlementsJSON{
				Agents:                entitlements.Agents,
				AuditLogging:          entitlements.AuditLogging,
				CostEstimation:        entitlements.CostEstimation,
				Operations:            entitlements.Operations,
				PrivateModuleRegistry: entitlements.PrivateModuleRegistry,
				RunTasks:              entitlements.RunTasks,
				SSO:                   entitlements.SSO,
				Sentinel:              entitlements.Sentinel,
				StateStorage:          entitlements.StateStorage,
				Teams:                 entitlements.Teams,
				VCSIntegrations:       entitlements.VCSIntegrations,
			},
		})
	}

	pairs := []output.KeyValue{
		{Key: "Name", Value: org.Name},
		{Key: "Email", Value: org.Email},
		{Key: "Plan", Value: sub.PlanName},
		{Key: "Created At", Value: org.CreatedAt.Format("2006-01-02 15:04:05")},
	}

	entPairs := []struct {
		key string
		val bool
	}{
		{"Agents", entitlements.Agents},
		{"Audit Logging", entitlements.AuditLogging},
		{"Cost Estimation", entitlements.CostEstimation},
		{"Operations", entitlements.Operations},
		{"Private Module Registry", entitlements.PrivateModuleRegistry},
		{"Run Tasks", entitlements.RunTasks},
		{"SSO", entitlements.SSO},
		{"Sentinel", entitlements.Sentinel},
		{"State Storage", entitlements.StateStorage},
		{"Teams", entitlements.Teams},
		{"VCS Integrations", entitlements.VCSIntegrations},
	}
	for _, ep := range entPairs {
		pairs = append(pairs, output.KeyValue{Key: ep.key, Value: strconv.FormatBool(ep.val)})
	}

	output.PrintKeyValue(os.Stdout, pairs)
	return nil
}
