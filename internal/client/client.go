package client

import (
	"context"
	"fmt"
	"net/url"

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
	address := viper.GetString("address")

	if token == "" {
		hostname := hostnameFromAddress(address)
		token = findTerraformToken(hostname)
	}

	if token == "" {
		return nil, fmt.Errorf("API token is required: set TFE_TOKEN environment variable, 'token' in config file, or run 'terraform login'")
	}

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

// hostnameFromAddress extracts hostname from an address URL.
func hostnameFromAddress(address string) string {
	if address == "" {
		return "app.terraform.io"
	}
	u, err := url.Parse(address)
	if err != nil || u.Host == "" {
		return address
	}
	return u.Hostname()
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
