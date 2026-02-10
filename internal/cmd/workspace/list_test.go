package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

type mockWSService struct {
	workspaces []*tfe.Workspace
	workspace  *tfe.Workspace
	listErr    error
	readErr    error
	listFn     func(opts *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error)
}

func (m *mockWSService) ListWorkspaces(_ context.Context, _ string, opts *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	if m.listFn != nil {
		return m.listFn(opts)
	}
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &tfe.WorkspaceList{
		Items: m.workspaces,
	}, nil
}

func (m *mockWSService) ReadWorkspace(_ context.Context, _ string, _ string) (*tfe.Workspace, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	return m.workspace, nil
}

func TestWorkspaceList_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockWSService{
		workspaces: []*tfe.Workspace{
			{
				Name:             "ws-1",
				ID:               "ws-abc123",
				ExecutionMode:    "remote",
				TerraformVersion: "1.5.0",
				Locked:           false,
				AutoApply:        true,
				UpdatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	cmd := newCmdWorkspaceListWith(func() (client.WorkspaceService, error) {
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

	mock := &mockWSService{
		workspaces: []*tfe.Workspace{
			{
				Name:             "ws-1",
				ID:               "ws-abc123",
				ExecutionMode:    "remote",
				TerraformVersion: "1.5.0",
				Locked:           true,
				AutoApply:        false,
				UpdatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
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

	for _, want := range []string{"NAME", "ws-1", "ws-abc123", "remote", "1.5.0", "true", "false", "2024-03-15 12:00:00"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceList_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockWSService{
		workspaces: []*tfe.Workspace{
			{
				Name:             "ws-1",
				ID:               "ws-abc123",
				ExecutionMode:    "remote",
				TerraformVersion: "1.5.0",
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

	for _, want := range []string{`"name": "ws-1"`, `"id": "ws-abc123"`, `"execution_mode": "remote"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceList_Empty(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockWSService{
		workspaces: []*tfe.Workspace{},
	}

	cmd := newCmdWorkspaceListWith(func() (client.WorkspaceService, error) {
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

	cmd := newCmdWorkspaceListWith(func() (client.WorkspaceService, error) {
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

	cmd := newCmdWorkspaceListWith(func() (client.WorkspaceService, error) {
		return &mockWSService{}, nil
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

func TestWorkspaceList_Pagination(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockWSService{
		listFn: func(opts *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
			page := opts.PageNumber
			if page == 0 {
				page = 1
			}
			switch page {
			case 1:
				return &tfe.WorkspaceList{
					Items: []*tfe.Workspace{
						{Name: "ws-1", ID: "ws-1", UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
					},
					Pagination: &tfe.Pagination{NextPage: 2, TotalPages: 2},
				}, nil
			case 2:
				return &tfe.WorkspaceList{
					Items: []*tfe.Workspace{
						{Name: "ws-2", ID: "ws-2", UpdatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
					},
					Pagination: &tfe.Pagination{NextPage: 0, TotalPages: 2},
				}, nil
			default:
				return nil, fmt.Errorf("unexpected page %d", page)
			}
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

	for _, want := range []string{"ws-1", "ws-2"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceList_Error(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockWSService{
		listErr: fmt.Errorf("api error"),
	}

	cmd := newCmdWorkspaceListWith(func() (client.WorkspaceService, error) {
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
