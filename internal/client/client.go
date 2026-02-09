package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"
)

// OrganizationService provides operations on HCP Terraform organizations.
type OrganizationService interface {
	ListOrganizations(ctx context.Context, opts *tfe.OrganizationListOptions) (*tfe.OrganizationList, error)
	ReadOrganization(ctx context.Context, org string) (*tfe.Organization, error)
	ReadEntitlements(ctx context.Context, org string) (*tfe.Entitlements, error)
}

// WorkspaceService provides operations on HCP Terraform workspaces.
type WorkspaceService interface {
	ListWorkspaces(ctx context.Context, org string, opts *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error)
	ReadWorkspace(ctx context.Context, org string, name string) (*tfe.Workspace, error)
}

// RunService provides operations on HCP Terraform runs.
type RunService interface {
	ListRuns(ctx context.Context, workspaceID string, opts *tfe.RunListOptions) (*tfe.RunList, error)
	ReadRun(ctx context.Context, runID string) (*tfe.Run, error)
}

// VariableService provides operations on HCP Terraform workspace variables.
type VariableService interface {
	ListVariables(ctx context.Context, workspaceID string, opts *tfe.VariableListOptions) (*tfe.VariableList, error)
	CreateVariable(ctx context.Context, workspaceID string, opts tfe.VariableCreateOptions) (*tfe.Variable, error)
	UpdateVariable(ctx context.Context, workspaceID string, variableID string, opts tfe.VariableUpdateOptions) (*tfe.Variable, error)
	DeleteVariable(ctx context.Context, workspaceID string, variableID string) error
}

// ProjectService provides operations on HCP Terraform projects.
type ProjectService interface {
	ListProjects(ctx context.Context, org string, opts *tfe.ProjectListOptions) (*tfe.ProjectList, error)
}

// AssessmentResult holds the current assessment (drift detection) result for a workspace.
type AssessmentResult struct {
	Drifted            bool
	Succeeded          bool
	ResourcesDrifted   int
	ResourcesUndrifted int
	CreatedAt          string
}

// AssessmentService provides operations to read workspace assessment results.
type AssessmentService interface {
	ReadCurrentAssessment(ctx context.Context, workspaceID string) (*AssessmentResult, error)
}

// ExplorerWorkspace holds a workspace entry returned by the Explorer API.
type ExplorerWorkspace struct {
	WorkspaceName      string
	Drifted            bool
	ResourcesDrifted   int
	ResourcesUndrifted int
}

// ExplorerWorkspaceList holds the result of an Explorer API query.
type ExplorerWorkspaceList struct {
	Items      []ExplorerWorkspace
	TotalPages int
	NextPage   int
}

// ExplorerService provides operations using the Explorer API.
type ExplorerService interface {
	ListExplorerWorkspaces(ctx context.Context, org string, driftedOnly bool, page int) (*ExplorerWorkspaceList, error)
}

// SubscriptionInfo holds organization subscription/plan information.
type SubscriptionInfo struct {
	PlanName   string
	IsFreeTier bool
	IsActive   bool
}

// SubscriptionService provides operations to read organization subscription info.
type SubscriptionService interface {
	ReadSubscription(ctx context.Context, org string) (*SubscriptionInfo, error)
}

