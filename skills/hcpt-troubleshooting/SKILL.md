---
name: hcpt-troubleshooting
description: Debug HCP Terraform run failures using hcpt CLI. Use when a Terraform run fails, a PR check is red, or you need to investigate workspace errors.
---

# HCP Terraform Troubleshooting

This skill guides you through debugging failed HCP Terraform runs using the `hcpt` CLI tool.

## Prerequisites

- `hcpt` CLI installed and configured with a valid HCP Terraform token
- Organization name configured (`hcpt config set org <org-name>`)

## Debugging Workflow

### Step 1: Identify the Failed Run

If you have a GitHub PR with a failed Terraform check:

```bash
hcpt run show --repo <owner/repo> --pr <number>
```

If multiple workspaces are linked, specify which one:

```bash
hcpt run show --repo <owner/repo> --pr <number> --workspace <name>
```

If you already know the run ID:

```bash
hcpt run show --org <org> --run <run-id>
```

### Step 2: Check Error Logs

Extract only the error lines from the apply/plan logs:

```bash
hcpt run logs --org <org> --run <run-id> --error-only
```

For full logs:

```bash
hcpt run logs --org <org> --run <run-id>
```

### Step 3: Review Workspace Configuration

Check workspace settings for misconfiguration:

```bash
hcpt workspace show --org <org> --workspace <name>
```

### Step 4: Check Variables

List workspace variables to verify values:

```bash
hcpt variable list --org <org> --workspace <name>
```

Update a variable if needed:

```bash
hcpt variable set --org <org> --workspace <name> --key <key> --value <value>
```

### Step 5: Review Run History

Check recent runs to identify patterns:

```bash
hcpt run list --org <org> --workspace <name>
```

Filter by status to find all failed runs:

```bash
hcpt run list --org <org> --workspace <name> --status errored
```

## Common Failure Patterns

- **Authentication errors**: Check `TFE_TOKEN` or cloud provider credentials in variables
- **State lock**: Another run may be in progress; check run list for active runs
- **Provider version mismatch**: Review workspace Terraform version setting
- **Missing variables**: Compare required variables with `hcpt variable list` output

## Tips

- Use `--json` flag on any command for machine-readable output
- Use `hcpt run show --watch` to monitor a running plan/apply until completion
- Use `hcpt config list` to verify your current configuration
