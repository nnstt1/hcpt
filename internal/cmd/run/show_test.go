package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"
)

type mockRunShowService struct {
	run          *tfe.Run
	runErr       error
	workspace    *tfe.Workspace
	workspaceErr error
	runList      *tfe.RunList
	runListErr   error
}

func (m *mockRunShowService) ListRuns(_ context.Context, _ string, _ *tfe.RunListOptions) (*tfe.RunList, error) {
	if m.runListErr != nil {
		return nil, m.runListErr
	}
	return m.runList, nil
}

func (m *mockRunShowService) ReadRun(_ context.Context, _ string) (*tfe.Run, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.run, nil
}

func (m *mockRunShowService) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	return nil, nil
}

func (m *mockRunShowService) ReadWorkspace(_ context.Context, _, _ string) (*tfe.Workspace, error) {
	if m.workspaceErr != nil {
		return nil, m.workspaceErr
	}
	return m.workspace, nil
}

func (m *mockRunShowService) ReadRunWithApply(_ context.Context, _ string) (*tfe.Run, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.run, nil
}

func (m *mockRunShowService) ReadPlanJSONOutput(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("ReadPlanJSONOutput not implemented in mockRunShowService")
}

type mockRunShowServiceExtended struct {
	mockRunShowService
	planJSON    []byte
	planJSONErr error
}

func (m *mockRunShowServiceExtended) ReadPlanJSONOutput(_ context.Context, _ string) ([]byte, error) {
	if m.planJSONErr != nil {
		return nil, m.planJSONErr
	}
	return m.planJSON, nil
}

func TestRunShow_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-abc123",
			Status:           tfe.RunApplied,
			Message:          "Apply complete",
			TerraformVersion: "1.5.0",
			HasChanges:       true,
			IsDestroy:        false,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"run-abc123"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunShow_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-abc123",
			Status:           tfe.RunApplied,
			Message:          "Apply complete",
			TerraformVersion: "1.5.0",
			HasChanges:       true,
			IsDestroy:        false,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			Plan: &tfe.Plan{
				ResourceAdditions:    2,
				ResourceChanges:      1,
				ResourceDestructions: 3,
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "run-abc123", "", "", false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"ID:", "run-abc123", "Status:", "applied", "Message:", "Apply complete", "Terraform Version:", "1.5.0", "Has Changes:", "true", "Plan Changes:", "+2 ~1 -3"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Is Destroy:") {
		t.Errorf("expected 'Is Destroy' to be removed, got:\n%s", got)
	}
}

