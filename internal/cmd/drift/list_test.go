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

type mockDriftService struct {
	workspaces  []*tfe.Workspace
	workspace   *tfe.Workspace
	listErr     error
	readErr     error
	assessments map[string]*client.AssessmentResult
	assessErr   error
}

func (m *mockDriftService) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &tfe.WorkspaceList{
		Items: m.workspaces,
	}, nil
}

func (m *mockDriftService) ReadWorkspace(_ context.Context, _ string, _ string) (*tfe.Workspace, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	return m.workspace, nil
}

func (m *mockDriftService) ReadCurrentAssessment(_ context.Context, workspaceID string) (*client.AssessmentResult, error) {
	if m.assessErr != nil {
		return nil, m.assessErr
	}
	if m.assessments != nil {
		return m.assessments[workspaceID], nil
	}
	return nil, nil
}

func TestDriftList_DriftedOnly(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftService{
		workspaces: []*tfe.Workspace{
			{Name: "prod-vpc", ID: "ws-001"},
			{Name: "staging", ID: "ws-002"},
			{Name: "dev", ID: "ws-003"},
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-001": {Drifted: true, ResourcesDrifted: 3, CreatedAt: "2025-01-20T10:30:00.000Z"},
			"ws-002": {Drifted: false, ResourcesDrifted: 0, CreatedAt: "2025-01-20T10:30:00.000Z"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Only drifted workspace should appear
	if !strings.Contains(got, "prod-vpc") {
		t.Errorf("expected drifted workspace 'prod-vpc' in output, got:\n%s", got)
	}
	if strings.Contains(got, "staging") {
		t.Errorf("non-drifted workspace 'staging' should not appear in output, got:\n%s", got)
	}
	if strings.Contains(got, "dev") {
		t.Errorf("not-ready workspace 'dev' should not appear in output, got:\n%s", got)
	}
}

func TestDriftList_All(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockDriftService{
		workspaces: []*tfe.Workspace{
			{Name: "prod-vpc", ID: "ws-001"},
			{Name: "staging", ID: "ws-002"},
			{Name: "dev", ID: "ws-003"},
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-001": {Drifted: true, ResourcesDrifted: 3, CreatedAt: "2025-01-20T10:30:00.000Z"},
			"ws-002": {Drifted: false, ResourcesDrifted: 0, CreatedAt: "2025-01-20T10:30:00.000Z"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"WORKSPACE", "DRIFTED", "prod-vpc", "staging", "dev", "true", "false", "not ready"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestDriftList_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockDriftService{
		workspaces: []*tfe.Workspace{
			{Name: "prod-vpc", ID: "ws-001"},
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-001": {Drifted: true, ResourcesDrifted: 3, ResourcesUndrifted: 12, CreatedAt: "2025-01-20T10:30:00.000Z"},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"workspace": "prod-vpc"`, `"drifted": true`, `"resources_drifted": 3`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestDriftList_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdDriftListWith(func() (driftService, error) {
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
	if !strings.Contains(err.Error(), "organization is required") {
		t.Errorf("expected 'organization is required' error, got: %v", err)
	}
}

func TestDriftList_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdDriftListWith(func() (driftService, error) {
		return nil, fmt.Errorf("token missing")
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
	if !strings.Contains(err.Error(), "token missing") {
		t.Errorf("expected 'token missing' error, got: %v", err)
	}
}

func TestDriftList_ListError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockDriftService{
		listErr: fmt.Errorf("api error"),
	}

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "api error") {
		t.Errorf("expected 'api error', got: %v", err)
	}
}

func TestDriftList_AssessmentError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockDriftService{
		workspaces: []*tfe.Workspace{
			{Name: "prod-vpc", ID: "ws-001"},
		},
		assessErr: fmt.Errorf("assessment API error"),
	}

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "assessment API error") {
		t.Errorf("expected 'assessment API error', got: %v", err)
	}
}

func TestDriftList_ConcurrentResultOrder(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	// Create 50 workspaces all drifted to verify order is preserved
	workspaces := make([]*tfe.Workspace, 50)
	assessments := make(map[string]*client.AssessmentResult)
	for i := range 50 {
		id := fmt.Sprintf("ws-%03d", i)
		name := fmt.Sprintf("workspace-%03d", i)
		workspaces[i] = &tfe.Workspace{Name: name, ID: id}
		assessments[id] = &client.AssessmentResult{
			Drifted:          true,
			ResourcesDrifted: i + 1,
			CreatedAt:        "2025-01-20T10:30:00.000Z",
		}
	}

	mock := &mockDriftService{
		workspaces:  workspaces,
		assessments: assessments,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Verify order: workspace-000 should appear before workspace-049
	idx0 := strings.Index(got, "workspace-000")
	idx49 := strings.Index(got, "workspace-049")
	if idx0 == -1 || idx49 == -1 {
		t.Fatalf("expected both workspace-000 and workspace-049 in output, got:\n%s", got)
	}
	if idx0 >= idx49 {
		t.Errorf("expected workspace-000 before workspace-049, but got idx0=%d idx49=%d", idx0, idx49)
	}
}

func TestDriftList_AssessmentErrorPropagation(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	// Create enough workspaces to exercise concurrency
	workspaces := make([]*tfe.Workspace, 10)
	for i := range 10 {
		workspaces[i] = &tfe.Workspace{
			Name: fmt.Sprintf("ws-%d", i),
			ID:   fmt.Sprintf("ws-%03d", i),
		}
	}

	mock := &mockDriftService{
		workspaces: workspaces,
		assessErr:  fmt.Errorf("rate limited"),
	}

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftList(mock, "test-org", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Fatal("expected error from concurrent assessment fetching, got nil")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected 'rate limited' error, got: %v", err)
	}
}
