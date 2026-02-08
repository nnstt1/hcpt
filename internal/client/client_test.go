package client_test

import (
	"testing"

	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

func TestNewClientWrapper_NoToken(t *testing.T) {
	viper.Reset()
	viper.Set("token", "")

	_, err := client.NewClientWrapper()
	if err == nil {
		t.Fatal("expected error when token is empty, got nil")
	}

	expected := "API token is required"
	if got := err.Error(); !contains(got, expected) {
		t.Errorf("expected error containing %q, got %q", expected, got)
	}
}

func TestNewClientWrapper_WithToken(t *testing.T) {
	viper.Reset()
	viper.Set("token", "test-token")
	viper.Set("address", "https://app.terraform.io")

	cw, err := client.NewClientWrapper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cw == nil {
		t.Fatal("expected non-nil ClientWrapper")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr)
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
