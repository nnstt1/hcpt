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

	err := runDriftShow(mock, "test-org", "my-workspace", false)

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

	err := runDriftShow(mock, "test-org", "my-workspace", false)

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

	err := runDriftShow(mock, "test-org", "my-workspace", false)

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

	err := runDriftShow(mock, "test-org", "my-workspace", false)

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

	err := runDriftShow(mock, "test-org", "my-workspace", false)

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

func mockWithDiffData() *mockDriftShowService {
	return &mockDriftShowService{
		workspace: &tfe.Workspace{
			Name: "my-workspace",
			ID:   "ws-abc123",
		},
		assessments: map[string]*client.AssessmentResult{
			"ws-abc123": {
				ID:                 "asmnt-001",
				Drifted:            true,
				Succeeded:          true,
				ResourcesDrifted:   1,
				ResourcesUndrifted: 5,
				CreatedAt:          "2025-01-20T10:30:00.000Z",
			},
		},
		driftDetails: map[string][]client.DriftedResource{
			"asmnt-001": {
				{
					Address: "aws_security_group.web",
					Type:    "aws_security_group",
					Name:    "web",
					Action:  "update",
					Before: map[string]interface{}{
						"ingress": []interface{}{
							map[string]interface{}{
								"cidr_blocks": []interface{}{"10.0.0.0/16"},
								"from_port":   float64(443),
							},
						},
						"tags": map[string]interface{}{
							"env": "prod",
						},
						"old_field": "removed",
					},
					After: map[string]interface{}{
						"ingress": []interface{}{
							map[string]interface{}{
								"cidr_blocks": []interface{}{"0.0.0.0/0"},
								"from_port":   float64(443),
							},
						},
						"tags": map[string]interface{}{
							"env":     "staging",
							"version": "2",
						},
					},
				},
			},
		},
	}
}

func TestDriftShow_Verbose_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := mockWithDiffData()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftShow(mock, "test-org", "my-workspace", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Should contain diff markers
	for _, want := range []string{
		"Resource:", "aws_security_group.web", "(update)",
		"~", "=>",
		"ingress.0.cidr_blocks.0",
		`"10.0.0.0/16"`, `"0.0.0.0/0"`,
		"+", "tags.version",
		"-", "old_field",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in verbose output, got:\n%s", want, got)
		}
	}
}

