# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A CLI tool to retrieve HCP Terraform configurations and workspace information.

## Tech Stack

- **Language**: Go
- **Module name**: `github.com/nnstt1/hcpt`
- **CLI framework**: [Cobra](https://github.com/spf13/cobra)
- **Configuration management**: [Viper](https://github.com/spf13/viper)
- **API clients**:
  - [go-tfe](https://github.com/hashicorp/go-tfe) (Official Go client for HCP Terraform / Terraform Enterprise)
  - [go-github](https://github.com/google/go-github) (GitHub API v3 client)
- **Linter**: golangci-lint
- **Release**: GoReleaser
- **CI**: GitHub Actions

## Command Structure

```
hcpt
├── org list          # List organizations
├── org show          # Show organization details, contract plan, and entitlements
├── project list      # List projects within an organization
├── drift list        # List drifted workspaces (--all for all results)
├── drift show        # Show drift detection details for a specific workspace
├── workspace list    # List workspaces within an organization
├── workspace show    # Show details of a specific workspace
├── run list          # Show run history for a workspace (filterable with --status)
├── run show          # Show run details (--watch to monitor until completion)
├── variable list     # List variables in a workspace
├── variable set      # Create/update a variable (upsert)
├── variable delete   # Delete a variable
├── config set        # Save a configuration value
├── config get        # Get a configuration value
└── config list       # List all configuration values
```

## Output Format

- Default: table format
- `--json` flag: JSON format

## Authentication

- API token is read from the `TFE_TOKEN` environment variable or `~/.hcpt.yaml` config file
- Viper manages priority between environment variables and config file (env vars > config file)

## GitHub Integration

- `--pr` and `--repo` flags added to the `run show` command
- Automatically retrieves HCP Terraform run-id from GitHub PR commit status
- Token resolution order: `gh auth token` → `GITHUB_TOKEN` env var → `github-token` in `~/.hcpt.yaml`
- Use `--workspace` to specify a workspace when multiple are present

## Git Branch Workflow

- For issues, create a `feature/<issue-number>-<summary>` branch and work there
- After implementation, create a PR and merge into the main branch
- Do not commit or push directly to the main branch

## Build & Development Commands

```bash
# Build
go build -o hcpt .

# Test
go test ./...

# Test a single package
go test ./internal/cmd/workspace/

# Run a single test
go test ./internal/cmd/workspace/ -run TestWorkspaceList

# Lint
golangci-lint run

# Tidy dependencies
go mod tidy
```

## Architecture

```
├── main.go                  # Entry point
├── internal/
│   ├── cmd/                 # Cobra command definitions
│   │   ├── root.go          # Root command (includes Viper initialization)
│   │   ├── drift/           # drift subcommand
│   │   ├── org/             # org subcommand
│   │   ├── workspace/       # workspace subcommand
│   │   └── run/             # run subcommand
│   ├── client/              # Wrapper for go-tfe client
│   └── output/              # Formatter for table/JSON output
├── .golangci.yml            # golangci-lint configuration
├── .goreleaser.yml          # GoReleaser configuration
└── .github/workflows/       # GitHub Actions CI
```

- Code is placed under `internal/` to prevent imports from external packages
- Each subcommand is isolated in its own directory; command registration is done in `init()`
- `internal/client/` centralizes go-tfe client initialization and token retrieval
- `internal/output/` provides shared logic for switching between table and JSON output
