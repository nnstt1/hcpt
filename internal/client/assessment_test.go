package client

import (
	"strings"
	"testing"
	"time"
)

func TestParseAssessmentResponse_Drifted(t *testing.T) {
	body := []byte(`{
		"data": {
			"id": "asmnt-abc123",
			"type": "assessment-results",
			"attributes": {
				"drifted": true,
				"succeeded": true,
				"resources-drifted": 3,
				"resources-undrifted": 12,
				"created-at": "2025-01-20T10:30:00.000Z"
			}
		}
	}`)

	result, err := parseAssessmentResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Drifted {
		t.Error("expected Drifted to be true")
	}
	if !result.Succeeded {
		t.Error("expected Succeeded to be true")
	}
	if result.ResourcesDrifted != 3 {
		t.Errorf("expected ResourcesDrifted=3, got %d", result.ResourcesDrifted)
	}
	if result.ResourcesUndrifted != 12 {
		t.Errorf("expected ResourcesUndrifted=12, got %d", result.ResourcesUndrifted)
	}
	if result.CreatedAt != "2025-01-20T10:30:00.000Z" {
		t.Errorf("expected CreatedAt=%q, got %q", "2025-01-20T10:30:00.000Z", result.CreatedAt)
	}
}

func TestParseAssessmentResponse_NoDrift(t *testing.T) {
	body := []byte(`{
		"data": {
			"id": "asmnt-def456",
			"type": "assessment-results",
			"attributes": {
				"drifted": false,
				"succeeded": true,
				"resources-drifted": 0,
				"resources-undrifted": 15,
				"created-at": "2025-01-20T10:30:00.000Z"
			}
		}
	}`)

	result, err := parseAssessmentResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Drifted {
		t.Error("expected Drifted to be false")
	}
	if result.ResourcesDrifted != 0 {
		t.Errorf("expected ResourcesDrifted=0, got %d", result.ResourcesDrifted)
	}
	if result.ResourcesUndrifted != 15 {
		t.Errorf("expected ResourcesUndrifted=15, got %d", result.ResourcesUndrifted)
	}
}

func TestParseAssessmentResponse_InvalidJSON(t *testing.T) {
	body := []byte(`invalid json`)

	_, err := parseAssessmentResponse(body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected 'failed to parse' error, got: %v", err)
	}
}

func TestParseAssessmentResponse_EmptyBody(t *testing.T) {
	body := []byte(`{}`)

	result, err := parseAssessmentResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Drifted {
		t.Error("expected Drifted to be false for empty response")
	}
	if result.ResourcesDrifted != 0 {
		t.Errorf("expected ResourcesDrifted=0, got %d", result.ResourcesDrifted)
	}
}

func TestRetryAfterDuration_ValidHeader(t *testing.T) {
	got := retryAfterDuration("5", 0)
	if got != 5*time.Second {
		t.Errorf("expected 5s, got %v", got)
	}
}

func TestRetryAfterDuration_InvalidHeader_FallbackExponential(t *testing.T) {
	tests := []struct {
		header  string
		attempt int
		want    time.Duration
	}{
		{"", 0, 1 * time.Second},
		{"", 1, 2 * time.Second},
		{"", 2, 4 * time.Second},
		{"invalid", 0, 1 * time.Second},
		{"invalid", 1, 2 * time.Second},
	}
	for _, tt := range tests {
		got := retryAfterDuration(tt.header, tt.attempt)
		if got != tt.want {
			t.Errorf("retryAfterDuration(%q, %d) = %v, want %v", tt.header, tt.attempt, got, tt.want)
		}
	}
}

func TestRetryAfterDuration_ZeroHeader(t *testing.T) {
	// "0" is not a valid positive value, should fall back to exponential
	got := retryAfterDuration("0", 0)
	if got != 1*time.Second {
		t.Errorf("expected 1s fallback, got %v", got)
	}
}

func TestRetryAfterDuration_NegativeHeader(t *testing.T) {
	got := retryAfterDuration("-1", 0)
	if got != 1*time.Second {
		t.Errorf("expected 1s fallback for negative header, got %v", got)
	}
}

