package client

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/google/go-github/v82/github"
	"github.com/spf13/viper"
)

// DetectGitHubRepository detects GitHub repository (owner/repo) from current directory's Git remote.
// It prioritizes 'origin' remote if multiple remotes exist.
// Returns owner/repo format or an error if not a Git repo or GitHub remote not found.
func DetectGitHubRepository() (string, error) {
	// Try to get origin remote URL first
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		// If origin doesn't exist, try to get any remote
		cmd = exec.Command("git", "remote")
		remotesOutput, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git repository not found in current directory\nPlease specify repository using --repo flag (e.g., --repo owner/repo)")
		}

		remotes := strings.Fields(string(remotesOutput))
		if len(remotes) == 0 {
			return "", fmt.Errorf("no git remote found in current directory\nPlease specify repository using --repo flag (e.g., --repo owner/repo)")
		}

		// Get URL of the first remote
		cmd = exec.Command("git", "remote", "get-url", remotes[0])
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get git remote URL\nPlease specify repository using --repo flag (e.g., --repo owner/repo)")
		}
	}

	remoteURL := strings.TrimSpace(string(output))
	return parseGitHubRepository(remoteURL)
}

// parseGitHubRepository parses GitHub repository (owner/repo) from a Git remote URL.
// Supports both SSH (git@github.com:owner/repo.git) and HTTPS (https://github.com/owner/repo.git) formats.
func parseGitHubRepository(remoteURL string) (string, error) {
	// SSH format: git@github.com:owner/repo.git
	sshPattern := regexp.MustCompile(`^git@github\.com:([^/]+)/(.+?)(\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(remoteURL); matches != nil {
		owner := matches[1]
		repo := strings.TrimSuffix(matches[2], ".git")
		return fmt.Sprintf("%s/%s", owner, repo), nil
	}

	// HTTPS format: https://github.com/owner/repo.git or https://github.com/owner/repo
	httpsPattern := regexp.MustCompile(`^https://github\.com/([^/]+)/(.+?)(\.git)?$`)
	if matches := httpsPattern.FindStringSubmatch(remoteURL); matches != nil {
		owner := matches[1]
		repo := strings.TrimSuffix(matches[2], ".git")
		return fmt.Sprintf("%s/%s", owner, repo), nil
	}

	// Not a GitHub remote
	return "", fmt.Errorf("git remote is not a GitHub repository: %s\nPlease specify repository using --repo flag (e.g., --repo owner/repo)", remoteURL)
}

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
