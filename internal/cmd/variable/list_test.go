package variable

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/spf13/viper"
)

type mockVariableListService struct {
	workspace *tfe.Workspace
	variables []*tfe.Variable
	readErr   error
	listErr   error
	listVarFn func(opts *tfe.VariableListOptions) (*tfe.VariableList, error)
	createVar *tfe.Variable
	createErr error
	updateVar *tfe.Variable
	updateErr error
	deleteErr error
}

func (m *mockVariableListService) ListWorkspaces(_ context.Context, _ string, _ *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	return nil, nil
}

func (m *mockVariableListService) ReadWorkspace(_ context.Context, _ string, _ string) (*tfe.Workspace, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	return m.workspace, nil
}

func (m *mockVariableListService) ListVariables(_ context.Context, _ string, opts *tfe.VariableListOptions) (*tfe.VariableList, error) {
	if m.listVarFn != nil {
		return m.listVarFn(opts)
	}
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &tfe.VariableList{
		Items: m.variables,
	}, nil
}

func (m *mockVariableListService) CreateVariable(_ context.Context, _ string, _ tfe.VariableCreateOptions) (*tfe.Variable, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createVar, nil
}

func (m *mockVariableListService) UpdateVariable(_ context.Context, _ string, _ string, _ tfe.VariableUpdateOptions) (*tfe.Variable, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.updateVar, nil
}

func (m *mockVariableListService) DeleteVariable(_ context.Context, _ string, _ string) error {
	return m.deleteErr
}

func TestVariableList_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{
			{
				Key:       "AWS_REGION",
				Value:     "us-east-1",
				Category:  tfe.CategoryTerraform,
				Sensitive: false,
				HCL:       false,
			},
			{
				Key:       "SECRET_KEY",
				Value:     "",
				Category:  tfe.CategoryEnv,
				Sensitive: true,
				HCL:       false,
			},
		},
	}

	cmd := newCmdVariableListWith(func() (variableListService, error) {
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

func TestVariableList_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{
			{
				Key:       "AWS_REGION",
				Value:     "us-east-1",
				Category:  tfe.CategoryTerraform,
				Sensitive: false,
				HCL:       false,
			},
			{
				Key:       "SECRET_KEY",
				Value:     "",
				Category:  tfe.CategoryEnv,
				Sensitive: true,
				HCL:       false,
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runVariableList(mock, "test-org", "my-ws")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"KEY", "VALUE", "CATEGORY", "AWS_REGION", "us-east-1", "terraform", "(sensitive)"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestVariableList_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{
			{
				Key:       "AWS_REGION",
				Value:     "us-east-1",
				Category:  tfe.CategoryTerraform,
				Sensitive: false,
				HCL:       false,
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runVariableList(mock, "test-org", "my-ws")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"key": "AWS_REGION"`, `"value": "us-east-1"`, `"category": "terraform"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestVariableList_Pagination(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		listVarFn: func(opts *tfe.VariableListOptions) (*tfe.VariableList, error) {
			page := opts.PageNumber
			if page == 0 {
				page = 1
			}
			switch page {
			case 1:
				return &tfe.VariableList{
					Items: []*tfe.Variable{
						{Key: "VAR_1", Value: "val1", Category: tfe.CategoryTerraform},
					},
					Pagination: &tfe.Pagination{NextPage: 2, TotalPages: 2},
				}, nil
			case 2:
				return &tfe.VariableList{
					Items: []*tfe.Variable{
						{Key: "VAR_2", Value: "val2", Category: tfe.CategoryEnv},
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

	err := runVariableList(mock, "test-org", "my-ws")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"VAR_1", "VAR_2"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestVariableList_Empty(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		variables: []*tfe.Variable{},
	}

	cmd := newCmdVariableListWith(func() (variableListService, error) {
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

func TestVariableList_ListError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		workspace: &tfe.Workspace{ID: "ws-abc123", Name: "my-ws"},
		listErr:   fmt.Errorf("variables api error"),
	}

	cmd := newCmdVariableListWith(func() (variableListService, error) {
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
	if !strings.Contains(err.Error(), "variables api error") {
		t.Errorf("expected 'variables api error', got: %v", err)
	}
}

func TestVariableList_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdVariableListWith(func() (variableListService, error) {
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

func TestVariableList_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdVariableListWith(func() (variableListService, error) {
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
	if !strings.Contains(err.Error(), "organization is required") {
		t.Errorf("expected 'organization is required' error, got: %v", err)
	}
}

func TestVariableList_NoWorkspace(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdVariableListWith(func() (variableListService, error) {
		return &mockVariableListService{}, nil
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

func TestVariableList_WorkspaceReadError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockVariableListService{
		readErr: fmt.Errorf("workspace not found"),
	}

	cmd := newCmdVariableListWith(func() (variableListService, error) {
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
