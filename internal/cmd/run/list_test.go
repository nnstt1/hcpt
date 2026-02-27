package run

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
)

type mockRunListService struct {
	workspace  *tfe.Workspace
	runs       []*tfe.Run
	readErr    error
	listErr    error
	run        *tfe.Run
	readRunErr error
	listRunFn  func(opts *tfe.RunListOptions) (*tfe.RunList, error)
}

func (m *mockRunListService) ReadWorkspace(_ context.Context, _ string, _ string) (*tfe.Workspace, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	return m.workspace, nil
}

func (m *mockRunListService) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	return nil, nil
}

func (m *mockRunListService) ListRuns(_ context.Context, _ string, opts *tfe.RunListOptions) (*tfe.RunList, error) {
	if m.listRunFn != nil {
		return m.listRunFn(opts)
	}
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &tfe.RunList{
		Items: m.runs,
	}, nil
}

func (m *mockRunListService) ReadRun(_ context.Context, _ string) (*tfe.Run, error) {
	if m.readRunErr != nil {
		return nil, m.readRunErr
	}
	return m.run, nil
}

func (m *mockRunListService) ReadRunWithApply(_ context.Context, _ string) (*tfe.Run, error) {
	if m.readRunErr != nil {
		return nil, m.readRunErr
	}
	return m.run, nil
}

func TestRunList_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockRunListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		runs: []*tfe.Run{
			{
				ID:         "run-123",
				Status:     tfe.RunApplied,
				Message:    "Apply complete",
				HasChanges: true,
				CreatedAt:  time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
			{
				ID:         "run-456",
				Status:     tfe.RunPlanned,
				Message:    "Queued by user",
				HasChanges: false,
				CreatedAt:  time.Date(2024, 3, 14, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:         "run-789",
				Status:     tfe.RunPlannedAndFinished,
				Message:    "Plan only run",
				PlanOnly:   true,
				HasChanges: false,
				CreatedAt:  time.Date(2024, 3, 13, 8, 0, 0, 0, time.UTC),
			},
		},
	}

	cmd := newCmdRunListWith(func() (runListService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-w", "my-ws"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockRunListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		runs: []*tfe.Run{
			{
				ID:         "run-123",
				Status:     tfe.RunApplied,
				Message:    "Apply complete",
				HasChanges: true,
				CreatedAt:  time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunList(mock, "test-org", "my-ws", "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"ID", "STATUS", "PLAN ONLY", "run-123", "applied", "Apply complete", "false", "true", "2024-03-15 12:00:00"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestRunList_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockRunListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		runs: []*tfe.Run{
			{
				ID:         "run-123",
				Status:     tfe.RunApplied,
				Message:    "Apply complete",
				HasChanges: true,
				CreatedAt:  time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunList(mock, "test-org", "my-ws", "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"id": "run-123"`, `"status": "applied"`, `"plan_only": false`, `"has_changes": true`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestRunList_Pagination(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockRunListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		listRunFn: func(opts *tfe.RunListOptions) (*tfe.RunList, error) {
			page := opts.PageNumber
			if page == 0 {
				page = 1
			}
			switch page {
			case 1:
				return &tfe.RunList{
					Items: []*tfe.Run{
						{ID: "run-1", Status: tfe.RunApplied, Message: "msg1", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
					},
					Pagination: &tfe.Pagination{NextPage: 2, TotalPages: 2},
				}, nil
			case 2:
				return &tfe.RunList{
					Items: []*tfe.Run{
						{ID: "run-2", Status: tfe.RunPlanned, Message: "msg2", CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
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

	err := runRunList(mock, "test-org", "my-ws", "")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"run-1", "run-2"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestRunList_Empty(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockRunListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		runs:      []*tfe.Run{},
	}

	cmd := newCmdRunListWith(func() (runListService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-w", "my-ws"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunList_ListRunsError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockRunListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		listErr:   fmt.Errorf("runs api error"),
	}

	cmd := newCmdRunListWith(func() (runListService, error) {
		return mock, nil
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
	if !strings.Contains(err.Error(), "runs api error") {
		t.Errorf("expected 'runs api error', got: %v", err)
	}
}

func TestRunList_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdRunListWith(func() (runListService, error) {
		return nil, fmt.Errorf("token missing")
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
	if !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected 'token missing' error, got: %v", err)
	}
}

func TestRunList_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunListWith(func() (runListService, error) {
		return &mockRunListService{}, nil
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
		t.Errorf("expected 'organization is required' error, got: %v", err)
	}
}

func TestRunList_NoWorkspace(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdRunListWith(func() (runListService, error) {
		return &mockRunListService{}, nil
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
	if !strings.Contains(err.Error(), "workspace is required") {
		t.Errorf("expected 'workspace is required' error, got: %v", err)
	}
}

func TestRunList_WorkspaceReadError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockRunListService{
		readErr: fmt.Errorf("workspace not found"),
	}

	cmd := newCmdRunListWith(func() (runListService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"-w", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace not found") {
		t.Errorf("expected 'workspace not found' error, got: %v", err)
	}
}

func TestRunList_StatusFilter(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	var capturedStatus string
	mock := &mockRunListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		listRunFn: func(opts *tfe.RunListOptions) (*tfe.RunList, error) {
			capturedStatus = opts.Status
			return &tfe.RunList{
				Items: []*tfe.Run{
					{ID: "run-123", Status: tfe.RunApplied, Message: "done", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
	}

	cmd := newCmdRunListWith(func() (runListService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-w", "my-ws", "--status", "applied"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStatus != "applied" {
		t.Errorf("expected status filter %q, got %q", "applied", capturedStatus)
	}
}

func TestRunList_StatusFilterMultiple(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	var capturedStatus string
	mock := &mockRunListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		listRunFn: func(opts *tfe.RunListOptions) (*tfe.RunList, error) {
			capturedStatus = opts.Status
			return &tfe.RunList{
				Items: []*tfe.Run{
					{ID: "run-123", Status: tfe.RunApplied, Message: "done", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
					{ID: "run-456", Status: tfe.RunErrored, Message: "failed", CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
				},
			}, nil
		},
	}

	cmd := newCmdRunListWith(func() (runListService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-w", "my-ws", "--status", "applied,errored"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStatus != "applied,errored" {
		t.Errorf("expected status filter %q, got %q", "applied,errored", capturedStatus)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a very long message that should be truncated", 20, "this is a very lo..."},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
		}
	}
}
