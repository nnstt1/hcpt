package workspace

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

type mockExplorerService struct {
	items   []client.ExplorerWorkspace
	listErr error
}

func (m *mockExplorerService) ListExplorerWorkspaces(_ context.Context, _ string, _ client.ExplorerListOptions) (*client.ExplorerWorkspaceList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &client.ExplorerWorkspaceList{
		Items:      m.items,
		TotalPages: 1,
		NextPage:   0,
	}, nil
}

func TestWorkspaceList_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockExplorerService{
		items: []client.ExplorerWorkspace{
			{
				WorkspaceName:    "ws-1",
				WorkspaceID:      "ws-abc123",
				ProjectName:      "default",
				TerraformVersion: "1.5.0",
				CurrentRunStatus: "applied",
				UpdatedAt:        "2024-03-15T12:00:00Z",
			},
		},
	}

	cmd := newCmdWorkspaceListWith(func() (client.ExplorerService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkspaceList_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockExplorerService{
		items: []client.ExplorerWorkspace{
			{
				WorkspaceName:    "ws-1",
				WorkspaceID:      "ws-abc123",
				ProjectName:      "production",
				TerraformVersion: "1.5.0",
				CurrentRunStatus: "applied",
				UpdatedAt:        "2024-03-15T12:00:00Z",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceList(mock, "test-org", "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"NAME", "ws-1", "ws-abc123", "production", "1.5.0", "applied", "2024-03-15T12:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceList_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockExplorerService{
		items: []client.ExplorerWorkspace{
			{
				WorkspaceName:    "ws-1",
				WorkspaceID:      "ws-abc123",
				ProjectName:      "default",
				TerraformVersion: "1.5.0",
				CurrentRunStatus: "applied",
				UpdatedAt:        "2024-03-15T12:00:00Z",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceList(mock, "test-org", "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{
		`"name": "ws-1"`,
		`"id": "ws-abc123"`,
		`"terraform_version": "1.5.0"`,
		`"current_run_status": "applied"`,
		`"project_name": "default"`,
		`"updated_at": "2024-03-15T12:00:00Z"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceList_Empty(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockExplorerService{
		items: []client.ExplorerWorkspace{},
	}

	cmd := newCmdWorkspaceListWith(func() (client.ExplorerService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkspaceList_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdWorkspaceListWith(func() (client.ExplorerService, error) {
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

func TestWorkspaceList_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdWorkspaceListWith(func() (client.ExplorerService, error) {
		return &mockExplorerService{}, nil
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

func TestWorkspaceList_Error(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockExplorerService{
		listErr: fmt.Errorf("api error"),
	}

	cmd := newCmdWorkspaceListWith(func() (client.ExplorerService, error) {
		return mock, nil
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
	if !strings.Contains(err.Error(), "api error") {
		t.Errorf("expected 'api error', got: %v", err)
	}
}

type mockExplorerServicePaginated struct {
	pages map[int][]client.ExplorerWorkspace
}

func (m *mockExplorerServicePaginated) ListExplorerWorkspaces(_ context.Context, _ string, opts client.ExplorerListOptions) (*client.ExplorerWorkspaceList, error) {
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

func TestWorkspaceList_Pagination(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockExplorerServicePaginated{
		pages: map[int][]client.ExplorerWorkspace{
			1: {{WorkspaceName: "ws-page1", WorkspaceID: "ws-1", TerraformVersion: "1.5.0"}},
			2: {{WorkspaceName: "ws-page2", WorkspaceID: "ws-2", TerraformVersion: "1.6.0"}},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceList(mock, "test-org", "")

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
