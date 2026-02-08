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

type mockOrgShowService struct {
	org             *tfe.Organization
	orgErr          error
	subscription    *client.SubscriptionInfo
	subscriptionErr error
	entitlements    *tfe.Entitlements
	entitlementsErr error
}

func (m *mockOrgShowService) ListOrganizations(_ context.Context, _ *tfe.OrganizationListOptions) (*tfe.OrganizationList, error) {
	return nil, nil
}

func (m *mockOrgShowService) ReadOrganization(_ context.Context, _ string) (*tfe.Organization, error) {
	if m.orgErr != nil {
		return nil, m.orgErr
	}
	return m.org, nil
}

func (m *mockOrgShowService) ReadSubscription(_ context.Context, _ string) (*client.SubscriptionInfo, error) {
	if m.subscriptionErr != nil {
		return nil, m.subscriptionErr
	}
	return m.subscription, nil
}

func (m *mockOrgShowService) ReadEntitlements(_ context.Context, _ string) (*tfe.Entitlements, error) {
	if m.entitlementsErr != nil {
		return nil, m.entitlementsErr
	}
	return m.entitlements, nil
}

func TestOrgShow_Table(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockOrgShowService{
		org: &tfe.Organization{
			Name:      "test-org",
			Email:     "admin@example.com",
			CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		subscription: &client.SubscriptionInfo{
			PlanName: "Free",
			IsActive: true,
		},
		entitlements: &tfe.Entitlements{
			Teams:    true,
			Sentinel: true,
		},
	}

	cmd := newCmdOrgShowWith(func() (orgShowService, error) {
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

func TestOrgShow_Table_Output(t *testing.T) {
	viper.Reset()
	viper.Set("json", false)
	viper.Set("org", "test-org")

	mock := &mockOrgShowService{
		org: &tfe.Organization{
			Name:      "test-org",
			Email:     "admin@example.com",
			CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		subscription: &client.SubscriptionInfo{
			PlanName: "Standard",
			IsActive: true,
		},
		entitlements: &tfe.Entitlements{
			Teams:           true,
			Sentinel:        true,
			SSO:             false,
			VCSIntegrations: true,
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runOrgShow(mock, "test-org")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{"Name:", "test-org", "Email:", "admin@example.com", "Plan:", "Standard", "Teams:", "true", "Sentinel:", "true", "SSO:", "false"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestOrgShow_JSON(t *testing.T) {
	viper.Reset()
	viper.Set("json", true)
	viper.Set("org", "test-org")

	mock := &mockOrgShowService{
		org: &tfe.Organization{
			Name:      "test-org",
			Email:     "admin@example.com",
			CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		subscription: &client.SubscriptionInfo{
			PlanName: "Free",
			IsActive: true,
		},
		entitlements: &tfe.Entitlements{
			Sentinel: true,
			Teams:    true,
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runOrgShow(mock, "test-org")

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	for _, want := range []string{`"name": "test-org"`, `"email": "admin@example.com"`, `"plan": "Free"`, `"sentinel": true`, `"teams": true`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, got)
		}
	}
}

func TestOrgShow_NoOrg(t *testing.T) {
	viper.Reset()

	cmd := newCmdOrgShowWith(func() (orgShowService, error) {
		return &mockOrgShowService{}, nil
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

func TestOrgShow_ClientError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	cmd := newCmdOrgShowWith(func() (orgShowService, error) {
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

func TestOrgShow_ReadOrgError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockOrgShowService{
		orgErr: fmt.Errorf("org not found"),
	}

	cmd := newCmdOrgShowWith(func() (orgShowService, error) {
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
	if !strings.Contains(err.Error(), "org not found") {
		t.Errorf("expected 'org not found' error, got: %v", err)
	}
}

func TestOrgShow_ReadSubscriptionError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockOrgShowService{
		org: &tfe.Organization{
			Name:      "test-org",
			Email:     "admin@example.com",
			CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		subscriptionErr: fmt.Errorf("subscription error"),
	}

	cmd := newCmdOrgShowWith(func() (orgShowService, error) {
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
	if !strings.Contains(err.Error(), "subscription error") {
		t.Errorf("expected 'subscription error', got: %v", err)
	}
}

func TestOrgShow_ReadEntitlementsError(t *testing.T) {
	viper.Reset()
	viper.Set("org", "test-org")

	mock := &mockOrgShowService{
		org: &tfe.Organization{
			Name:      "test-org",
			Email:     "admin@example.com",
			CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		subscription: &client.SubscriptionInfo{
			PlanName: "Free",
			IsActive: true,
		},
		entitlementsErr: fmt.Errorf("entitlements error"),
	}

	cmd := newCmdOrgShowWith(func() (orgShowService, error) {
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
	if !strings.Contains(err.Error(), "entitlements error") {
		t.Errorf("expected 'entitlements error', got: %v", err)
	}
}
