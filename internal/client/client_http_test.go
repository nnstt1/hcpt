package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v83/github"
	"github.com/spf13/viper"
)

// newTestClientWrapper creates a ClientWrapper pointing to the given test server URL.
func newTestClientWrapper(serverURL string) *ClientWrapper {
	return &ClientWrapper{
		address: serverURL,
		token:   "test-token",
	}
}

// --- hostnameFromAddress ---

func TestHostnameFromAddress_NoScheme(t *testing.T) {
	// URL with no scheme → url.Parse succeeds but u.Host == ""
	got := hostnameFromAddress("localhost")
	if got != "localhost" {
		t.Errorf("expected 'localhost', got %q", got)
	}
}

// --- ReadCurrentAssessment ---

func TestReadCurrentAssessment_Success(t *testing.T) {
	body := `{
		"data": {
			"id": "asmnt-xyz",
			"type": "assessment-results",
			"attributes": {
				"drifted": true,
				"succeeded": true,
				"resources-drifted": 2,
				"resources-undrifted": 8,
				"created-at": "2025-03-01T00:00:00Z"
			}
		}
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	result, err := cw.ReadCurrentAssessment(context.Background(), "ws-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != "asmnt-xyz" {
		t.Errorf("expected ID 'asmnt-xyz', got %q", result.ID)
	}
	if !result.Drifted {
		t.Error("expected Drifted to be true")
	}
	if result.ResourcesDrifted != 2 {
		t.Errorf("expected ResourcesDrifted=2, got %d", result.ResourcesDrifted)
	}
}

func TestReadCurrentAssessment_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	result, err := cw.ReadCurrentAssessment(context.Background(), "ws-abc123")
	if err != nil {
		t.Fatalf("expected nil error for 404, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for 404, got %+v", result)
	}
}

func TestReadCurrentAssessment_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ReadCurrentAssessment(context.Background(), "ws-abc123")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 error, got: %v", err)
	}
}

func TestReadCurrentAssessment_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ReadCurrentAssessment(context.Background(), "ws-abc123")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestReadCurrentAssessment_RateLimitContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	responded := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		close(responded) // Signal that 429 was sent
	}))
	defer ts.Close()

	// Cancel context after the 429 response is sent but before the retry sleep ends
	go func() {
		<-responded
		cancel()
	}()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ReadCurrentAssessment(ctx, "ws-abc123")
	if err == nil {
		t.Fatal("expected error when context is cancelled during rate limit retry, got nil")
	}
	// Accept context.Canceled or a transport error wrapping context cancellation
	if err != context.Canceled && !strings.Contains(err.Error(), "context") {
		t.Errorf("expected context-related error, got: %v", err)
	}
}

// --- ReadAssessmentDriftDetails ---

func TestReadAssessmentDriftDetails_Success(t *testing.T) {
	body := `{
		"resource_drift": [
			{
				"address": "aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"change": {"actions": ["update"]}
			}
		]
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/json-output") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	resources, err := cw.ReadAssessmentDriftDetails(context.Background(), "asmnt-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].Address != "aws_instance.web" {
		t.Errorf("expected address 'aws_instance.web', got %q", resources[0].Address)
	}
}

func TestReadAssessmentDriftDetails_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ReadAssessmentDriftDetails(context.Background(), "asmnt-abc123")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("expected HTTP 403 error, got: %v", err)
	}
}

func TestReadAssessmentDriftDetails_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ReadAssessmentDriftDetails(context.Background(), "asmnt-abc123")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// --- ReadSubscription ---

func TestReadSubscription_Success(t *testing.T) {
	body := `{
		"data": {
			"id": "sub-123",
			"type": "subscriptions",
			"attributes": {"is-active": true, "is-public-free-tier": false},
			"relationships": {
				"feature-set": {"data": {"id": "fs-plus", "type": "feature-sets"}}
			}
		},
		"included": [
			{
				"id": "fs-plus",
				"type": "feature-sets",
				"attributes": {"name": "Plus", "is-free-tier": false, "is-current": true}
			}
		]
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/subscription") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	info, err := cw.ReadSubscription(context.Background(), "my-org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PlanName != "Plus" {
		t.Errorf("expected plan name 'Plus', got %q", info.PlanName)
	}
	if !info.IsActive {
		t.Error("expected IsActive to be true")
	}
	if info.IsFreeTier {
		t.Error("expected IsFreeTier to be false")
	}
}

func TestReadSubscription_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ReadSubscription(context.Background(), "my-org")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf("expected HTTP 401 error, got: %v", err)
	}
}

func TestReadSubscription_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ReadSubscription(context.Background(), "my-org")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// --- ListExplorerWorkspaces ---

func TestListExplorerWorkspaces_Success(t *testing.T) {
	body := `{
		"data": [
			{
				"type": "workspace-summaries",
				"attributes": {
					"workspace-name": "prod",
					"external-id": "ws-abc",
					"workspace-terraform-version": "1.6.0",
					"current-run-status": "applied",
					"project-name": "default",
					"workspace-updated-at": "2025-01-01T00:00:00Z",
					"drifted": false,
					"resources-drifted": 0,
					"resources-undrifted": 10
				}
			}
		],
		"meta": {
			"pagination": {"total-pages": 1, "current-page": 1, "next-page": 0}
		}
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/explorer") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	result, err := cw.ListExplorerWorkspaces(context.Background(), "my-org", ExplorerListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].WorkspaceName != "prod" {
		t.Errorf("expected workspace name 'prod', got %q", result.Items[0].WorkspaceName)
	}
	if result.TotalPages != 1 {
		t.Errorf("expected TotalPages=1, got %d", result.TotalPages)
	}
}

