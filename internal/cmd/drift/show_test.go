package drift

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

type mockDriftShowService struct {
	workspace       *tfe.Workspace
	readErr         error
	assessments     map[string]*client.AssessmentResult
	assessErr       error
	driftDetails    map[string][]client.DriftedResource
	driftDetailsErr error
}

func (m *mockDriftShowService) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDriftShowService) ReadWorkspace(_ context.Context, _ string, _ string) (*tfe.Workspace, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	return m.workspace, nil
}

func (m *mockDriftShowService) ReadCurrentAssessment(_ context.Context, workspaceID string) (*client.AssessmentResult, error) {
	if m.assessErr != nil {
		return nil, m.assessErr
	}
	if m.assessments != nil {
		return m.assessments[workspaceID], nil
	}
	return nil, nil
}

func (m *mockDriftShowService) ReadAssessmentDriftDetails(_ context.Context, assessmentID string) ([]client.DriftedResource, error) {
	if m.driftDetailsErr != nil {
		return nil, m.driftDetailsErr
	}
	if m.driftDetails != nil {
		return m.driftDetails[assessmentID], nil
	}
	return nil, nil
}

func TestDriftShow_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftShowService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-abc123": {
				ID:                 "asmnt-001",
				Drifted:            true,
				Succeeded:          true,
				ResourcesDrifted:   3,
				ResourcesUndrifted: 12,
				CreatedAt:          "2025-01-20T10:30:00.000Z",
			},
		},
		driftDetails: map[string][]client.DriftedResource{
			"asmnt-001": {
				{Address: "aws_security_group.web", Type: "aws_security_group", Name: "web", Action: "update"},
				{Address: "aws_s3_bucket.logs", Type: "aws_s3_bucket", Name: "logs", Action: "update"},
				{Address: "aws_iam_role.lambda", Type: "aws_iam_role", Name: "lambda", Action: "delete"},
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{
		"Workspace:", "my-workspace",
		"Drifted:", "true",
		"Resources Drifted:", "3",
		"RESOURCE", "TYPE", "ACTION",
		"aws_security_group.web", "aws_security_group", "update",
		"aws_s3_bucket.logs", "aws_s3_bucket",
		"aws_iam_role.lambda", "aws_iam_role", "delete",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestDriftShow_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockDriftShowService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-abc123": {
				ID:                 "asmnt-001",
				Drifted:            true,
				Succeeded:          true,
				ResourcesDrifted:   3,
				ResourcesUndrifted: 12,
				CreatedAt:          "2025-01-20T10:30:00.000Z",
			},
		},
		driftDetails: map[string][]client.DriftedResource{
			"asmnt-001": {
				{Address: "aws_security_group.web", Type: "aws_security_group", Name: "web", Action: "update"},
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{
		`"workspace": "my-workspace"`,
		`"drifted": true`,
		`"resources_drifted": 3`,
		`"drifted_resources"`,
		`"address": "aws_security_group.web"`,
		`"type": "aws_security_group"`,
		`"action": "update"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestDriftShow_NoDrift_NoResourceTable(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftShowService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-abc123": {
				ID:                 "asmnt-001",
				Drifted:            false,
				Succeeded:          true,
				ResourcesDrifted:   0,
				ResourcesUndrifted: 15,
				CreatedAt:          "2025-01-20T10:30:00.000Z",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if strings.Contains(got, "RESOURCE") {
		t.Errorf("resource table should not appear when not drifted, got:\n%s", got)
	}
	if !strings.Contains(got, "false") {
		t.Errorf("expected 'false' for drifted, got:\n%s", got)
	}
}

func TestDriftShow_NotReady(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftShowService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "not ready") {
		t.Errorf("expected 'not ready' in output, got:\n%s", got)
	}
}

func TestDriftShow_DriftDetailsError_NonFatal(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftShowService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-abc123": {
				ID:                 "asmnt-001",
				Drifted:            true,
				Succeeded:          true,
				ResourcesDrifted:   2,
				ResourcesUndrifted: 10,
				CreatedAt:          "2025-01-20T10:30:00.000Z",
			},
		},
		driftDetailsErr: fmt.Errorf("json-output endpoint returned HTTP 500"),
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Capture stderr for warning
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	err := runDriftShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("expected no error (non-fatal), got: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Summary should still be shown
	if !strings.Contains(got, "Drifted:") {
		t.Errorf("expected summary output, got:\n%s", got)
	}

	// Warning should be on stderr
	var errBuf bytes.Buffer
	_, _ = errBuf.ReadFrom(rErr)
	errGot := errBuf.String()

	if !strings.Contains(errGot, "Warning") {
		t.Errorf("expected warning on stderr, got:\n%s", errGot)
	}
}

func TestDriftShow_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdDriftShowWith(func() (driftShowService, error) {
		return &mockDriftShowService{}, nil
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

func TestDriftShow_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdDriftShowWith(func() (driftShowService, error) {
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

func TestDriftShow_ReadWorkspaceError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockDriftShowService{
		readErr: fmt.Errorf("workspace not found"),
	}

	cmd := newCmdDriftShowWith(func() (driftShowService, error) {
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

func TestDriftShow_AssessmentError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockDriftShowService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
		},
		assessErr: fmt.Errorf("assessment API error"),
	}

	cmd := newCmdDriftShowWith(func() (driftShowService, error) {
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

func TestDriftShow_NoArgs(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdDriftShowWith(func() (driftShowService, error) {
		return &mockDriftShowService{}, nil
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
