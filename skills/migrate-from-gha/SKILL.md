---
name: migrate-from-gha
description: Migrate a GitHub Actions workflow to RWX. Translates triggers, jobs, steps into an optimized RWX config with DAG parallelism, content-based caching, and RWX packages.
argument-hint: [path/to/.github/workflows/ci.yml]
allowed-tools: Bash(curl *)
---

## RWX Reference

!`curl -sL https://www.rwx.com/docs/rwx/migrating/rwx-reference.md`

## GHA-to-RWX Mapping Reference

!`curl -sL https://www.rwx.com/docs/rwx/migrating/gha-reference.md`

## Migration Procedure

You are migrating a GitHub Actions workflow to RWX. Follow these steps exactly.

### Step 1: Read the source workflow

Read the GitHub Actions workflow file at `$ARGUMENTS`. If no path is provided, look for
`.github/workflows/` and list the available workflow files for the user to choose from.

### Step 2: Analyze the workflow structure

Identify:
- All jobs and their `needs:` dependencies
- All steps within each job
- Triggers (`on:` events)
- Secrets referenced (`${{ secrets.* }}`)
- Environment variables (`env:` blocks at workflow, job, and step level)
- Matrix strategies
- Services
- Composite action references (`uses: ./.github/actions/*`)
- Reusable workflow calls (`uses: org/repo/.github/workflows/*`)
- Artifact upload/download steps
- Cache steps (these will be removed — RWX handles caching natively)

### Step 3: Follow local composite action references

For steps using `uses: ./.github/actions/foo`:
- Read that action's `action.yml`
- Inline its logic into the translated RWX config

For cross-repo references (`uses: org/repo@ref`):
- Add a `# TODO:` comment explaining what the action does and that the user needs to
  translate it manually or find an RWX package equivalent

### Step 4: Use MCP tools if available

MCP tools specific to this migration are not yet available, so for now you can skip this step.

### Step 5: Apply RWX optimization rules

This is the core of the migration. Do NOT produce a 1:1 mapping. Apply the optimization
rules from the RWX Reference and GHA-to-RWX Mapping Reference above — including DAG
decomposition, content-based caching, package substitution, trigger mapping, secret mapping,
and environment variable translation.

### Step 6: Write the output

Write the generated RWX config to `.rwx/<name>.yml`, where `<name>` is derived from the
source workflow filename (e.g., `ci.github.yml` → `.rwx/ci.yml`).

Structure the file in this order:
1. `on:` triggers
2. `base:` image and config
3. `tool-cache:` (if needed)
4. `tasks:` array, ordered by DAG level (independent tasks first, then their dependents)

### Step 7: Validate

After writing the file, review any LSP diagnostics (errors and warnings) that appear. This
plugin bundles an LSP server (`rwx lsp serve`) that automatically validates RWX config files —
diagnostics are surfaced automatically after you write or edit `.rwx/*.yml` files.

If there are diagnostics:

- Read the diagnostic messages
- Fix the issues in the generated config
- Re-check diagnostics after each fix until the file is clean

Common issues the LSP will catch:

- Invalid YAML structure
- Unknown task keys or properties
- Outdated package versions (the LSP will suggest updates)
- Missing required fields

You can also initiate test runs locally without pushing the code. See `rwx run --help` for documentation.

### Step 8: Explain

Provide a summary to the user:
- What the original workflow did
- How the RWX version is structured differently (and why it's better)
- The DAG shape: which tasks run in parallel vs sequentially
- Any `# TODO:` items that need manual attention
- Secrets that need to be configured in RWX Cloud
- Estimated parallelism improvement (e.g., "6 sequential steps → 3-level DAG")

Then suggest the user run `/rwx:review-migration` to get an independent review of the
generated config. This runs a separate reviewer that re-reads both files from scratch and
checks for semantic gaps, missing steps, and optimization opportunities.
