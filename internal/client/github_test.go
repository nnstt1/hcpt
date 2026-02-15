package client

import (
	"context"
	"testing"
)

// mockGitHubService is a mock implementation of GitHubService for testing.
type mockGitHubService struct {
	runID string
	err   error
}

func (m *mockGitHubService) GetRunIDFromPR(_ context.Context, _, _ string, _ int, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.runID, nil
}

// TestGetRunIDFromPR tests the basic functionality of GetRunIDFromPR.
// Note: This test requires actual GitHub API access, so we only test the mock here.
func TestGetRunIDFromPR(t *testing.T) {
	tests := []struct {
		name          string
		runID         string
		err           error
		expectedRunID string
		expectError   bool
	}{
		{
			name:          "successful retrieval",
			runID:         "run-abc123",
			err:           nil,
			expectedRunID: "run-abc123",
			expectError:   false,
		},
		{
			name:          "API error",
			runID:         "",
			err:           context.DeadlineExceeded,
			expectedRunID: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockGitHubService{
				runID: tt.runID,
				err:   tt.err,
			}

			runID, err := mock.GetRunIDFromPR(context.Background(), "owner", "repo", 1, "")

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if runID != tt.expectedRunID {
				t.Errorf("expected run ID %q, got %q", tt.expectedRunID, runID)
			}
		})
	}
}

// TestParseGitHubRepository tests the parseGitHubRepository function.
func TestParseGitHubRepository(t *testing.T) {
	tests := []struct {
		name         string
		remoteURL    string
		expectedRepo string
		expectError  bool
	}{
		{
			name:         "SSH format with .git",
			remoteURL:    "git@github.com:nnstt1/hcpt.git",
			expectedRepo: "nnstt1/hcpt",
			expectError:  false,
		},
		{
			name:         "SSH format without .git",
			remoteURL:    "git@github.com:owner/repo",
			expectedRepo: "owner/repo",
			expectError:  false,
		},
		{
			name:         "HTTPS format with .git",
			remoteURL:    "https://github.com/nnstt1/hcpt.git",
			expectedRepo: "nnstt1/hcpt",
			expectError:  false,
		},
		{
			name:         "HTTPS format without .git",
			remoteURL:    "https://github.com/owner/repo",
			expectedRepo: "owner/repo",
			expectError:  false,
		},
		{
			name:         "GitLab SSH",
			remoteURL:    "git@gitlab.com:owner/repo.git",
			expectedRepo: "",
			expectError:  true,
		},
		{
			name:         "GitLab HTTPS",
			remoteURL:    "https://gitlab.com/owner/repo.git",
			expectedRepo: "",
			expectError:  true,
		},
		{
			name:         "Bitbucket SSH",
			remoteURL:    "git@bitbucket.org:owner/repo.git",
			expectedRepo: "",
			expectError:  true,
		},
		{
			name:         "invalid format",
			remoteURL:    "not-a-valid-url",
			expectedRepo: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := parseGitHubRepository(tt.remoteURL)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if repo != tt.expectedRepo {
					t.Errorf("expected repo %q, got %q", tt.expectedRepo, repo)
				}
			}
		})
	}
}
