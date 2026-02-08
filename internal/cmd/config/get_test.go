package config

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestRunConfigGet(t *testing.T) {
	t.Run("get org value", func(t *testing.T) {
		viper.Reset()
		viper.Set("org", "my-org")

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runConfigGet("org")

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		got := strings.TrimSpace(buf.String())

		if got != "my-org" {
			t.Errorf("expected %q, got %q", "my-org", got)
		}
	})

	t.Run("get token value is masked", func(t *testing.T) {
		viper.Reset()
		viper.Set("token", "my-secret-token-1234")

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runConfigGet("token")

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		got := strings.TrimSpace(buf.String())

		if strings.Contains(got, "my-secret-token-1234") {
			t.Error("token should be masked, but full value was shown")
		}
		if !strings.Contains(got, "1234") {
			t.Errorf("expected masked token to end with '1234', got %q", got)
		}
		if !strings.Contains(got, "****") {
			t.Errorf("expected masked token to contain '****', got %q", got)
		}
	})

	t.Run("get empty value", func(t *testing.T) {
		viper.Reset()

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runConfigGet("org")

		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		got := strings.TrimSpace(buf.String())

		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		err := runConfigGet("invalid-key")
		if err == nil {
			t.Fatal("expected error for invalid key")
		}
		if !strings.Contains(err.Error(), "unknown config key") {
			t.Errorf("expected 'unknown config key' error, got: %v", err)
		}
	})
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-secret-token-1234", "****...1234"},
		{"abc", "****"},
		{"", "****"},
		{"abcd", "****"},
		{"abcde", "****...bcde"},
	}
	for _, tt := range tests {
		got := maskToken(tt.input)
		if got != tt.expected {
			t.Errorf("maskToken(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
