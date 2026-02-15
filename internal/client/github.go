package client

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/google/go-github/v68/github"
	"github.com/spf13/viper"
)

// GitHubService provides operations on GitHub repositories.
type GitHubService interface {
	GetRunIDFromPR(ctx context.Context, owner, repo string, prNumber int, workspaceName string) (string, error)
}

// GitHubClientWrapper wraps the go-github client.
type GitHubClientWrapper struct {
	client *github.Client
}

// NewGitHubClientWrapper creates a new GitHubClientWrapper using token from gh CLI, env, or config.
func NewGitHubClientWrapper() (*GitHubClientWrapper, error) {
	token := resolveGitHubToken()
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required: use 'gh auth token', GITHUB_TOKEN env, or 'github-token' in config file")
	}

	client := github.NewClient(nil).WithAuthToken(token)

	return &GitHubClientWrapper{client: client}, nil
}

// resolveGitHubToken resolves GitHub token from multiple sources.
// Priority: gh CLI > GITHUB_TOKEN env > config file.
func resolveGitHubToken() string {
	// 1. Try gh CLI
	cmd := exec.Command("gh", "auth", "token")
	if output, err := cmd.Output(); err == nil {
		token := strings.TrimSpace(string(output))
		if token != "" {
			return token
		}
	}

	// 2. Try GITHUB_TOKEN environment variable
	if token := viper.GetString("GITHUB_TOKEN"); token != "" {
		return token
	}

	// 3. Try config file
	return viper.GetString("github-token")
}

// GetRunIDFromPR retrieves the HCP Terraform run ID from a GitHub PR's commit statuses.
func (c *GitHubClientWrapper) GetRunIDFromPR(ctx context.Context, owner, repo string, prNumber int, workspaceName string) (string, error) {
	// Get PR details
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR #%d: %w", prNumber, err)
	}

	// Get commit statuses for the HEAD commit
	commitSHA := pr.GetHead().GetSHA()
	statuses, _, err := c.client.Repositories.ListStatuses(ctx, owner, repo, commitSHA, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get commit statuses for %s: %w", commitSHA, err)
	}

	// Extract run IDs from statuses (deduplicate by context, keeping the latest)
	runIDPattern := regexp.MustCompile(`app\.terraform\.io/.+/runs/(run-[A-Za-z0-9]+)`)
	contextToRunID := make(map[string]string) // context -> run-id (latest)

	for _, status := range statuses {
		targetURL := status.GetTargetURL()
		if matches := runIDPattern.FindStringSubmatch(targetURL); matches != nil {
			runID := matches[1]
			context := status.GetContext()
			// Only keep the first (latest) run-id for each context
			if _, exists := contextToRunID[context]; !exists {
				contextToRunID[context] = runID
			}
		}
	}

	if len(contextToRunID) == 0 {
		return "", fmt.Errorf("no HCP Terraform run found in PR #%d", prNumber)
	}

	// If workspace name is specified, filter by context
	if workspaceName != "" {
		for context, runID := range contextToRunID {
			if strings.Contains(context, workspaceName) {
				return runID, nil
			}
		}
		return "", fmt.Errorf("no HCP Terraform run found for workspace '%s' in PR #%d", workspaceName, prNumber)
	}

	// If only one run found, return it
	if len(contextToRunID) == 1 {
		for _, runID := range contextToRunID {
			return runID, nil
		}
	}

	// Multiple runs found, list contexts for user guidance
	contexts := make([]string, 0, len(contextToRunID))
	for context := range contextToRunID {
		contexts = append(contexts, fmt.Sprintf("  - %s", context))
	}
	return "", fmt.Errorf("multiple HCP Terraform runs found in PR #%d:\n%s\nUse --workspace/-w to specify which one", prNumber, strings.Join(contexts, "\n"))
}
