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
}

func (m *mockWSService) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
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

func TestWorkspaceShow_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockWSService{
		workspace: &tfe.Workspace{
			Name:             "my-workspace",
			ID:               "ws-abc123",
			Description:      "A test workspace",
			ExecutionMode:    "remote",
			TerraformVersion: "1.5.0",
			Locked:           false,
			AutoApply:        true,
			WorkingDirectory: "infra/",
			ResourceCount:    42,
			CreatedAt:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	cmd := newCmdWorkspaceShowWith(func() (client.WorkspaceService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"my-workspace"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkspaceShow_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockWSService{
		workspace: &tfe.Workspace{
			Name:             "my-workspace",
			ID:               "ws-abc123",
			Description:      "A test workspace",
			ExecutionMode:    "remote",
			TerraformVersion: "1.5.0",
			Locked:           false,
			AutoApply:        true,
			WorkingDirectory: "infra/",
			ResourceCount:    42,
			CreatedAt:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"Name:", "my-workspace", "ID:", "ws-abc123", "Description:", "A test workspace", "42", "infra/"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceShow_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockWSService{
		workspace: &tfe.Workspace{
			Name:             "my-workspace",
			ID:               "ws-abc123",
			Description:      "A test workspace",
			ExecutionMode:    "remote",
			TerraformVersion: "1.5.0",
			ResourceCount:    42,
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"name": "my-workspace"`, `"id": "ws-abc123"`, `"resource_count": 42`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceShow_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdWorkspaceShowWith(func() (client.WorkspaceService, error) {
		return &mockWSService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"my-workspace"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "organization is required") {
		t.Errorf("expected 'organization is required' error, got: %v", err)
	}
}

func TestWorkspaceShow_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdWorkspaceShowWith(func() (client.WorkspaceService, error) {
		return nil, fmt.Errorf("token missing")
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"my-workspace"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected 'token missing' error, got: %v", err)
	}
}

func TestWorkspaceShow_NoArgs(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdWorkspaceShowWith(func() (client.WorkspaceService, error) {
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
}

func TestWorkspaceShow_Error(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockWSService{
		readErr: fmt.Errorf("not found"),
	}

	cmd := newCmdWorkspaceShowWith(func() (client.WorkspaceService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}