func TestRunShow_Table_WithTimestamps(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-abc123",
			Status:           tfe.RunApplied,
			Message:          "Apply complete",
			TerraformVersion: "1.5.0",
			HasChanges:       true,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			StatusTimestamps: &tfe.RunStatusTimestamps{
				PlannedAt: time.Date(2024, 3, 15, 12, 3, 0, 0, time.UTC),
				AppliedAt: time.Date(2024, 3, 15, 12, 5, 0, 0, time.UTC),
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "run-abc123", "", "", false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "Planned At:") {
		t.Errorf("expected 'Planned At:' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "Applied At:") {
		t.Errorf("expected 'Applied At:' in output, got:\n%s", got)
	}
}

func TestRunShow_Table_NonTerminalStatus(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-planning",
			Status:           tfe.RunPlanning,
			Message:          "Triggered via UI",
			TerraformVersion: "1.5.0",
			HasChanges:       false,
			IsDestroy:        false,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "run-planning", "", "", false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// In non-terminal status, Has Changes and Plan Changes are displayed as "-"
	for _, want := range []string{"Has Changes:", "Plan Changes:"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
	// Verify that true/false is not displayed
	if strings.Contains(got, "Has Changes:        false") || strings.Contains(got, "Has Changes:        true") {
		t.Errorf("expected Has Changes to be '-' for non-terminal status, got:\n%s", got)
	}
	if strings.Contains(got, "Is Destroy:") {
		t.Errorf("expected 'Is Destroy' to be removed, got:\n%s", got)
	}
}

func TestRunShow_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)

	mock := &mockRunShowService{
		run: &tfe.Run{
			ID:               "run-abc123",
			Status:           tfe.RunApplied,
			Message:          "Apply complete",
			TerraformVersion: "1.5.0",
			HasChanges:       true,
			IsDestroy:        false,
			CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			Plan: &tfe.Plan{
				ResourceAdditions:    1,
				ResourceChanges:      2,
				ResourceDestructions: 0,
			},
			StatusTimestamps: &tfe.RunStatusTimestamps{
				PlannedAt: time.Date(2024, 3, 15, 12, 2, 0, 0, time.UTC),
				AppliedAt: time.Date(2024, 3, 15, 12, 5, 0, 0, time.UTC),
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "run-abc123", "", "", false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"id": "run-abc123"`, `"status": "applied"`, `"has_changes": true`, `"resource_additions": 1`, `"resource_changes": 2`, `"resource_destructions": 0`, `"planned_at":`, `"applied_at":`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
	// Verify that is_destroy is not included (to confirm fields without timestamps are omitted)
	if strings.Contains(got, "is_destroy") {
		t.Errorf("expected 'is_destroy' to be removed from JSON, got:\n%s", got)
	}
}

func TestRunShow_ClientError(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunShowWith(func() (runShowService, error) {
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

func TestRunShow_ReadError(t *testing.T) {
	viper.Reset()

	mock := &mockRunShowService{
		runErr: fmt.Errorf("run not found"),
	}

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"run-nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "run not found") {
		t.Errorf("expected 'run not found' error, got: %v", err)
	}
}

func TestRunShow_NoArgs(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return &mockRunShowService{}, nil
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
	if !strings.Contains(err.Error(), "either run-id, --pr, or --workspace/-w is required") {
		t.Errorf("expected 'either run-id, --pr, or --workspace/-w is required' error, got: %v", err)
	}
}

func TestRunShow_WithWorkspace_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	latestRun := &tfe.Run{
		ID:               "run-latest",
		Status:           tfe.RunApplied,
		Message:          "Latest run",
		TerraformVersion: "1.5.0",
		HasChanges:       true,
		IsDestroy:        false,
		CreatedAt:        time.Date(2024, 3, 16, 10, 0, 0, 0, time.UTC),
		Plan: &tfe.Plan{
			ResourceAdditions:    1,
			ResourceChanges:      0,
			ResourceDestructions: 2,
		},
	}

	mock := &mockRunShowService{
		workspace: &tfe.Workspace{
			ID:   "ws-123",
			Name: "production",
		},
		run: latestRun,
		runList: &tfe.RunList{
			Items: []*tfe.Run{latestRun},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "", "test-org", "production", false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"ID:", "run-latest", "Status:", "applied", "Latest run"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestRunShow_WithWorkspace_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	latestRun := &tfe.Run{
		ID:               "run-latest",
		Status:           tfe.RunApplied,
		Message:          "Latest run",
		TerraformVersion: "1.5.0",
		HasChanges:       true,
		IsDestroy:        false,
		CreatedAt:        time.Date(2024, 3, 16, 10, 0, 0, 0, time.UTC),
	}

	mock := &mockRunShowService{
		workspace: &tfe.Workspace{
			ID:   "ws-123",
			Name: "production",
		},
		run: latestRun,
		runList: &tfe.RunList{
			Items: []*tfe.Run{latestRun},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShow(mock, "", "test-org", "production", false, false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"id": "run-latest"`, `"status": "applied"`, `"message": "Latest run"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestRunShow_WithWorkspace_NoRuns(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockRunShowService{
		workspace: &tfe.Workspace{
			ID:   "ws-123",
			Name: "empty-workspace",
		},
		runList: &tfe.RunList{
			Items: []*tfe.Run{},
		},
	}

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"-w", "empty-workspace"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no runs found for workspace") {
		t.Errorf("expected 'no runs found' error, got: %v", err)
	}
}

func TestRunShow_WithWorkspace_WorkspaceNotFound(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockRunShowService{
		workspaceErr: fmt.Errorf("workspace not found"),
	}

	cmd := newCmdRunShowWith(func() (runShowService, error) {
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
	if !strings.Contains(err.Error(), "failed to read workspace") {
		t.Errorf("expected 'failed to read workspace' error, got: %v", err)
	}
}

func TestRunShow_WithWorkspace_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return &mockRunShowService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"-w", "production"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "organization is required") {
		t.Errorf("expected 'organization is required' error, got: %v", err)
	}
}

func TestRunShow_WithWorkspace_ListRunsError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockRunShowService{
		workspace: &tfe.Workspace{
			ID:   "ws-123",
			Name: "production",
		},
		runListErr: fmt.Errorf("API error"),
	}

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"-w", "production"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to list runs") {
		t.Errorf("expected 'failed to list runs' error, got: %v", err)
	}
}

// mockRunShowServiceWithWatch supports simulating status changes over time
type mockRunShowServiceWithWatch struct {
	workspace    *tfe.Workspace
	workspaceErr error
	runList      *tfe.RunList
	runListErr   error
	runs         []*tfe.Run // runs to return in sequence
	readCount    int        // number of times ReadRun has been called
	readErr      error      // error to return on ReadRun
}

func (m *mockRunShowServiceWithWatch) ListRuns(_ context.Context, _ string, _ *tfe.RunListOptions) (*tfe.RunList, error) {
	if m.runListErr != nil {
		return nil, m.runListErr
	}
	return m.runList, nil
}

func (m *mockRunShowServiceWithWatch) ReadRun(_ context.Context, _ string) (*tfe.Run, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	if m.readCount >= len(m.runs) {
		// Return the last run if we've exhausted the sequence
		return m.runs[len(m.runs)-1], nil
	}
	r := m.runs[m.readCount]
	m.readCount++
	return r, nil
}

func (m *mockRunShowServiceWithWatch) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	return nil, nil
}

func (m *mockRunShowServiceWithWatch) ReadWorkspace(_ context.Context, _, _ string) (*tfe.Workspace, error) {
	if m.workspaceErr != nil {
		return nil, m.workspaceErr
	}
	return m.workspace, nil
}

func (m *mockRunShowServiceWithWatch) ReadRunWithApply(_ context.Context, _ string) (*tfe.Run, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	if m.readCount >= len(m.runs) {
		return m.runs[len(m.runs)-1], nil
	}
	r := m.runs[m.readCount]
	m.readCount++
	return r, nil
}

func (m *mockRunShowServiceWithWatch) ReadPlanJSONOutput(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("ReadPlanJSONOutput not implemented in mockRunShowServiceWithWatch")
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status   tfe.RunStatus
		expected bool
	}{
		{tfe.RunApplied, true},
		{tfe.RunErrored, true},
		{tfe.RunCanceled, true},
		{tfe.RunDiscarded, true},
		{tfe.RunPlannedAndFinished, true},
		{tfe.RunPlannedAndSaved, true},
		{tfe.RunPending, false},
		{tfe.RunPlanning, false},
		{tfe.RunApplying, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := isTerminalStatus(tt.status)
			if got != tt.expected {
				t.Errorf("isTerminalStatus(%v) = %v, want %v", tt.status, got, tt.expected)
			}
		})
	}
}

func TestFormatStatusUpdate(t *testing.T) {
	ts := time.Date(2026, 2, 14, 10, 30, 45, 0, time.UTC)
	got := formatStatusUpdate(ts, tfe.RunApplying)
	want := "2026-02-14 10:30:45  Status: applying"
	if got != want {
		t.Errorf("formatStatusUpdate() = %q, want %q", got, want)
	}
}

func TestRunShow_Watch_StatusChange(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowServiceWithWatch{
		runs: []*tfe.Run{
			{
				ID:               "run-watch123",
				Status:           tfe.RunPlanning,
				Message:          "Planning",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
			{
				ID:               "run-watch123",
				Status:           tfe.RunPlanning,
				Message:          "Planning",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
			{
				ID:               "run-watch123",
				Status:           tfe.RunApplying,
				Message:          "Planning",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
			{
				ID:               "run-watch123",
				Status:           tfe.RunApplied,
				Message:          "Planning",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
				Plan: &tfe.Plan{
					ResourceAdditions:    1,
					ResourceChanges:      2,
					ResourceDestructions: 3,
				},
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShowWithInterval(mock, "run-watch123", "", "", true, false, 10*time.Millisecond)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Verify initial display
	if !strings.Contains(got, "ID:") || !strings.Contains(got, "run-watch123") {
		t.Errorf("expected initial display in output, got:\n%s", got)
	}

	// Verify separator line
	if !strings.Contains(got, "---") {
		t.Errorf("expected separator line in output, got:\n%s", got)
	}

	// Verify that status is output on each poll
	if !strings.Contains(got, "Status: planning") {
		t.Errorf("expected 'Status: planning' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "Status: applying") {
		t.Errorf("expected 'Status: applying' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "Status: applied") {
		t.Errorf("expected 'Status: applied' in output, got:\n%s", got)
	}

	// Verify that detailed info is re-displayed at completion
	if !strings.Contains(got, "Plan Changes:       +1 ~2 -3") {
		t.Errorf("expected final Plan Changes in output, got:\n%s", got)
	}
	if !strings.Contains(got, "Has Changes:        true") {
		t.Errorf("expected final Has Changes in output, got:\n%s", got)
	}
}

func TestRunShow_Watch_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)

	mock := &mockRunShowServiceWithWatch{
		runs: []*tfe.Run{
			{
				ID:               "run-watch-json",
				Status:           tfe.RunPlanning,
				Message:          "Planning",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
			{
				ID:               "run-watch-json",
				Status:           tfe.RunApplied,
				Message:          "Applied",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShowWithInterval(mock, "run-watch-json", "", "", true, false, 10*time.Millisecond)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// In JSON mode, only the final result is output
	if !strings.Contains(got, `"id": "run-watch-json"`) {
		t.Errorf("expected JSON output with run ID, got:\n%s", got)
	}
	if !strings.Contains(got, `"status": "applied"`) {
		t.Errorf("expected final status 'applied' in JSON output, got:\n%s", got)
	}

	// Verify that separator lines and status updates are not output in JSON mode
	if strings.Contains(got, "---") {
		t.Errorf("unexpected separator line in JSON output, got:\n%s", got)
	}
	if strings.Contains(got, "Status: planning") {
		t.Errorf("unexpected status update in JSON output, got:\n%s", got)
	}
}

func TestRunShow_Watch_AlreadyTerminal(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowServiceWithWatch{
		runs: []*tfe.Run{
			{
				ID:               "run-terminal",
				Status:           tfe.RunApplied,
				Message:          "Already applied",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRunShowWithInterval(mock, "run-terminal", "", "", true, false, 10*time.Millisecond)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Only initial display is output
	if !strings.Contains(got, "ID:") || !strings.Contains(got, "run-terminal") {
		t.Errorf("expected initial display in output, got:\n%s", got)
	}

	// Separator line is not output (run already in terminal status)
	if strings.Contains(got, "---") {
		t.Errorf("unexpected separator line in output (run already terminal), got:\n%s", got)
	}

	// Verify that ReadRun is not called
	if mock.readCount > 1 {
		t.Errorf("expected 1 ReadRun call for terminal run, got %d", mock.readCount)
	}
}

func TestRunShow_Watch_APIError(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowServiceWithWatch{
		runs: []*tfe.Run{
			{
				ID:               "run-error",
				Status:           tfe.RunPlanning,
				Message:          "Planning",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
			// Second and subsequent calls return error, then succeed
			{
				ID:               "run-error",
				Status:           tfe.RunPlanning,
				Message:          "Planning",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
			{
				ID:               "run-error",
				Status:           tfe.RunApplied,
				Message:          "Applied",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				IsDestroy:        false,
				CreatedAt:        time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			},
		},
		readErr: nil, // A separate mock is needed to simulate errors
	}

	oldStdout := os.Stdout
	stdoutR, stdoutW, _ := os.Pipe()
	os.Stdout = stdoutW

	oldStderr := os.Stderr
	stderrR, stderrW, _ := os.Pipe()
	os.Stderr = stderrW

	// Switch to mock with error
	mockWithError := &mockRunShowServiceWithWatchError{
		initialRun: mock.runs[0],
		finalRun:   mock.runs[2],
	}

	err := runRunShowWithInterval(mockWithError, "run-error", "", "", true, false, 10*time.Millisecond)

	_ = stdoutW.Close()
	_ = stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var stdoutBuf bytes.Buffer
	_, _ = stdoutBuf.ReadFrom(stdoutR)
	gotStdout := stdoutBuf.String()

	var stderrBuf bytes.Buffer
	_, _ = stderrBuf.ReadFrom(stderrR)
	gotStderr := stderrBuf.String()

	// Verify that warning message is output to stderr
	if !strings.Contains(gotStderr, "Warning: failed to read run") {
		t.Errorf("expected warning message in stderr, got:\n%s", gotStderr)
	}

	// Verify that applied status is displayed after eventual success
	if !strings.Contains(gotStdout, "Status: applied") {
		t.Errorf("expected final status in stdout, got:\n%s", gotStdout)
	}
}

// mockRunShowServiceWithWatchError simulates API error on second call
type mockRunShowServiceWithWatchError struct {
	initialRun *tfe.Run
	finalRun   *tfe.Run
	callCount  int
}

func (m *mockRunShowServiceWithWatchError) ListRuns(_ context.Context, _ string, _ *tfe.RunListOptions) (*tfe.RunList, error) {
	return nil, nil
}

func (m *mockRunShowServiceWithWatchError) ReadRun(_ context.Context, _ string) (*tfe.Run, error) {
	m.callCount++
	if m.callCount == 1 {
		return m.initialRun, nil
	}
	if m.callCount == 2 {
		return nil, fmt.Errorf("temporary API error")
	}
	return m.finalRun, nil
}

func (m *mockRunShowServiceWithWatchError) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	return nil, nil
}

func (m *mockRunShowServiceWithWatchError) ReadWorkspace(_ context.Context, _, _ string) (*tfe.Workspace, error) {
	return nil, nil
}

func (m *mockRunShowServiceWithWatchError) ReadRunWithApply(_ context.Context, _ string) (*tfe.Run, error) {
	m.callCount++
	if m.callCount == 1 {
		return m.initialRun, nil
	}
	if m.callCount == 2 {
		return nil, fmt.Errorf("temporary API error")
	}
	return m.finalRun, nil
}

func (m *mockRunShowServiceWithWatchError) ReadPlanJSONOutput(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("ReadPlanJSONOutput not implemented in mockRunShowServiceWithWatchError")
}

func TestRunShow_WithPR_InvalidRepoFormat(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return &mockRunShowService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--pr", "42", "--repo", "invalidrepo"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--repo must be in format 'owner/repo'") {
		t.Errorf("expected 'owner/repo format' error, got: %v", err)
	}
}

func TestRunShow_RunID_and_PR_MutuallyExclusive(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return &mockRunShowService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"run-abc123", "--pr", "42", "--repo", "owner/repo"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot specify both run-id and --pr") {
		t.Errorf("expected 'mutually exclusive' error, got: %v", err)
	}
}

func TestRunShow_NoArgsNoPRNoWorkspace(t *testing.T) {
	viper.Reset()

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return &mockRunShowService{}, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "either run-id, --pr, or --workspace/-w is required") {
		t.Errorf("expected 'run-id or --pr or --workspace/-w required' error, got: %v", err)
	}
}

func TestRunShow_PlanJSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowServiceExtended{
		mockRunShowService: mockRunShowService{
			run: &tfe.Run{
				ID:     "run-abc123",
				Status: tfe.RunApplied,
				Plan: &tfe.Plan{
					ID:                   "plan-xyz",
					ResourceAdditions:    2,
					ResourceChanges:      1,
					ResourceDestructions: 0,
				},
				Message:          "Apply complete",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				CreatedAt:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		planJSON: []byte(`{"format_version":"1.2","terraform_version":"1.5.0","resource_changes":[{"address":"aws_instance.example","mode":"managed","type":"aws_instance","name":"example","change":{"actions":["create"]}}]}`),
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return mock, nil
	})
	cmd.SetArgs([]string{"run-abc123", "--plan-json"})

	err := cmd.Execute()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify table output
	if !strings.Contains(output, "run-abc123") {
		t.Errorf("expected run ID in output")
	}

	// Verify separator and plan JSON
	if !strings.Contains(output, "---") {
		t.Errorf("expected separator in output")
	}
	if !strings.Contains(output, "format_version") {
		t.Errorf("expected plan JSON in output")
	}
	if !strings.Contains(output, "aws_instance.example") {
		t.Errorf("expected resource in plan JSON")
	}
}

func TestRunShow_PlanJSON_JSONMode(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)

	mock := &mockRunShowServiceExtended{
		mockRunShowService: mockRunShowService{
			run: &tfe.Run{
				ID:     "run-abc123",
				Status: tfe.RunApplied,
				Plan: &tfe.Plan{
					ID:                   "plan-xyz",
					ResourceAdditions:    2,
					ResourceChanges:      1,
					ResourceDestructions: 0,
				},
				Message:          "Apply complete",
				TerraformVersion: "1.5.0",
				HasChanges:       true,
				CreatedAt:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		planJSON: []byte(`{"format_version":"1.2","terraform_version":"1.5.0"}`),
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return mock, nil
	})
	cmd.SetArgs([]string{"run-abc123", "--plan-json"})

	err := cmd.Execute()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	// Verify "run" field
	run, ok := result["run"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'run' field in JSON output")
	}
	if run["id"] != "run-abc123" {
		t.Errorf("unexpected run ID: %v", run["id"])
	}

	// Verify "plan_json" field
	planJSON, ok := result["plan_json"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'plan_json' field in JSON output")
	}
	if planJSON["format_version"] != "1.2" {
		t.Errorf("unexpected format_version: %v", planJSON["format_version"])
	}
}

func TestRunShow_PlanJSON_NoPlan(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockRunShowServiceExtended{
		mockRunShowService: mockRunShowService{
			run: &tfe.Run{
				ID:     "run-abc123",
				Status: tfe.RunPlanned,
				Plan:   nil, // No plan
			},
		},
	}

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return mock, nil
	})
	cmd.SetArgs([]string{"run-abc123", "--plan-json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when plan is nil")
	}
	if !strings.Contains(err.Error(), "does not have a plan") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunShow_WithWatch_PlanJSON_Error(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	cmd := newCmdRunShowWith(func() (runShowService, error) {
		return &mockRunShowServiceExtended{}, nil
	})
	cmd.SetArgs([]string{"run-abc123", "--watch", "--plan-json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when using --watch with --plan-json")
	}
	if !strings.Contains(err.Error(), "cannot be used with --watch") {
		t.Errorf("unexpected error message: %v", err)
	}
}
