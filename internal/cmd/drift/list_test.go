package drift

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

type mockDriftListService struct {
	items      []client.ExplorerWorkspace
	projects   []*tfe.Project
	listErr    error
	projectErr error
	lastPage   int    // track requested page
	lastFilter string // track requested project filter
}

func (m *mockDriftListService) ListExplorerWorkspaces(_ context.Context, _ string, opts client.ExplorerListOptions) (*client.ExplorerWorkspaceList, error) {
	m.lastPage = opts.Page
	m.lastFilter = opts.ProjectName
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &client.ExplorerWorkspaceList{
		Items:      m.items,
		TotalPages: 1,
		NextPage:   0,
	}, nil
}

func (m *mockDriftListService) ListProjects(_ context.Context, _ string, opts *tfe.ProjectListOptions) (*tfe.ProjectList, error) {
	if m.projectErr != nil {
		return nil, m.projectErr
	}
	var items []*tfe.Project
	for _, p := range m.projects {
		if opts.Name == "" || p.Name == opts.Name {
			items = append(items, p)
		}
	}
	return &tfe.ProjectList{Items: items}, nil
}

func TestDriftList_DriftedOnly(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftListService{
		items: []client.ExplorerWorkspace{
			{WorkspaceName: "prod-vpc", Drifted: true, ResourcesDrifted: 3},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", false, "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "prod-vpc") {
		t.Errorf("expected 'prod-vpc' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "true") {
		t.Errorf("expected 'true' in output, got:\n%s", got)
	}
}

func TestDriftList_All(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftListService{
		items: []client.ExplorerWorkspace{
			{WorkspaceName: "prod-vpc", Drifted: true, ResourcesDrifted: 3},
			{WorkspaceName: "staging", Drifted: false, ResourcesDrifted: 0},
			{WorkspaceName: "dev", Drifted: false, ResourcesDrifted: 0},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", true, "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"WORKSPACE", "DRIFTED", "prod-vpc", "staging", "dev", "true", "false"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestDriftList_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockDriftListService{
		items: []client.ExplorerWorkspace{
			{WorkspaceName: "prod-vpc", Drifted: true, ResourcesDrifted: 3, ResourcesUndrifted: 12},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", false, "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"workspace": "prod-vpc"`, `"drifted": true`, `"resources_drifted": 3`, `"resources_undrifted": 12`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestDriftList_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdDriftListWith(func() (driftListService, error) {
		return &mockDriftListService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "organization is required") {
		t.Errorf("expected 'organization is required' error, got: %v", err)
	}
}

func TestDriftList_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdDriftListWith(func() (driftListService, error) {
		return nil, fmt.Errorf("token missing")
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected 'token missing' error, got: %v", err)
	}
}

func TestDriftList_ExplorerError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockDriftListService{
		listErr: fmt.Errorf("explorer API error"),
	}

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", true, "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "explorer API error") {
		t.Errorf("expected 'explorer API error', got: %v", err)
	}
}

func TestDriftList_EmptyResult(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftListService{
		items: []client.ExplorerWorkspace{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", false, "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Should still have headers
	if !strings.Contains(got, "WORKSPACE") {
		t.Errorf("expected header in output, got:\n%s", got)
	}
}

func TestDriftList_ProjectFilter(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftListService{
		projects: []*tfe.Project{{Name: "my-project", ID: "prj-123"}},
		items: []client.ExplorerWorkspace{
			{WorkspaceName: "prod-vpc", ProjectName: "my-project", Drifted: true, ResourcesDrifted: 3},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", false, "my-project")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastFilter != "my-project" {
		t.Errorf("expected explorer query filtered by project 'my-project', got %q", mock.lastFilter)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "prod-vpc") {
		t.Errorf("expected 'prod-vpc' in output, got:\n%s", got)
	}
}

func TestDriftList_ProjectNotFound(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockDriftListService{
		projects: []*tfe.Project{{Name: "other-project", ID: "prj-999"}},
	}

	err := runDriftList(mock, "test-org", false, "missing-project")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `project "missing-project" not found`) {
		t.Errorf("expected 'project not found' error, got: %v", err)
	}
}

// mockDriftListServicePaginated supports multi-page responses.
type mockDriftListServicePaginated struct {
	pages map[int][]client.ExplorerWorkspace
}

func (m *mockDriftListServicePaginated) ListExplorerWorkspaces(_ context.Context, _ string, opts client.ExplorerListOptions) (*client.ExplorerWorkspaceList, error) {
	page := opts.Page
	items := m.pages[page]
	totalPages := len(m.pages)
	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = 0
	}
	return &client.ExplorerWorkspaceList{
		Items:      items,
		TotalPages: totalPages,
		NextPage:   nextPage,
	}, nil
}

func (m *mockDriftListServicePaginated) ListProjects(_ context.Context, _ string, _ *tfe.ProjectListOptions) (*tfe.ProjectList, error) {
	return &tfe.ProjectList{}, nil
}

func TestDriftList_Pagination(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftListServicePaginated{
		pages: map[int][]client.ExplorerWorkspace{
			1: {{WorkspaceName: "ws-page1", Drifted: true, ResourcesDrifted: 1}},
			2: {{WorkspaceName: "ws-page2", Drifted: true, ResourcesDrifted: 2}},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", false, "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "ws-page1") {
		t.Errorf("expected 'ws-page1' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "ws-page2") {
		t.Errorf("expected 'ws-page2' in output, got:\n%s", got)
	}
}
