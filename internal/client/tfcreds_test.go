package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindTokenFromEnv(t *testing.T) {
	t.Run("token found", func(t *testing.T) {
		t.Setenv("TF_TOKEN_app_terraform_io", "env-token-123")
		got := findTokenFromEnv("app.terraform.io")
		if got != "env-token-123" {
			t.Errorf("expected %q, got %q", "env-token-123", got)
		}
	})

	t.Run("token not set", func(t *testing.T) {
		got := findTokenFromEnv("app.terraform.io")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("hostname with dashes", func(t *testing.T) {
		t.Setenv("TF_TOKEN_my_tfe_example_com", "dash-token")
		got := findTokenFromEnv("my-tfe.example.com")
		if got != "dash-token" {
			t.Errorf("expected %q, got %q", "dash-token", got)
		}
	})
}

func TestFindTokenFromCredentialsJSON(t *testing.T) {
	t.Run("token found", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		dir := filepath.Join(home, ".terraform.d")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		content := `{
  "credentials": {
    "app.terraform.io": {
      "token": "json-token-456"
    }
  }
}`
		if err := os.WriteFile(filepath.Join(dir, "credentials.tfrc.json"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		got := findTokenFromCredentialsJSON("app.terraform.io")
		if got != "json-token-456" {
			t.Errorf("expected %q, got %q", "json-token-456", got)
		}
	})

	t.Run("hostname mismatch", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		dir := filepath.Join(home, ".terraform.d")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		content := `{
  "credentials": {
    "app.terraform.io": {
      "token": "json-token-456"
    }
  }
}`
		if err := os.WriteFile(filepath.Join(dir, "credentials.tfrc.json"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		got := findTokenFromCredentialsJSON("other.terraform.io")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		got := findTokenFromCredentialsJSON("app.terraform.io")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
}

func TestFindTokenFromTerraformRC(t *testing.T) {
	t.Run("token found", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), ".terraformrc")
		content := `credentials "app.terraform.io" {
  token = "hcl-token-789"
}
`
		if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("TF_CLI_CONFIG_FILE", tmpFile)

		got := findTokenFromTerraformRC("app.terraform.io")
		if got != "hcl-token-789" {
			t.Errorf("expected %q, got %q", "hcl-token-789", got)
		}
	})

	t.Run("multiple credentials blocks", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), ".terraformrc")
		content := `credentials "app.terraform.io" {
  token = "token-a"
}

credentials "tfe.example.com" {
  token = "token-b"
}
`
		if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("TF_CLI_CONFIG_FILE", tmpFile)

		got := findTokenFromTerraformRC("tfe.example.com")
		if got != "token-b" {
			t.Errorf("expected %q, got %q", "token-b", got)
		}
	})

	t.Run("hostname mismatch", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), ".terraformrc")
		content := `credentials "app.terraform.io" {
  token = "hcl-token-789"
}
`
		if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("TF_CLI_CONFIG_FILE", tmpFile)

		got := findTokenFromTerraformRC("other.host.com")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		t.Setenv("TF_CLI_CONFIG_FILE", "/nonexistent/.terraformrc")

		got := findTokenFromTerraformRC("app.terraform.io")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
}

func TestFindTerraformToken_Priority(t *testing.T) {
	// Set up credentials.tfrc.json
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".terraform.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonContent := `{
  "credentials": {
    "app.terraform.io": {
      "token": "json-token"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "credentials.tfrc.json"), []byte(jsonContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Set up .terraformrc
	tmpFile := filepath.Join(t.TempDir(), ".terraformrc")
	hclContent := `credentials "app.terraform.io" {
  token = "hcl-token"
}
`
	if err := os.WriteFile(tmpFile, []byte(hclContent), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TF_CLI_CONFIG_FILE", tmpFile)

	t.Run("env var takes priority", func(t *testing.T) {
		t.Setenv("TF_TOKEN_app_terraform_io", "env-token")
		got := findTerraformToken("app.terraform.io")
		if got != "env-token" {
			t.Errorf("expected %q, got %q", "env-token", got)
		}
	})

	t.Run("json takes priority over hcl", func(t *testing.T) {
		got := findTerraformToken("app.terraform.io")
		if got != "json-token" {
			t.Errorf("expected %q, got %q", "json-token", got)
		}
	})
}

func TestTerraformRCPath(t *testing.T) {
	t.Run("custom path from env", func(t *testing.T) {
		t.Setenv("TF_CLI_CONFIG_FILE", "/custom/path/.terraformrc")
		got := terraformRCPath()
		if got != "/custom/path/.terraformrc" {
			t.Errorf("expected %q, got %q", "/custom/path/.terraformrc", got)
		}
	})

	t.Run("default path", func(t *testing.T) {
		t.Setenv("TF_CLI_CONFIG_FILE", "")

		got := terraformRCPath()
		if !strings.HasSuffix(got, ".terraformrc") {
			t.Errorf("expected path ending with .terraformrc, got %q", got)
		}
	})
}

func TestHostnameFromAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{"https URL", "https://app.terraform.io", "app.terraform.io"},
		{"https URL with path", "https://tfe.example.com/api", "tfe.example.com"},
		{"empty address", "", "app.terraform.io"},
		{"URL with port", "https://tfe.example.com:8443", "tfe.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hostnameFromAddress(tt.address)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