func TestListExplorerWorkspaces_DriftedOnly(t *testing.T) {
	var gotParams url.Values
	body := `{"data": [], "meta": {"pagination": {"total-pages": 0, "current-page": 1, "next-page": 0}}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ListExplorerWorkspaces(context.Background(), "my-org", ExplorerListOptions{
		DriftedOnly: true,
		Page:        2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Get("filter[0][drifted][is][0]") != "true" {
		t.Errorf("expected drifted filter, got params: %v", gotParams)
	}
	if gotParams.Get("page[number]") != "2" {
		t.Errorf("expected page[number]=2, got %q", gotParams.Get("page[number]"))
	}
}

func TestListExplorerWorkspaces_SearchFilter(t *testing.T) {
	var gotParams url.Values
	body := `{"data": [], "meta": {"pagination": {"total-pages": 0, "current-page": 1, "next-page": 0}}}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParams = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, body)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ListExplorerWorkspaces(context.Background(), "my-org", ExplorerListOptions{
		Search: "prod",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Get("filter[0][workspace-name][contains][0]") != "prod" {
		t.Errorf("expected search filter, got params: %v", gotParams)
	}
}

func TestListExplorerWorkspaces_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ListExplorerWorkspaces(context.Background(), "my-org", ExplorerListOptions{})
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("expected HTTP 403 error, got: %v", err)
	}
}

func TestListExplorerWorkspaces_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer ts.Close()

	cw := newTestClientWrapper(ts.URL)
	_, err := cw.ListExplorerWorkspaces(context.Background(), "my-org", ExplorerListOptions{})
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// --- resolveGitHubToken ---

func TestResolveGitHubToken_EnvVar(t *testing.T) {
	// gh CLI likely not available or not authenticated in test env,
	// so we test the viper env var fallback.
	viper.Reset()
	viper.Set("GITHUB_TOKEN", "env-github-token")
	defer viper.Reset()

	token := resolveGitHubToken()
	// Token may come from gh CLI if installed; if not, should be env-github-token
	if token == "" {
		t.Error("expected non-empty token from GITHUB_TOKEN")
	}
}

func TestResolveGitHubToken_ConfigFile(t *testing.T) {
	viper.Reset()
	viper.Set("github-token", "config-github-token")
	defer viper.Reset()

	token := resolveGitHubToken()
	// Token may come from gh CLI if installed; if not, should be config-github-token
	if token == "" {
		t.Error("expected non-empty token from github-token config")
	}
}

// --- NewGitHubClientWrapper ---

func TestNewGitHubClientWrapper_NoToken(t *testing.T) {
	viper.Reset()
	// Ensure gh CLI won't return a token by unsetting all sources
	t.Setenv("PATH", "")
	t.Setenv("GITHUB_TOKEN", "")

	_, err := NewGitHubClientWrapper()
	if err == nil {
		t.Fatal("expected error when no GitHub token, got nil")
	}
	if !strings.Contains(err.Error(), "GitHub token is required") {
		t.Errorf("expected 'GitHub token is required' error, got: %v", err)
	}
}

func TestNewGitHubClientWrapper_WithToken(t *testing.T) {
	viper.Reset()
	t.Setenv("PATH", "")
	viper.Set("github-token", "test-github-token")
	defer viper.Reset()

	gcw, err := NewGitHubClientWrapper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gcw == nil {
		t.Fatal("expected non-nil GitHubClientWrapper")
	}
}

// --- GetRunIDFromPR ---

