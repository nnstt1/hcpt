package skills

import (
	"testing"
	"testing/fstest"
)

func TestNewCmdSkills(t *testing.T) {
	fs := fstest.MapFS{
		"test-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: test-skill\ndescription: A test skill.\n---\n\n# Test Skill\n"),
		},
	}

	cmd := NewCmdSkills(fs)

	if cmd.Use != "skills" {
		t.Errorf("expected Use to be 'skills', got %q", cmd.Use)
	}
	if !cmd.DisableFlagParsing {
		t.Error("expected DisableFlagParsing to be true")
	}
	if !cmd.SilenceUsage {
		t.Error("expected SilenceUsage to be true")
	}

	// Test that the command runs successfully with "list" subcommand
	cmd.SetArgs([]string{"list"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSanitizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0.8.0", "0.8.0"},
		{"v0.8.0", "0.8.0"},
		{"v0.8.0 (abc1234)", "0.8.0"},
		{"v0.8.0 (abc1234) (modified)", "0.8.0"},
		{"0.8.2-0.20260313+dirty", "0.8.2-0.20260313"},
		{"dev", "0.0.0-dev"},
		{"dev (abc1234)", "0.0.0-dev"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeVersion(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
