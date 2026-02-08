package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

func TestRunConfigSet(t *testing.T) {
	t.Run("set org in new file", func(t *testing.T) {
		home := t.TempDir()
		configPath := filepath.Join(home, ".hcpt.yaml")
		viper.Reset()
		viper.SetConfigFile(configPath)

		if err := runConfigSet("org", "my-org"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}

		var got map[string]string
		if err := yaml.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got["org"] != "my-org" {
			t.Errorf("expected org=%q, got %q", "my-org", got["org"])
		}
	})

	t.Run("set org preserves existing values", func(t *testing.T) {
		home := t.TempDir()
		configPath := filepath.Join(home, ".hcpt.yaml")

		initial := map[string]string{"token": "existing-token"}
		data, _ := yaml.Marshal(initial)
		if err := os.WriteFile(configPath, data, 0o600); err != nil {
			t.Fatal(err)
		}

		viper.Reset()
		viper.SetConfigFile(configPath)

		if err := runConfigSet("org", "new-org"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}

		var got map[string]string
		if err := yaml.Unmarshal(result, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got["token"] != "existing-token" {
			t.Errorf("expected token=%q, got %q", "existing-token", got["token"])
		}
		if got["org"] != "new-org" {
			t.Errorf("expected org=%q, got %q", "new-org", got["org"])
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		err := runConfigSet("invalid-key", "value")
		if err == nil {
			t.Fatal("expected error for invalid key")
		}
	})

	t.Run("default config path when no config file used", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		viper.Reset()

		if err := runConfigSet("org", "default-path-org"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := filepath.Join(home, ".hcpt.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}

		var got map[string]string
		if err := yaml.Unmarshal(data, &got); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if got["org"] != "default-path-org" {
			t.Errorf("expected org=%q, got %q", "default-path-org", got["org"])
		}
	})
}
