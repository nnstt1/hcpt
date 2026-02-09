package project

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

type mockProjectService struct {
	projects []*tfe.Project
	err      error
	listFn   func(opts *tfe.ProjectListOptions) (*tfe.ProjectList, error)
}

func (m *mockProjectService) ListProjects(_ context.Context, _ string, opts *tfe.ProjectListOptions) (*tfe.ProjectList, error) {
	if m.listFn != nil {
		return m.listFn(opts)
	}
	if m.err != nil {
		return nil, m.err
	}
	return &tfe.ProjectList{
		Items: m.projects,
	}, nil
}

func TestProjectList_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockProjectService{
		projects: []*tfe.Project{
			{
				Name:        "Default Project",
				ID:          "prj-abc123",
				Description: "The default project",
			},
			{
				Name:        "Production",
				ID:          "prj-def456",
				Description: "Production workspaces",
			},
		},
	}

	cmd := newCmdProjectListWith(func() (client.ProjectService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectList_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockProjectService{
		projects: []*tfe.Project{
			{
				Name:        "Default Project",
				ID:          "prj-abc123",
				Description: "The default project",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjectList(mock, "test-org")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"NAME", "ID", "DESCRIPTION", "Default Project", "prj-abc123", "The default project"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestProjectList_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockProjectService{
		projects: []*tfe.Project{
			{
				Name:        "Default Project",
				ID:          "prj-abc123",
				Description: "The default project",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjectList(mock, "test-org")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"name": "Default Project"`, `"id": "prj-abc123"`, `"description": "The default project"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestProjectList_Pagination(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockProjectService{
		listFn: func(opts *tfe.ProjectListOptions) (*tfe.ProjectList, error) {
			page := opts.PageNumber
			if page == 0 {
				page = 1
			}
			switch page {
			case 1:
				return &tfe.ProjectList{
					Items: []*tfe.Project{
						{Name: "proj-1", ID: "prj-1", Description: "First"},
					},
					Pagination: &tfe.Pagination{NextPage: 2, TotalPages: 2},
				}, nil
			case 2:
				return &tfe.ProjectList{
					Items: []*tfe.Project{
						{Name: "proj-2", ID: "prj-2", Description: "Second"},
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

	err := runProjectList(mock, "test-org")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"proj-1", "proj-2"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestProjectList_Empty(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockProjectService{
		projects: []*tfe.Project{},
	}

	cmd := newCmdProjectListWith(func() (client.ProjectService, error) {
		return mock, nil
	})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectList_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdProjectListWith(func() (client.ProjectService, error) {
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

func TestProjectList_Error(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockProjectService{
		err: fmt.Errorf("api error"),
	}

	cmd := newCmdProjectListWith(func() (client.ProjectService, error) {
		return mock, nil
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
	if !strings.Contains(err.Error(), "api error") {
		t.Errorf("expected error containing 'api error', got: %v", err)
	}
}

func TestProjectList_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdProjectListWith(func() (client.ProjectService, error) {
		return &mockProjectService{}, nil
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
