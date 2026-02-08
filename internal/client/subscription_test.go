package client

import (
	"strings"
	"testing"
)

func TestParseSubscriptionResponse_FullResponse(t *testing.T) {
	// Based on actual HCP Terraform API response
	body := []byte(`{
		"data": {
			"id": "sub-123",
			"type": "subscriptions",
			"attributes": {
				"is-active": true,
				"is-public-free-tier": true
			},
			"relationships": {
				"feature-set": {
					"data": {"id": "fs-abc", "type": "feature-sets"}
				}
			}
		},
		"included": [
			{
				"id": "fs-abc",
				"type": "feature-sets",
				"attributes": {
					"name": "Free",
					"is-free-tier": true,
					"is-current": true
				}
			}
		]
	}`)

	info, err := parseSubscriptionResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PlanName != "Free" {
		t.Errorf("expected plan name %q, got %q", "Free", info.PlanName)
	}
	if !info.IsFreeTier {
		t.Error("expected IsFreeTier to be true")
	}
	if !info.IsActive {
		t.Error("expected IsActive to be true")
	}
}

func TestParseSubscriptionResponse_PaidPlan(t *testing.T) {
	body := []byte(`{
		"data": {
			"id": "sub-456",
			"type": "subscriptions",
			"attributes": {
				"is-active": true,
				"is-public-free-tier": false
			},
			"relationships": {
				"feature-set": {
					"data": {"id": "fs-def", "type": "feature-sets"}
				}
			}
		},
		"included": [
			{
				"id": "fs-def",
				"type": "feature-sets",
				"attributes": {
					"name": "Business",
					"is-free-tier": false,
					"is-current": true
				}
			}
		]
	}`)

	info, err := parseSubscriptionResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PlanName != "Business" {
		t.Errorf("expected plan name %q, got %q", "Business", info.PlanName)
	}
	if info.IsFreeTier {
		t.Error("expected IsFreeTier to be false")
	}
	if !info.IsActive {
		t.Error("expected IsActive to be true")
	}
}

func TestParseSubscriptionResponse_FallbackToID(t *testing.T) {
	body := []byte(`{
		"data": {
			"id": "sub-123",
			"type": "subscriptions",
			"attributes": {
				"is-active": true,
				"is-public-free-tier": false
			},
			"relationships": {
				"feature-set": {
					"data": {"id": "fs-abc", "type": "feature-sets"}
				}
			}
		}
	}`)

	info, err := parseSubscriptionResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PlanName != "fs-abc" {
		t.Errorf("expected plan name %q, got %q", "fs-abc", info.PlanName)
	}
}

func TestParseSubscriptionResponse_NoPlanInfo(t *testing.T) {
	body := []byte(`{
		"data": {
			"id": "sub-123",
			"type": "subscriptions",
			"attributes": {},
			"relationships": {
				"feature-set": {
					"data": {}
				}
			}
		}
	}`)

	_, err := parseSubscriptionResponse(body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "did not contain plan information") {
		t.Errorf("expected 'did not contain plan information' error, got: %v", err)
	}
}

func TestParseSubscriptionResponse_InvalidJSON(t *testing.T) {
	body := []byte(`invalid json`)

	_, err := parseSubscriptionResponse(body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected 'failed to parse' error, got: %v", err)
	}
}
