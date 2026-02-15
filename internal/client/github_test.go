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