// ClientWrapper wraps the go-tfe client and implements all service interfaces.
type ClientWrapper struct {
	client  *tfe.Client
	address string
	token   string
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

	return &ClientWrapper{client: client, address: address, token: token}, nil
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

func (c *ClientWrapper) ReadOrganization(ctx context.Context, org string) (*tfe.Organization, error) {
	return c.client.Organizations.Read(ctx, org)
}

func (c *ClientWrapper) ReadEntitlements(ctx context.Context, org string) (*tfe.Entitlements, error) {
	return c.client.Organizations.ReadEntitlements(ctx, org)
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

func (c *ClientWrapper) ReadRun(ctx context.Context, runID string) (*tfe.Run, error) {
	return c.client.Runs.Read(ctx, runID)
}

func (c *ClientWrapper) ListVariables(ctx context.Context, workspaceID string, opts *tfe.VariableListOptions) (*tfe.VariableList, error) {
	return c.client.Variables.List(ctx, workspaceID, opts)
}

func (c *ClientWrapper) CreateVariable(ctx context.Context, workspaceID string, opts tfe.VariableCreateOptions) (*tfe.Variable, error) {
	return c.client.Variables.Create(ctx, workspaceID, opts)
}

func (c *ClientWrapper) UpdateVariable(ctx context.Context, workspaceID string, variableID string, opts tfe.VariableUpdateOptions) (*tfe.Variable, error) {
	return c.client.Variables.Update(ctx, workspaceID, variableID, opts)
}

func (c *ClientWrapper) DeleteVariable(ctx context.Context, workspaceID string, variableID string) error {
	return c.client.Variables.Delete(ctx, workspaceID, variableID)
}

func (c *ClientWrapper) ListProjects(ctx context.Context, org string, opts *tfe.ProjectListOptions) (*tfe.ProjectList, error) {
	return c.client.Projects.List(ctx, org, opts)
}

// ListExplorerWorkspaces queries the Explorer API for workspace drift data.
// If driftedOnly is true, adds a server-side filter for drifted=true.
func (c *ClientWrapper) ListExplorerWorkspaces(ctx context.Context, org string, driftedOnly bool, page int) (*ExplorerWorkspaceList, error) {
	address := c.address
	if address == "" {
		address = "https://app.terraform.io"
	}

	apiURL := strings.TrimRight(address, "/") + "/api/v2/organizations/" + url.PathEscape(org) + "/explorer"

	params := url.Values{}
	params.Set("type", "workspaces")
	params.Set("page[size]", "100")
	if page > 0 {
		params.Set("page[number]", strconv.Itoa(page))
	}
	if driftedOnly {
		params.Set("filter[0][drifted][is][0]", "true")
	}

	fullURL := apiURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch explorer data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("explorer endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read explorer response: %w", err)
	}

	return parseExplorerWorkspacesResponse(body)
}

// parseExplorerWorkspacesResponse extracts workspace data from the Explorer API JSON response.
func parseExplorerWorkspacesResponse(body []byte) (*ExplorerWorkspaceList, error) {
	var response struct {
		Data []struct {
			Attributes struct {
				WorkspaceName      string `json:"workspace-name"`
				Drifted            bool   `json:"drifted"`
				ResourcesDrifted   int    `json:"resources-drifted"`
				ResourcesUndrifted int    `json:"resources-undrifted"`
			} `json:"attributes"`
		} `json:"data"`
		Meta struct {
			Pagination struct {
				TotalPages  int `json:"total-pages"`
				CurrentPage int `json:"current-page"`
				NextPage    int `json:"next-page"`
			} `json:"pagination"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse explorer response: %w", err)
	}

	items := make([]ExplorerWorkspace, 0, len(response.Data))
	for _, d := range response.Data {
		items = append(items, ExplorerWorkspace{
			WorkspaceName:      d.Attributes.WorkspaceName,
			Drifted:            d.Attributes.Drifted,
			ResourcesDrifted:   d.Attributes.ResourcesDrifted,
			ResourcesUndrifted: d.Attributes.ResourcesUndrifted,
		})
	}

	return &ExplorerWorkspaceList{
		Items:      items,
		TotalPages: response.Meta.Pagination.TotalPages,
		NextPage:   response.Meta.Pagination.NextPage,
	}, nil
}

// ReadCurrentAssessment fetches the current assessment result for a workspace.
// Returns nil, nil if assessment is disabled or has not run (HTTP 404).
// Retries on HTTP 429 (rate limit) with backoff.
func (c *ClientWrapper) ReadCurrentAssessment(ctx context.Context, workspaceID string) (*AssessmentResult, error) {
	address := c.address
	if address == "" {
		address = "https://app.terraform.io"
	}

	apiURL := strings.TrimRight(address, "/") + "/api/v2/workspaces/" + url.PathEscape(workspaceID) + "/current-assessment-result"

	const maxRetries = 3

	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/vnd.api+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch assessment result: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("assessment endpoint rate limited (HTTP 429) after %d retries", maxRetries)
			}
			wait := retryAfterDuration(resp.Header.Get("Retry-After"), attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
				continue
			}
		}

		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("assessment endpoint returned HTTP %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read assessment response: %w", err)
		}

		return parseAssessmentResponse(body)
	}

	return nil, fmt.Errorf("assessment endpoint: unexpected retry loop exit")
}

// retryAfterDuration parses the Retry-After header or falls back to exponential backoff.
func retryAfterDuration(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	// Exponential backoff: 1s, 2s, 4s
	return time.Duration(1<<uint(attempt)) * time.Second
}

// parseAssessmentResponse extracts assessment result from the JSON:API response.
func parseAssessmentResponse(body []byte) (*AssessmentResult, error) {
	var response struct {
		Data struct {
			Attributes struct {
				Drifted            bool   `json:"drifted"`
				Succeeded          bool   `json:"succeeded"`
				ResourcesDrifted   int    `json:"resources-drifted"`
				ResourcesUndrifted int    `json:"resources-undrifted"`
				CreatedAt          string `json:"created-at"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse assessment response: %w", err)
	}

	return &AssessmentResult{
		Drifted:            response.Data.Attributes.Drifted,
		Succeeded:          response.Data.Attributes.Succeeded,
		ResourcesDrifted:   response.Data.Attributes.ResourcesDrifted,
		ResourcesUndrifted: response.Data.Attributes.ResourcesUndrifted,
		CreatedAt:          response.Data.Attributes.CreatedAt,
	}, nil
}

// ReadSubscription fetches subscription info from the organizations API.
func (c *ClientWrapper) ReadSubscription(_ context.Context, org string) (*SubscriptionInfo, error) {
	address := c.address
	if address == "" {
		address = "https://app.terraform.io"
	}

	apiURL := strings.TrimRight(address, "/") + "/api/v2/organizations/" + url.PathEscape(org) + "/subscription"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription endpoint returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription response: %w", err)
	}

	return parseSubscriptionResponse(body)
}

// parseSubscriptionResponse extracts subscription info from the JSON:API response.
func parseSubscriptionResponse(body []byte) (*SubscriptionInfo, error) {
	var response struct {
		Data struct {
			Attributes struct {
				IsActive         bool `json:"is-active"`
				IsPublicFreeTier bool `json:"is-public-free-tier"`
			} `json:"attributes"`
			Relationships struct {
				FeatureSet struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"feature-set"`
			} `json:"relationships"`
		} `json:"data"`
		Included []struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				Name       string `json:"name"`
				IsFreeTier bool   `json:"is-free-tier"`
				IsCurrent  bool   `json:"is-current"`
			} `json:"attributes"`
		} `json:"included"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse subscription response: %w", err)
	}

	info := &SubscriptionInfo{
		IsActive:   response.Data.Attributes.IsActive,
		IsFreeTier: response.Data.Attributes.IsPublicFreeTier,
	}

	// Find the plan name from included feature-set resources
	featureSetID := response.Data.Relationships.FeatureSet.Data.ID
	for _, inc := range response.Included {
		if inc.Type == "feature-sets" && inc.ID == featureSetID {
			info.PlanName = inc.Attributes.Name
			info.IsFreeTier = info.IsFreeTier || inc.Attributes.IsFreeTier
			return info, nil
		}
	}

	// Fallback: use feature-set ID if no included resource matched
	if featureSetID != "" {
		info.PlanName = featureSetID
		return info, nil
	}

	return nil, fmt.Errorf("subscription response did not contain plan information")
}
