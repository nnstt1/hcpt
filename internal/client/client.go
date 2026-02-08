package client

import (
	"context"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"
)

// OrganizationService provides operations on HCP Terraform organizations.
type OrganizationService interface {
	ListOrganizations(ctx context.Context, opts *tfe.OrganizationListOptions) (*tfe.OrganizationList, error)
}

// WorkspaceService provides operations on HCP Terraform workspaces.
type WorkspaceService interface {
	ListWorkspaces(ctx context.Context, org string, opts *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error)
	ReadWorkspace(ctx context.Context, org string, name string) (*tfe.Workspace, error)
}

// RunService provides operations on HCP Terraform runs.
type RunService interface {
	ListRuns(ctx context.Context, workspaceID string, opts *tfe.RunListOptions) (*tfe.RunList, error)
}

// ClientWrapper wraps the go-tfe client and implements all service interfaces.
type ClientWrapper struct {
	client *tfe.Client
}

// NewClientWrapper creates a new ClientWrapper using configuration from Viper.
func NewClientWrapper() (*ClientWrapper, error) {
	token := viper.GetString("token")
	if token == "" {
		return nil, fmt.Errorf("API token is required: set TFE_TOKEN environment variable or 'token' in config file")
	}

	address := viper.GetString("address")

	config := &tfe.Config{
		Token:   token,
		Address: address,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HCP Terraform client: %w", err)
	}

	return &ClientWrapper{client: client}, nil
}

func (c *ClientWrapper) ListOrganizations(ctx context.Context, opts *tfe.OrganizationListOptions) (*tfe.OrganizationList, error) {
	return c.client.Organizations.List(ctx, opts)
}

func (c *ClientWrapper) ListWorkspaces(ctx context.Context, org string, opts *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	return c.client.Workspaces.List(ctx, org, opts)
}

func (c *ClientWrapper) ReadWorkspace(ctx context.Context, org string, name string) (*tfe.Workspace, error) {
	return c.client.Workspaces.Read(ctx, org, name)
}

func (c *ClientWrapper) ListRuns(ctx context.Context, workspaceID string, opts *tfe.RunListOptions) (*tfe.RunList, error) {
	return c.client.Runs.List(ctx, workspaceID, opts)
}
