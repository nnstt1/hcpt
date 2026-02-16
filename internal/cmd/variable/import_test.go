package variable

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"
)

func TestVariableImport_Create(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
		createVar: &tfe.Variable{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	filename := filepath.Join("testdata", "test.tfvars")
	err := runVariableImport(mock, "test-org", "my-ws", filename, tfe.CategoryTerraform, false, false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "Successfully imported") {
		t.Errorf("expected success message, got:\n%s", got)
	}
	if !strings.Contains(got, "Created: 3") {
		t.Errorf("expected 'Created: 3', got:\n%s", got)
	}
}

func TestVariableImport_Update(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{
			{
				ID:       "var-123",
				Key:      "region",
				Value:    "ap-northeast-1",
				Category: tfe.CategoryTerraform,
			},
		},
		updateVar: &tfe.Variable{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	filename := filepath.Join("testdata", "test.tfvars.json")
	// overwrite=true で自動上書き
	err := runVariableImport(mock, "test-org", "my-ws", filename, tfe.CategoryTerraform, false, true, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "Successfully imported") {
		t.Errorf("expected success message, got:\n%s", got)
	}
	if !strings.Contains(got, "Updated: 1") {
		t.Errorf("expected 'Updated: 1', got:\n%s", got)
	}
	if !strings.Contains(got, "Created: 1") {
		t.Errorf("expected 'Created: 1', got:\n%s", got)
	}
}

func TestVariableImport_DryRun(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	filename := filepath.Join("testdata", "test.tfvars")
	err := runVariableImport(mock, "test-org", "my-ws", filename, tfe.CategoryTerraform, false, false, true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "[DRY RUN]") {
		t.Errorf("expected '[DRY RUN]' message, got:\n%s", got)
	}
	if !strings.Contains(got, "Would create variable") {
		t.Errorf("expected 'Would create variable' message, got:\n%s", got)
	}
}

func TestVariableImport_FileNotFound(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
	}

	err := runVariableImport(mock, "test-org", "my-ws", "nonexistent.tfvars", tfe.CategoryTerraform, false, false, false)

	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
	}
}

func TestVariableImport_WorkspaceNotFound(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		readErr: fmt.Errorf("workspace not found"),
	}

	filename := filepath.Join("testdata", "test.tfvars")
	err := runVariableImport(mock, "test-org", "my-ws", filename, tfe.CategoryTerraform, false, false, false)

	if err == nil {
		t.Fatal("expected error for workspace not found, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read workspace") {
		t.Errorf("expected 'failed to read workspace' error, got: %v", err)
	}
}

func TestVariableImport_ListVariablesError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		listErr:   fmt.Errorf("API error"),
	}

	filename := filepath.Join("testdata", "test.tfvars")
	err := runVariableImport(mock, "test-org", "my-ws", filename, tfe.CategoryTerraform, false, false, false)

	if err == nil {
		t.Fatal("expected error for list variables failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list variables") {
		t.Errorf("expected 'failed to list variables' error, got: %v", err)
	}
}

func TestVariableImport_CreateError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
		createErr: fmt.Errorf("API error"),
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	filename := filepath.Join("testdata", "test.tfvars")
	err := runVariableImport(mock, "test-org", "my-ws", filename, tfe.CategoryTerraform, false, false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err == nil {
		t.Fatal("expected error for create variable failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create variable") {
		t.Errorf("expected 'failed to create variable' error, got: %v", err)
	}
}

func TestVariableImport_EmptyFile(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	// Create empty test file
	tmpFile := filepath.Join(os.TempDir(), "empty.tfvars.json")
	err := os.WriteFile(tmpFile, []byte("{}"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runVariableImport(mock, "test-org", "my-ws", tmpFile, tfe.CategoryTerraform, false, false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "No variables found in file") {
		t.Errorf("expected 'No variables found' message, got:\n%s", got)
	}
}
