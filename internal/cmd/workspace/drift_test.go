package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

type mockWSDriftService struct {
	mockWSService
	assessments map[string]*client.AssessmentResult
	assessErr   error
}

func (m *mockWSDriftService) ReadCurrentAssessment(_ context.Context, workspaceID string) (*client.AssessmentResult, error) {
	if m.assessErr != nil {
		return nil, m.assessErr
	}
	if m.assessments != nil {
		return m.assessments[workspaceID], nil
	}
	return nil, nil
}

func TestWorkspaceDrift_Single_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockWSDriftService{
		mockWSService: mockWSService{
			workspace: &tfe.Workspace{
				Name:               "my-workspace",
				ID:                 "ws-abc123",
				AssessmentsEnabled: true,
			},
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-abc123": {
				Drifted:            true,
				Succeeded:          true,
				ResourcesDrifted:   3,
				ResourcesUndrifted: 12,
				CreatedAt:          "2025-01-20T10:30:00.000Z",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceDrift(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"Workspace:", "my-workspace", "Assessments:", "true", "Drifted:", "true", "Resources Drifted:", "3", "Resources Undrifted:", "12", "Last Assessment:", "2025-01-20T10:30:00.000Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceDrift_Single_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockWSDriftService{
		mockWSService: mockWSService{
			workspace: &tfe.Workspace{
				Name:               "my-workspace",
				ID:                 "ws-abc123",
				AssessmentsEnabled: true,
			},
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-abc123": {
				Drifted:            true,
				Succeeded:          true,
				ResourcesDrifted:   3,
				ResourcesUndrifted: 12,
				CreatedAt:          "2025-01-20T10:30:00.000Z",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceDrift(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"workspace": "my-workspace"`, `"assessments": true`, `"drifted": true`, `"resources_drifted": 3`, `"resources_undrifted": 12`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceDrift_All_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockWSDriftService{
		mockWSService: mockWSService{
			workspaces: []*tfe.Workspace{
				{Name: "prod-vpc", ID: "ws-001", AssessmentsEnabled: true},
				{Name: "staging", ID: "ws-002", AssessmentsEnabled: true},
				{Name: "dev", ID: "ws-003", AssessmentsEnabled: false},
			},
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-001": {Drifted: true, ResourcesDrifted: 3, CreatedAt: "2025-01-20T10:30:00.000Z"},
			"ws-002": {Drifted: false, ResourcesDrifted: 0, CreatedAt: "2025-01-20T10:30:00.000Z"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceDriftAll(mock, "test-org")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"WORKSPACE", "ASSESSMENTS", "DRIFTED", "prod-vpc", "staging", "dev", "true", "false"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
	// dev workspace should show "-" for drifted since assessments are disabled
	lines := strings.Split(got, "\n")
	for _, line := range lines {
		if strings.Contains(line, "dev") && !strings.Contains(line, "-") {
			t.Errorf("expected '-' for dev workspace with disabled assessments, got:\n%s", line)
		}
	}
}

func TestWorkspaceDrift_AssessmentsDisabled_NoResult(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	// Assessment API returns nil (404) — no assessment available
	mock := &mockWSDriftService{
		mockWSService: mockWSService{
			workspace: &tfe.Workspace{
				Name:               "my-workspace",
				ID:                 "ws-abc123",
				AssessmentsEnabled: false,
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceDrift(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "false") {
		t.Errorf("expected 'false' for assessments in output, got:\n%s", got)
	}
	// Should show "-" for drift fields when no assessment result exists
	if !strings.Contains(got, "-") {
		t.Errorf("expected '-' for no assessment result, got:\n%s", got)
	}
}

func TestWorkspaceDrift_OrgEnforced_WorkspaceDisabled(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	// AssessmentsEnabled is false but org enforces assessments,
	// so the API returns a valid result
	mock := &mockWSDriftService{
		mockWSService: mockWSService{
			workspace: &tfe.Workspace{
				Name:               "my-workspace",
				ID:                 "ws-abc123",
				AssessmentsEnabled: false,
			},
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-abc123": {
				Drifted:            true,
				Succeeded:          true,
				ResourcesDrifted:   5,
				ResourcesUndrifted: 20,
				CreatedAt:          "2025-01-20T10:30:00.000Z",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWorkspaceDrift(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Even though AssessmentsEnabled is false, drift data should be shown
	for _, want := range []string{"Drifted:", "true", "Resources Drifted:", "5", "Resources Undrifted:", "20"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceDrift_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdWorkspaceDriftWith(func() (wsDriftService, error) {
		return &mockWSDriftService{}, nil
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

func TestWorkspaceDrift_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdWorkspaceDriftWith(func() (wsDriftService, error) {
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

func TestWorkspaceDrift_ReadWorkspaceError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockWSDriftService{
		mockWSService: mockWSService{
			readErr: fmt.Errorf("workspace not found"),
		},
	}

	cmd := newCmdWorkspaceDriftWith(func() (wsDriftService, error) {
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
	if !strings.Contains(err.Error(), "workspace not found") {
		t.Errorf("expected 'workspace not found' error, got: %v", err)
	}
}

func TestWorkspaceDrift_AssessmentAPIError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockWSDriftService{
		mockWSService: mockWSService{
			workspace: &tfe.Workspace{
				Name:               "my-workspace",
				ID:                 "ws-abc123",
				AssessmentsEnabled: true,
			},
		},
		assessErr: fmt.Errorf("assessment API error"),
	}

	cmd := newCmdWorkspaceDriftWith(func() (wsDriftService, error) {
		return mock, nil
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
	if !strings.Contains(err.Error(), "assessment API error") {
		t.Errorf("expected 'assessment API error' error, got: %v", err)
	}
}

func TestWorkspaceDrift_NoArgsNoAll(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdWorkspaceDriftWith(func() (wsDriftService, error) {
		return &mockWSDriftService{}, nil
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
	if !strings.Contains(err.Error(), "workspace name is required") {
		t.Errorf("expected 'workspace name is required' error, got: %v", err)
	}
}
