package variable

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"
)

func TestVariableDelete_Success(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{
			{
				ID:       "var-123",
				Key:      "MY_VAR",
				Category: tfe.CategoryTerraform,
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runVariableDelete(mock, "test-org", "my-ws", "MY_VAR", tfe.CategoryTerraform)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, `Deleted variable "MY_VAR"`) {
		t.Errorf("expected 'Deleted variable' message, got:\n%s", got)
	}
}

func TestVariableDelete_NotFound(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
	}

	err := runVariableDelete(mock, "test-org", "my-ws", "NONEXISTENT", tfe.CategoryTerraform)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestVariableDelete_DeleteError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{
			{
				ID:       "var-123",
				Key:      "MY_VAR",
				Category: tfe.CategoryTerraform,
			},
		},
		deleteErr: fmt.Errorf("delete api error"),
	}

	err := runVariableDelete(mock, "test-org", "my-ws", "MY_VAR", tfe.CategoryTerraform)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete api error") {
		t.Errorf("expected 'delete api error', got: %v", err)
	}
}

func TestVariableDelete_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdVariableDeleteWith(func() (variableDeleteService, error) {
		return nil, fmt.Errorf("token missing")
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"KEY", "-w", "my-ws"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected 'token missing' error, got: %v", err)
	}
}

func TestVariableDelete_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdVariableDeleteWith(func() (variableDeleteService, error) {
		return &mockVariableListService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"KEY", "-w", "my-ws"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "organization is required") {
		t.Errorf("expected 'organization is required' error, got: %v", err)
	}
}

func TestVariableDelete_NoWorkspace(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdVariableDeleteWith(func() (variableDeleteService, error) {
		return &mockVariableListService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"KEY"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace is required") {
		t.Errorf("expected 'workspace is required' error, got: %v", err)
	}
}

func TestVariableDelete_NoArgs(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdVariableDeleteWith(func() (variableDeleteService, error) {
		return &mockVariableListService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"-w", "my-ws"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestVariableDelete_WorkspaceReadError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		readErr: fmt.Errorf("workspace not found"),
	}

	err := runVariableDelete(mock, "test-org", "my-ws", "KEY", tfe.CategoryTerraform)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace not found") {
		t.Errorf("expected 'workspace not found' error, got: %v", err)
	}
}
