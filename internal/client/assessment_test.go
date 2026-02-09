package client

import (
	"strings"
	"testing"
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
