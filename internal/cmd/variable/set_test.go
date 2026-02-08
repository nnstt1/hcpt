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

func TestVariableSet_Create(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
		createVar: &tfe.Variable{Key: "NEW_VAR"},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runVariableSet(mock, "test-org", "my-ws", "NEW_VAR", "new-value", tfe.CategoryTerraform, false, false, "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, `Created variable "NEW_VAR"`) {
		t.Errorf("expected 'Created variable' message, got:\n%s", got)
	}
}

func TestVariableSet_Update(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{
			{
				ID:       "var-123",
				Key:      "EXISTING_VAR",
				Value:    "old-value",
				Category: tfe.CategoryTerraform,
			},
		},
		updateVar: &tfe.Variable{Key: "EXISTING_VAR"},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runVariableSet(mock, "test-org", "my-ws", "EXISTING_VAR", "new-value", tfe.CategoryTerraform, false, false, "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, `Updated variable "EXISTING_VAR"`) {
		t.Errorf("expected 'Updated variable' message, got:\n%s", got)
	}
}

func TestVariableSet_CreateError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
		createErr: fmt.Errorf("create api error"),
	}

	err := runVariableSet(mock, "test-org", "my-ws", "NEW_VAR", "value", tfe.CategoryTerraform, false, false, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "create api error") {
		t.Errorf("expected 'create api error', got: %v", err)
	}
}

func TestVariableSet_UpdateError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{
			{
				ID:       "var-123",
				Key:      "EXISTING_VAR",
				Category: tfe.CategoryTerraform,
			},
		},
		updateErr: fmt.Errorf("update api error"),
	}

	err := runVariableSet(mock, "test-org", "my-ws", "EXISTING_VAR", "value", tfe.CategoryTerraform, false, false, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "update api error") {
		t.Errorf("expected 'update api error', got: %v", err)
	}
}

func TestVariableSet_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdVariableSetWith(func() (variableSetService, error) {
		return nil, fmt.Errorf("token missing")
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"KEY", "VALUE", "-w", "my-ws"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected 'token missing' error, got: %v", err)
	}
}

func TestVariableSet_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdVariableSetWith(func() (variableSetService, error) {
		return &mockVariableListService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"KEY", "VALUE", "-w", "my-ws"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "organization is required") {
		t.Errorf("expected 'organization is required' error, got: %v", err)
	}
}

func TestVariableSet_NoWorkspace(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdVariableSetWith(func() (variableSetService, error) {
		return &mockVariableListService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"KEY", "VALUE"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace is required") {
		t.Errorf("expected 'workspace is required' error, got: %v", err)
	}
}

func TestVariableSet_NoArgs(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdVariableSetWith(func() (variableSetService, error) {
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

func TestVariableSet_WorkspaceReadError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		readErr: fmt.Errorf("workspace not found"),
	}

	err := runVariableSet(mock, "test-org", "my-ws", "KEY", "VALUE", tfe.CategoryTerraform, false, false, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace not found") {
		t.Errorf("expected 'workspace not found' error, got: %v", err)
	}
}

func TestVariableSet_EnvCategory(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
		createVar: &tfe.Variable{Key: "ENV_VAR"},
	}

	cmd := newCmdVariableSetWith(func() (variableSetService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"ENV_VAR", "value", "-w", "my-ws", "--category", "env"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