func TestGetRunIDFromPR_SingleRun(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/pulls/1"):
			_, _ = fmt.Fprint(w, `{"number": 1, "head": {"sha": "abc123def456"}}`)
		case strings.Contains(r.URL.Path, "/commits/abc123def456/statuses"):
			_, _ = fmt.Fprint(w, `[{"context": "HCP Terraform / my-org / my-workspace", "target_url": "https://app.terraform.io/app/my-org/workspaces/my-workspace/runs/run-abcdef123456"}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL + "/")
	ghClient := github.NewClient(nil)
	ghClient.BaseURL = baseURL
	gcw := &GitHubClientWrapper{client: ghClient}

	runID, err := gcw.GetRunIDFromPR(context.Background(), "owner", "repo", 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runID != "run-abcdef123456" {
		t.Errorf("expected run-abcdef123456, got %q", runID)
	}
}

func TestGetRunIDFromPR_WithWorkspaceFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/pulls/2"):
			_, _ = fmt.Fprint(w, `{"number": 2, "head": {"sha": "deadbeef"}}`)
		case strings.Contains(r.URL.Path, "/commits/deadbeef/statuses"):
			_, _ = fmt.Fprint(w, `[
				{"context": "HCP Terraform / my-org / workspace-a", "target_url": "https://app.terraform.io/app/my-org/workspaces/workspace-a/runs/run-aaaa"},
				{"context": "HCP Terraform / my-org / workspace-b", "target_url": "https://app.terraform.io/app/my-org/workspaces/workspace-b/runs/run-bbbb"}
			]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL + "/")
	ghClient := github.NewClient(nil)
	ghClient.BaseURL = baseURL
	gcw := &GitHubClientWrapper{client: ghClient}

	runID, err := gcw.GetRunIDFromPR(context.Background(), "owner", "repo", 2, "workspace-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runID != "run-bbbb" {
		t.Errorf("expected run-bbbb, got %q", runID)
	}
}

func TestGetRunIDFromPR_MultipleRunsNoFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/pulls/3"):
			_, _ = fmt.Fprint(w, `{"number": 3, "head": {"sha": "cafebabe"}}`)
		case strings.Contains(r.URL.Path, "/commits/cafebabe/statuses"):
			_, _ = fmt.Fprint(w, `[
				{"context": "HCP Terraform / my-org / ws-a", "target_url": "https://app.terraform.io/app/my-org/workspaces/ws-a/runs/run-aaaa"},
				{"context": "HCP Terraform / my-org / ws-b", "target_url": "https://app.terraform.io/app/my-org/workspaces/ws-b/runs/run-bbbb"}
			]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL + "/")
	ghClient := github.NewClient(nil)
	ghClient.BaseURL = baseURL
	gcw := &GitHubClientWrapper{client: ghClient}

	_, err := gcw.GetRunIDFromPR(context.Background(), "owner", "repo", 3, "")
	if err == nil {
		t.Fatal("expected error for multiple runs without workspace filter, got nil")
	}
	if !strings.Contains(err.Error(), "multiple HCP Terraform runs") {
		t.Errorf("expected 'multiple HCP Terraform runs' error, got: %v", err)
	}
}

func TestGetRunIDFromPR_NoRuns(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/pulls/4"):
			_, _ = fmt.Fprint(w, `{"number": 4, "head": {"sha": "00000000"}}`)
		case strings.Contains(r.URL.Path, "/commits/00000000/statuses"):
			_, _ = fmt.Fprint(w, `[]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL + "/")
	ghClient := github.NewClient(nil)
	ghClient.BaseURL = baseURL
	gcw := &GitHubClientWrapper{client: ghClient}

	_, err := gcw.GetRunIDFromPR(context.Background(), "owner", "repo", 4, "")
	if err == nil {
		t.Fatal("expected error when no runs found, got nil")
	}
	if !strings.Contains(err.Error(), "no HCP Terraform run found") {
		t.Errorf("expected 'no HCP Terraform run found' error, got: %v", err)
	}
}

func TestGetRunIDFromPR_WorkspaceNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/pulls/5"):
			_, _ = fmt.Fprint(w, `{"number": 5, "head": {"sha": "11111111"}}`)
		case strings.Contains(r.URL.Path, "/commits/11111111/statuses"):
			_, _ = fmt.Fprint(w, `[{"context": "HCP Terraform / my-org / ws-a", "target_url": "https://app.terraform.io/app/my-org/workspaces/ws-a/runs/run-aaaa"}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	baseURL, _ := url.Parse(ts.URL + "/")
	ghClient := github.NewClient(nil)
	ghClient.BaseURL = baseURL
	gcw := &GitHubClientWrapper{client: ghClient}

	_, err := gcw.GetRunIDFromPR(context.Background(), "owner", "repo", 5, "nonexistent-workspace")
	if err == nil {
		t.Fatal("expected error when workspace not found, got nil")
	}
	if !strings.Contains(err.Error(), "no HCP Terraform run found for workspace") {
		t.Errorf("expected workspace not found error, got: %v", err)
	}
}
