package drift

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

func TestDriftShow_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
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

	err := runDriftShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"Workspace:", "my-workspace", "Drifted:", "true", "Resources Drifted:", "3", "Resources Undrifted:", "12", "Last Assessment:", "2025-01-20T10:30:00.000Z"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestDriftShow_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockDriftService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
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

	err := runDriftShow(mock, "test-org", "my-workspace")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"workspace": "my-workspace"`, `"drifted": true`, `"resources_drifted": 3`, `"resources_undrifted": 12`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestDriftShow_NotReady(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftService{
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

func TestDriftShow_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdDriftShowWith(func() (driftService, error) {
		return &mockDriftService{}, nil
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

	cmd := newCmdDriftShowWith(func() (driftService, error) {
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

	mock := &mockDriftService{
		readErr: fmt.Errorf("workspace not found"),
	}

	cmd := newCmdDriftShowWith(func() (driftService, error) {
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

	mock := &mockDriftService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
		},
		assessErr: fmt.Errorf("assessment API error"),
	}

	cmd := newCmdDriftShowWith(func() (driftService, error) {
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

	cmd := newCmdDriftShowWith(func() (driftService, error) {
		return &mockDriftService{}, nil
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
