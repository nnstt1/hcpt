package org

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

	"github.com/nnstt1/hcpt/internal/client"
)

type mockOrgService struct {
	orgs   []*tfe.Organization
	err    error
	listFn func(opts *tfe.OrganizationListOptions) (*tfe.OrganizationList, error)
}

func (m *mockOrgService) ListOrganizations(_ context.Context, opts *tfe.OrganizationListOptions) (*tfe.OrganizationList, error) {
	if m.listFn != nil {
		return m.listFn(opts)
	}
	if m.err != nil {
		return nil, m.err
	}
	return &tfe.OrganizationList{
		Items: m.orgs,
	}, nil
}

func (m *mockOrgService) ReadOrganization(_ context.Context, _ string) (*tfe.Organization, error) {
	return nil, nil
}

func (m *mockOrgService) ReadEntitlements(_ context.Context, _ string) (*tfe.Entitlements, error) {
	return nil, nil
}

func TestOrgList_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockOrgService{
		orgs: []*tfe.Organization{
			{
				Name:      "my-org",
				Email:     "admin@example.com",
				CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			{
				Name:      "another-org",
				Email:     "user@example.com",
				CreatedAt: time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC),
			},
		},
	}

	cmd := newCmdOrgListWith(func() (client.OrganizationService, error) {
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

func TestOrgList_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)

	mock := &mockOrgService{
		orgs: []*tfe.Organization{
			{
				Name:  "my-org",
				Email: "admin@example.com",
			},
		},
	}

	cmd := newCmdOrgListWith(func() (client.OrganizationService, error) {
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

func TestOrgList_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockOrgService{
		orgs: []*tfe.Organization{
			{
				Name:      "my-org",
				Email:     "admin@example.com",
				CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runOrgList(mock)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, "NAME") {
		t.Errorf("expected header NAME in output, got:\n%s", got)
	}
	if !strings.Contains(got, "my-org") {
		t.Errorf("expected 'my-org' in output, got:\n%s", got)
	}
	if !strings.Contains(got, "admin@example.com") {
		t.Errorf("expected email in output, got:\n%s", got)
	}
	if !strings.Contains(got, "2024-01-15 10:30:00") {
		t.Errorf("expected formatted date in output, got:\n%s", got)
	}
}

func TestOrgList_JSON_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)

	mock := &mockOrgService{
		orgs: []*tfe.Organization{
			{
				Name:      "my-org",
				Email:     "admin@example.com",
				CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runOrgList(mock)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if !strings.Contains(got, `"name": "my-org"`) {
		t.Errorf("expected JSON name field, got:\n%s", got)
	}
	if !strings.Contains(got, `"email": "admin@example.com"`) {
		t.Errorf("expected JSON email field, got:\n%s", got)
	}
}

func TestOrgList_Pagination(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockOrgService{
		listFn: func(opts *tfe.OrganizationListOptions) (*tfe.OrganizationList, error) {
			page := opts.PageNumber
			if page == 0 {
				page = 1
			}
			switch page {
			case 1:
				return &tfe.OrganizationList{
					Items: []*tfe.Organization{
						{Name: "org-1", Email: "a@example.com", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
					},
					Pagination: &tfe.Pagination{NextPage: 2, TotalPages: 2},
				}, nil
			case 2:
				return &tfe.OrganizationList{
					Items: []*tfe.Organization{
						{Name: "org-2", Email: "b@example.com", CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
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

	err := runOrgList(mock)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"org-1", "org-2"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestOrgList_Empty(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)

	mock := &mockOrgService{
		orgs: []*tfe.Organization{},
	}

	cmd := newCmdOrgListWith(func() (client.OrganizationService, error) {
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

func TestOrgList_ClientError(t *testing.T) {
	viper.Reset()

	cmd := newCmdOrgListWith(func() (client.OrganizationService, error) {
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

func TestOrgList_Error(t *testing.T) {
	viper.Reset()

	mock := &mockOrgService{
		err: fmt.Errorf("api error"),
	}

	cmd := newCmdOrgListWith(func() (client.OrganizationService, error) {
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
