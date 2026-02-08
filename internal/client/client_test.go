package client_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/nnstt1/hcpt/internal/client"
)

func TestNewClientWrapper_NoToken(t *testing.T) {
	viper.Reset()
	viper.Set("token", "")

	// Ensure no Terraform credentials are found
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TF_CLI_CONFIG_FILE", filepath.Join(t.TempDir(), "nonexistent"))
	t.Setenv("TF_TOKEN_app_terraform_io", "")

	_, err := client.NewClientWrapper()
	if err == nil {
		t.Fatal("expected error when token is empty, got nil")
	}

	expected := "API token is required"
	if got := err.Error(); !contains(got, expected) {
		t.Errorf("expected error containing %q, got %q", expected, got)
	}

	// Check that error message mentions terraform login
	terraformLogin := "terraform login"
	if got := err.Error(); !contains(got, terraformLogin) {
		t.Errorf("expected error containing %q, got %q", terraformLogin, got)
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

func TestNewClientWrapper_FallbackToTerraformCredentials(t *testing.T) {
	viper.Reset()
	viper.Set("token", "")
	viper.Set("address", "https://app.terraform.io")

	// Set up credentials.tfrc.json
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".terraform.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{
  "credentials": {
    "app.terraform.io": {
      "token": "fallback-token"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "credentials.tfrc.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cw, err := client.NewClientWrapper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cw == nil {
		t.Fatal("expected non-nil ClientWrapper")
	}
}

func TestNewClientWrapper_FallbackToTFTokenEnv(t *testing.T) {
	viper.Reset()
	viper.Set("token", "")
	viper.Set("address", "https://app.terraform.io")

	t.Setenv("HOME", t.TempDir())
	t.Setenv("TF_TOKEN_app_terraform_io", "env-fallback-token")

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
