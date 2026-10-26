package project

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

type mockProjectShowService struct {
	projects   []*client.Project
	workspaces []*client.Workspace
	projErr    error
	wsErr      error
	listFn     func(opts *client.ProjectListOptions) (*client.ProjectList, error)
}

func (m *mockProjectShowService) ListProjects(_ context.Context, _ string, opts *client.ProjectListOptions) (*client.ProjectList, error) {
	if m.listFn != nil {
		return m.listFn(opts)
	}
	if m.projErr != nil {
		return nil, m.projErr
	}
	return &client.ProjectList{
		Items: m.projects,
	}, nil
}

func (m *mockProjectShowService) ListWorkspaces(_ context.Context, _ string, _ *client.WorkspaceListOptions) (*client.WorkspaceList, error) {
	if m.wsErr != nil {
		return nil, m.wsErr
	}
	return &client.WorkspaceList{
		Items: m.workspaces,
	}, nil
}

func (m *mockProjectShowService) ReadWorkspace(_ context.Context, _ string, _ string) (*client.Workspace, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestProjectShow_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockProjectShowService{
		projects: []*client.Project{
			{
				Name:                 "my-project",
				ID:                   "prj-abc123",
				Description:          "My test project",
				DefaultExecutionMode: "remote",
			},
		},
		workspaces: []*client.Workspace{
			{Name: "ws-prod", ID: "ws-111"},
			{Name: "ws-dev", ID: "ws-222"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldStderr := os.Stderr
	_, wErr, _ := os.Pipe()
	os.Stderr = wErr

	err := runProjectShow(mock, "test-org", "my-project")

	_ = w.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"Name", "my-project", "ID", "prj-abc123", "Default Execution Mode", "remote", "Workspaces", "2", "ws-prod", "ws-dev"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestProjectShow_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockProjectShowService{
		projects: []*client.Project{
			{
				Name:                 "my-project",
				ID:                   "prj-abc123",
				Description:          "My test project",
				DefaultExecutionMode: "remote",
			},
		},
		workspaces: []*client.Workspace{
			{Name: "ws-prod", ID: "ws-111"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjectShow(mock, "test-org", "my-project")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"name": "my-project"`, `"id": "prj-abc123"`, `"default_execution_mode": "remote"`, `"workspaces"`, `"name": "ws-prod"`, `"id": "ws-111"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestProjectShow_NoWorkspaces(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockProjectShowService{
		projects: []*client.Project{
			{
				Name: "empty-project",
				ID:   "prj-empty",
			},
		},
		workspaces: []*client.Workspace{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjectShow(mock, "test-org", "empty-project")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "Workspaces") {
		t.Errorf("expected 'Workspaces' in output, got:\n%s", got)
	}
	if strings.Contains(got, "NAME") {
		t.Errorf("expected no workspace table header when no workspaces, got:\n%s", got)
	}
}

func TestProjectShow_NotFound(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockProjectShowService{
		projects: []*client.Project{},
	}

	err := runProjectShow(mock, "test-org", "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProjectShow_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdProjectShowWith(func() (projectShowService, error) {
		return &mockProjectShowService{}, nil
	})
	cmd.SetArgs([]string{"my-project"})

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

func TestProjectShow_WorkspaceListError(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockProjectShowService{
		projects: []*client.Project{
			{
				Name: "my-project",
				ID:   "prj-abc123",
			},
		},
		wsErr: fmt.Errorf("workspace api error"),
	}

	err := runProjectShow(mock, "test-org", "my-project")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace api error") {
		t.Errorf("expected 'workspace api error' error, got: %v", err)
	}
}

func TestProjectShow_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdProjectShowWith(func() (projectShowService, error) {
		return nil, fmt.Errorf("token missing")
	})
	cmd.SetArgs([]string{"my-project"})

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