func TestDriftShow_Verbose_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := mockWithDiffData()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftShow(mock, "test-org", "my-workspace", true)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{
		`"changes"`,
		`"ingress.0.cidr_blocks.0"`,
		`"before"`,
		`"after"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in verbose JSON output, got:\n%s", want, got)
		}
	}
}

func TestDriftShow_NoVerbose_NoChanges(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := mockWithDiffData()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDriftShow(mock, "test-org", "my-workspace", false)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if strings.Contains(got, `"changes"`) {
		t.Errorf("expected no 'changes' field without verbose, got:\n%s", got)
	}
}

func TestComputeDiffs(t *testing.T) {
	tests := []struct {
		name      string
		before    map[string]interface{}
		after     map[string]interface{}
		wantKeys  []string
		wantCount int
	}{
		{
			name:      "changed value",
			before:    map[string]interface{}{"name": "old"},
			after:     map[string]interface{}{"name": "new"},
			wantKeys:  []string{"name"},
			wantCount: 1,
		},
		{
			name:      "added value",
			before:    map[string]interface{}{},
			after:     map[string]interface{}{"new_key": "val"},
			wantKeys:  []string{"new_key"},
			wantCount: 1,
		},
		{
			name:      "removed value",
			before:    map[string]interface{}{"old_key": "val"},
			after:     map[string]interface{}{},
			wantKeys:  []string{"old_key"},
			wantCount: 1,
		},
		{
			name:      "no change",
			before:    map[string]interface{}{"same": "val"},
			after:     map[string]interface{}{"same": "val"},
			wantKeys:  nil,
			wantCount: 0,
		},
		{
			name: "nested map flattened",
			before: map[string]interface{}{
				"tags": map[string]interface{}{"env": "prod"},
			},
			after: map[string]interface{}{
				"tags": map[string]interface{}{"env": "staging"},
			},
			wantKeys:  []string{"tags.env"},
			wantCount: 1,
		},
		{
			name: "array flattened",
			before: map[string]interface{}{
				"cidrs": []interface{}{"10.0.0.0/16"},
			},
			after: map[string]interface{}{
				"cidrs": []interface{}{"0.0.0.0/0"},
			},
			wantKeys:  []string{"cidrs.0"},
			wantCount: 1,
		},
		{
			name:      "nil before and after",
			before:    nil,
			after:     nil,
			wantKeys:  nil,
			wantCount: 0,
		},
		{
			name:      "added nil value ignored",
			before:    map[string]interface{}{},
			after:     map[string]interface{}{"key": nil},
			wantKeys:  nil,
			wantCount: 0,
		},
		{
			name:      "removed nil value ignored",
			before:    map[string]interface{}{"key": nil},
			after:     map[string]interface{}{},
			wantKeys:  nil,
			wantCount: 0,
		},
		{
			name:      "nil vs empty array",
			before:    map[string]interface{}{"tags": nil},
			after:     map[string]interface{}{"tags": []interface{}{}},
			wantKeys:  []string{"tags"},
			wantCount: 1,
		},
		{
			name:      "empty array vs nil",
			before:    map[string]interface{}{"tags": []interface{}{}},
			after:     map[string]interface{}{"tags": nil},
			wantKeys:  []string{"tags"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := computeDiffs(tt.before, tt.after, nil, nil, nil)
			if len(diffs) != tt.wantCount {
				t.Errorf("expected %d diffs, got %d: %+v", tt.wantCount, len(diffs), diffs)
			}
			for _, wantKey := range tt.wantKeys {
				found := false
				for _, d := range diffs {
					if d.Key == wantKey {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected diff key %q not found in %+v", wantKey, diffs)
				}
			}
		})
	}
}

func TestComputeDiffs_KnownAfterApply(t *testing.T) {
	before := map[string]interface{}{
		"sku":  "Standard_B1ms",
		"name": "myvm",
	}
	after := map[string]interface{}{
		"sku":  nil, // null because it will be known after apply
		"name": "myvm",
	}
	afterUnknown := map[string]interface{}{
		"sku": true, // this attribute is known after apply
	}

	diffs := computeDiffs(before, after, afterUnknown, nil, nil)

	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %+v", len(diffs), diffs)
	}
	d := diffs[0]
	if d.Key != "sku" {
		t.Errorf("expected key 'sku', got %q", d.Key)
	}
	if d.After != "(known after apply)" {
		t.Errorf("expected After '(known after apply)', got %q", d.After)
	}
	if !d.KnownAfterApply {
		t.Error("expected KnownAfterApply to be true")
	}
}

func TestComputeDiffs_KnownAfterApply_NilAfterUnknown(t *testing.T) {
	// nil afterUnknown should not cause panic, null after stays "(null)"
	before := map[string]interface{}{"key": "value"}
	after := map[string]interface{}{"key": nil}

	diffs := computeDiffs(before, after, nil, nil, nil)

	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].After != "(null)" {
		t.Errorf("expected '(null)', got %q", diffs[0].After)
	}
	if diffs[0].KnownAfterApply {
		t.Error("expected KnownAfterApply to be false")
	}
}

func TestComputeDiffs_KnownAfterApply_ParentKey(t *testing.T) {
	// after_unknown marks a parent object as true, meaning all children are unknown.
	// after may not contain those keys at all (absent, not null).
	before := map[string]interface{}{
		"annotations": map[string]interface{}{
			"env":     "prod",
			"version": "1.0",
		},
		"name": "myresource",
	}
	after := map[string]interface{}{
		"name": "myresource",
		// "annotations" is absent — unknown after apply
	}
	afterUnknown := map[string]interface{}{
		"annotations": true, // entire annotations block is unknown
	}

	diffs := computeDiffs(before, after, afterUnknown, nil, nil)

	// should find 2 diffs: annotations.env and annotations.version, both known after apply
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d: %+v", len(diffs), diffs)
	}
	for _, d := range diffs {
		if d.After != "(known after apply)" {
			t.Errorf("key %q: expected '(known after apply)', got %q", d.Key, d.After)
		}
		if !d.KnownAfterApply {
			t.Errorf("key %q: expected KnownAfterApply=true", d.Key)
		}
	}
}

func TestComputeDiffs_SensitiveValue(t *testing.T) {
	before := map[string]interface{}{
		"password": "secret123",
		"name":     "myresource",
	}
	after := map[string]interface{}{
		"password": "newsecret456",
		"name":     "myresource",
	}
	beforeSensitive := map[string]interface{}{
		"password": true,
	}
	afterSensitive := map[string]interface{}{
		"password": true,
	}

	diffs := computeDiffs(before, after, nil, beforeSensitive, afterSensitive)

	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %+v", len(diffs), diffs)
	}
	d := diffs[0]
	if d.Key != "password" {
		t.Errorf("expected key 'password', got %q", d.Key)
	}
	if d.Before != "(sensitive value)" {
		t.Errorf("expected Before '(sensitive value)', got %q", d.Before)
	}
	if d.After != "(sensitive value)" {
		t.Errorf("expected After '(sensitive value)', got %q", d.After)
	}
	if !d.Sensitive {
		t.Error("expected Sensitive to be true")
	}
}

func TestComputeDiffs_SensitiveValue_ParentKey(t *testing.T) {
	// before_sensitive marks an entire nested block as sensitive.
	// Even though display is masked, raw value comparison detects actual changes.
	before := map[string]interface{}{
		"credentials": map[string]interface{}{
			"client_id":     "abc123",
			"client_secret": "supersecret",
		},
	}
	after := map[string]interface{}{
		"credentials": map[string]interface{}{
			"client_id":     "abc123",        // unchanged
			"client_secret": "newsupersecret", // changed
		},
	}
	beforeSensitive := map[string]interface{}{
		"credentials": true,
	}
	afterSensitive := map[string]interface{}{
		"credentials": true,
	}

	diffs := computeDiffs(before, after, nil, beforeSensitive, afterSensitive)

	// Only client_secret changed (raw value differs); client_id did not change
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d: %+v", len(diffs), diffs)
	}
	d := diffs[0]
	if d.Key != "credentials.client_secret" {
		t.Errorf("expected key 'credentials.client_secret', got %q", d.Key)
	}
	if d.Before != "(sensitive value)" {
		t.Errorf("expected Before '(sensitive value)', got %q", d.Before)
	}
	if d.After != "(sensitive value)" {
		t.Errorf("expected After '(sensitive value)', got %q", d.After)
	}
	if !d.Sensitive {
		t.Error("expected Sensitive=true")
	}
}
