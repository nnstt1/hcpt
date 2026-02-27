package run

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"
)

type mockRunLogsService struct {
	workspace    *tfe.Workspace
	workspaceErr error
	runList      *tfe.RunList
	runListErr   error
	run          *tfe.Run
	runErr       error
	logs         string
	logsErr      error
}

func (m *mockRunLogsService) ListRuns(_ context.Context, _ string, _ *tfe.RunListOptions) (*tfe.RunList, error) {
	if m.runListErr != nil {
		return nil, m.runListErr
	}
	return m.runList, nil
}

func (m *mockRunLogsService) ReadRun(_ context.Context, _ string) (*tfe.Run, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.run, nil
}

func (m *mockRunLogsService) ReadRunWithApply(_ context.Context, _ string) (*tfe.Run, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.run, nil
}

func (m *mockRunLogsService) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	return nil, nil
}

func (m *mockRunLogsService) ReadWorkspace(_ context.Context, _, _ string) (*tfe.Workspace, error) {
	if m.workspaceErr != nil {
		return nil, m.workspaceErr
	}
	return m.workspace, nil
}

func (m *mockRunLogsService) ReadApplyLogs(_ context.Context, _ string) (io.Reader, error) {
	if m.logsErr != nil {
		return nil, m.logsErr
	}
	return strings.NewReader(m.logs), nil
}

func TestRunLogs_Basic(t *testing.T) {
	viper.Reset()

	logContent := "2024-01-01T00:00:00.000Z [INFO]  provider.terraform-provider-aws: Initializing\n2024-01-01T00:00:01.000Z [INFO]  Apply complete! Resources: 2 added."

	mock := &mockRunLogsService{
		run: &tfe.Run{
			ID:     "run-abc123",
			Status: tfe.RunApplied,
			Apply: &tfe.Apply{
				ID: "apply-xyz789",
			},
		},
		logs: logContent,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunLogs(mock, "run-abc123", "", "", false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"Initializing", "Apply complete!"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestRunLogs_ErrorOnly(t *testing.T) {
	viper.Reset()

	logContent := `{"@level":"info","@message":"Refreshing state"}
{"@level":"error","@message":"Error: insufficient permissions"}
{"@level":"info","@message":"Apply cancelled"}`

	mock := &mockRunLogsService{
		run: &tfe.Run{
			ID:     "run-abc123",
			Status: tfe.RunErrored,
			Apply: &tfe.Apply{
				ID: "apply-xyz789",
			},
		},
		logs: logContent,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunLogs(mock, "run-abc123", "", "", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "insufficient permissions") {
		t.Errorf("expected error line in output, got:\n%s", got)
	}
	if strings.Contains(got, "Refreshing state") {
		t.Errorf("expected info line to be filtered out, got:\n%s", got)
	}
	if strings.Contains(got, "Apply cancelled") {
		t.Errorf("expected info line to be filtered out, got:\n%s", got)
	}
}

func TestRunLogs_WithWorkspace(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	logLine := "Apply complete! Resources: 1 added."

	mock := &mockRunLogsService{
		workspace: &tfe.Workspace{
			ID:   "ws-123",
			Name: "my-ws",
		},
		runList: &tfe.RunList{
			Items: []*tfe.Run{
				{
					ID:     "run-latest",
					Status: tfe.RunApplied,
					Apply: &tfe.Apply{
						ID: "apply-latest",
					},
				},
			},
		},
		run: &tfe.Run{
			ID:     "run-latest",
			Status: tfe.RunApplied,
			Apply: &tfe.Apply{
				ID: "apply-latest",
			},
		},
		logs: logLine,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunLogs(mock, "", "test-org", "my-ws", false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "Apply complete!") {
		t.Errorf("expected log output, got:\n%s", got)
	}
}

func TestRunLogs_NoApply(t *testing.T) {
	viper.Reset()

	mock := &mockRunLogsService{
		run: &tfe.Run{
			ID:     "run-planning",
			Status: tfe.RunPlanning,
			Apply:  nil, // No apply yet
		},
	}

	err := runRunLogs(mock, "run-planning", "", "", false)
	if err == nil {
		t.Fatal("expected error when apply is nil")
	}
	if !strings.Contains(err.Error(), "does not have an apply") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunLogs_NoArgs(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunLogsWith(func() (runLogsService, error) {
		return &mockRunLogsService{}, nil
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
	if !strings.Contains(err.Error(), "either run-id or --workspace/-w is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunLogs_ClientError(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunLogsWith(func() (runLogsService, error) {
		return nil, fmt.Errorf("token missing")
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"run-abc123"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected 'token missing' error, got: %v", err)
	}
}

func TestRunLogs_WithWorkspace_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunLogsWith(func() (runLogsService, error) {
		return &mockRunLogsService{}, nil
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
	if !strings.Contains(err.Error(), "organization is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunLogs_WithWorkspace_NoRuns(t *testing.T) {
	viper.Reset()

	mock := &mockRunLogsService{
		workspace: &tfe.Workspace{ID: "ws-123", Name: "empty-ws"},
		runList:   &tfe.RunList{Items: []*tfe.Run{}},
	}

	err := runRunLogs(mock, "", "test-org", "empty-ws", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no runs found for workspace") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIsErrorLine(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{`{"@level":"error","@message":"Something failed"}`, true},
		{`{"@level":"info","@message":"Apply complete"}`, false},
		{`{"@level":"warn","@message":"Deprecated usage"}`, false},
		{`2024-01-01T00:00:00.000Z [ERROR] something went wrong`, false},
		{`{"@level":"error"}`, true},
	}

	for _, tt := range tests {
		got := isErrorLine(tt.line)
		if got != tt.expected {
			t.Errorf("isErrorLine(%q) = %v, want %v", tt.line, got, tt.expected)
		}
	}
}
