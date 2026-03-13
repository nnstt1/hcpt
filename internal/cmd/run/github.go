package run

import (
	"context"
	"strings"

	"github.com/nnstt1/hcpt/internal/client"
)

// resolveRunIDFromPR fetches the HCP Terraform run-id from a GitHub pull request's commit statuses.
func resolveRunIDFromPR(ctx context.Context, repoFullName string, prNumber int, workspaceName string) (string, error) {
	ghClient, err := client.NewGitHubClientWrapper()
	if err != nil {
		return "", err
	}

	parts := strings.Split(repoFullName, "/")
	owner, repo := parts[0], parts[1]

	return ghClient.GetRunIDFromPR(ctx, owner, repo, prNumber, workspaceName)
}