func TestParseExplorerWorkspacesResponse(t *testing.T) {
	body := []byte(`{
		"data": [
			{
				"type": "workspace-summaries",
				"attributes": {
					"workspace-name": "prod-vpc",
					"external-id": "ws-abc123",
					"workspace-terraform-version": "1.5.0",
					"current-run-status": "applied",
					"project-name": "production",
					"workspace-updated-at": "2024-03-15T12:00:00Z",
					"drifted": true,
					"resources-drifted": 3,
					"resources-undrifted": 12
				}
			},
			{
				"type": "workspace-summaries",
				"attributes": {
					"workspace-name": "staging",
					"external-id": "ws-def456",
					"workspace-terraform-version": "1.6.0",
					"current-run-status": "planned_and_finished",
					"project-name": "staging",
					"workspace-updated-at": "2024-03-10T08:00:00Z",
					"drifted": false,
					"resources-drifted": 0,
					"resources-undrifted": 20
				}
			}
		],
		"meta": {
			"pagination": {
				"total-pages": 2,
				"current-page": 1,
				"next-page": 2
			}
		}
	}`)

	result, err := parseExplorerWorkspacesResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].WorkspaceName != "prod-vpc" {
		t.Errorf("expected workspace name 'prod-vpc', got %q", result.Items[0].WorkspaceName)
	}
	if result.Items[0].WorkspaceID != "ws-abc123" {
		t.Errorf("expected WorkspaceID 'ws-abc123', got %q", result.Items[0].WorkspaceID)
	}
	if result.Items[0].TerraformVersion != "1.5.0" {
		t.Errorf("expected TerraformVersion '1.5.0', got %q", result.Items[0].TerraformVersion)
	}
	if result.Items[0].CurrentRunStatus != "applied" {
		t.Errorf("expected CurrentRunStatus 'applied', got %q", result.Items[0].CurrentRunStatus)
	}
	if result.Items[0].ProjectName != "production" {
		t.Errorf("expected ProjectName 'production', got %q", result.Items[0].ProjectName)
	}
	if result.Items[0].UpdatedAt != "2024-03-15T12:00:00Z" {
		t.Errorf("expected UpdatedAt '2024-03-15T12:00:00Z', got %q", result.Items[0].UpdatedAt)
	}
	if !result.Items[0].Drifted {
		t.Error("expected prod-vpc to be drifted")
	}
	if result.Items[0].ResourcesDrifted != 3 {
		t.Errorf("expected ResourcesDrifted=3, got %d", result.Items[0].ResourcesDrifted)
	}
	if result.Items[1].Drifted {
		t.Error("expected staging to not be drifted")
	}
	if result.Items[1].WorkspaceID != "ws-def456" {
		t.Errorf("expected WorkspaceID 'ws-def456', got %q", result.Items[1].WorkspaceID)
	}
	if result.TotalPages != 2 {
		t.Errorf("expected TotalPages=2, got %d", result.TotalPages)
	}
	if result.NextPage != 2 {
		t.Errorf("expected NextPage=2, got %d", result.NextPage)
	}
}

func TestParseExplorerWorkspacesResponse_Empty(t *testing.T) {
	body := []byte(`{"data": [], "meta": {"pagination": {"total-pages": 0, "current-page": 1, "next-page": 0}}}`)

	result, err := parseExplorerWorkspacesResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestParseExplorerWorkspacesResponse_InvalidJSON(t *testing.T) {
	_, err := parseExplorerWorkspacesResponse([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected 'failed to parse' error, got: %v", err)
	}
}

func TestParseAssessmentJSONOutput_MultipleResources(t *testing.T) {
	body := []byte(`{
		"resource_drift": [
			{
				"address": "aws_security_group.web",
				"type": "aws_security_group",
				"name": "web",
				"change": {"actions": ["update"]}
			},
			{
				"address": "aws_s3_bucket.logs",
				"type": "aws_s3_bucket",
				"name": "logs",
				"change": {"actions": ["delete", "create"]}
			}
		]
	}`)

	resources, err := parseAssessmentJSONOutput(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].Address != "aws_security_group.web" {
		t.Errorf("expected address 'aws_security_group.web', got %q", resources[0].Address)
	}
	if resources[0].Type != "aws_security_group" {
		t.Errorf("expected type 'aws_security_group', got %q", resources[0].Type)
	}
	if resources[0].Name != "web" {
		t.Errorf("expected name 'web', got %q", resources[0].Name)
	}
	if resources[0].Action != "update" {
		t.Errorf("expected action 'update', got %q", resources[0].Action)
	}
	if resources[1].Action != "delete, create" {
		t.Errorf("expected action 'delete, create', got %q", resources[1].Action)
	}
}

func TestParseAssessmentJSONOutput_NoResourceDrift(t *testing.T) {
	body := []byte(`{"resource_drift": []}`)

	resources, err := parseAssessmentJSONOutput(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}
}

func TestParseAssessmentJSONOutput_MissingField(t *testing.T) {
	body := []byte(`{}`)

	resources, err := parseAssessmentJSONOutput(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}
}

func TestParseAssessmentJSONOutput_EmptyActions(t *testing.T) {
	body := []byte(`{
		"resource_drift": [
			{
				"address": "aws_instance.test",
				"type": "aws_instance",
				"name": "test",
				"change": {"actions": []}
			}
		]
	}`)

	resources, err := parseAssessmentJSONOutput(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].Action != "unknown" {
		t.Errorf("expected action 'unknown' for empty actions, got %q", resources[0].Action)
	}
}

func TestParseAssessmentJSONOutput_InvalidJSON(t *testing.T) {
	_, err := parseAssessmentJSONOutput([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected 'failed to parse' error, got: %v", err)
	}
}
