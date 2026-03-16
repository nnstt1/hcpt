---
name: hcpt-drift-analysis
description: Analyze and resolve infrastructure drift in HCP Terraform workspaces using hcpt CLI. Use when investigating drifted resources or reviewing drift detection results.
---

# HCP Terraform Drift Analysis

This skill guides you through detecting, analyzing, and resolving infrastructure drift using the `hcpt` CLI tool.

## Prerequisites

- `hcpt` CLI installed and configured with a valid HCP Terraform token
- Organization name configured (`hcpt config set org <org-name>`)
- Drift detection (assessments) enabled on the organization or workspace

## Drift Analysis Workflow

### Step 1: List Drifted Workspaces

Get all workspaces with detected drift:

```bash
hcpt drift list --org <org>
```

To include all workspaces (including non-drifted):

```bash
hcpt drift list --org <org> --all
```

### Step 2: Inspect Drift Details

For a specific workspace, view which resources have drifted:

```bash
hcpt drift show --org <org> --workspace <name>
```

Use JSON output for programmatic analysis:

```bash
hcpt drift show --org <org> --workspace <name> --json
```

### Step 3: Analyze Drift Impact

From the drift show output, review:

1. **Resource type and name**: Identify what drifted
2. **Changed attributes**: Understand what was modified outside Terraform
3. **Action type**: Whether the drift would result in an update or recreate

### Step 4: Resolve Drift

Options for resolving drift:

1. **Re-apply Terraform**: If the Terraform config is the source of truth, trigger a new run to bring infrastructure back in line
2. **Update Terraform config**: If the manual change was intentional, update the Terraform code to match the current state
3. **Import state**: If new resources were created outside Terraform, import them

### Step 5: Verify Resolution

After resolving drift, confirm the workspace is clean:

```bash
hcpt drift show --org <org> --workspace <name>
```

## Bulk Analysis with JSON

For analyzing drift across many workspaces programmatically:

```bash
hcpt drift list --org <org> --json
```

This returns structured data including:
- Workspace name
- Drift status
- Number of drifted/undrifted resources

## Tips

- Drift detection runs periodically via HCP Terraform assessments
- Not all workspaces have drift detection enabled; check organization settings
- Use `hcpt workspace show` to verify workspace configuration
- The `--json` flag is useful for piping output to other tools like `jq`
