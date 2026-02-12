---
name: migrate-from-gha
description: Migrate a GitHub Actions workflow to RWX. Translates triggers, jobs, steps into an optimized RWX config with DAG parallelism, content-based caching, and RWX packages.
argument-hint: [path/to/.github/workflows/ci.yml]
allowed-tools: Bash(curl *)
---

## Quick Reference

!`curl -sL https://www.rwx.com/docs/rwx/migrating/cheat-sheet.md`

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

Before continuing, tell the user what you found: how many jobs, the dependency graph between
them, which triggers are configured, and anything notable (matrix strategies, services,
composite actions, reusable workflows). Keep it brief — a few sentences, not a full listing.

### Step 3: Follow local composite action references

For steps using `uses: ./.github/actions/foo`:
- Read that action's `action.yml`
- Inline its logic into the translated RWX config

For cross-repo references (`uses: org/repo@ref`):
- Add a `# TODO:` comment explaining what the action does and that the user needs to
  translate it manually or find an RWX package equivalent

Tell the user which composite actions you inlined and which cross-repo references will need
TODO comments.

### Step 4: Use MCP tools if available

MCP tools specific to this migration are not yet available, so for now you can skip this step.

### Step 5: Apply RWX optimization rules

Fetch the full reference documentation now:
- RWX reference: `curl -sL https://www.rwx.com/docs/rwx/migrating/rwx-reference.md`
- GHA-to-RWX mapping: `curl -sL https://www.rwx.com/docs/rwx/migrating/gha-reference.md`

This is the core of the migration. Do NOT produce a 1:1 mapping. Apply the optimization
rules from the reference documentation — including DAG decomposition, content-based caching,
package substitution, trigger mapping, secret mapping, and environment variable translation.

Before writing the file, tell the user your planned DAG structure: which tasks you'll create,
what runs in parallel vs sequentially, and any notable optimization decisions (packages
substituted, caches removed, jobs decomposed). This lets the user course-correct before you
write the config.

### Step 6: Write the output

Write the generated RWX config to `.rwx/<name>.yml`, where `<name>` is derived from the
source workflow filename (e.g., `ci.github.yml` → `.rwx/ci.yml`).

Structure the file in this order:
1. `on:` triggers
2. `base:` image and config
3. `tool-cache:` (if needed)
4. `tasks:` array, ordered by DAG level (independent tasks first, then their dependents)

### Step 7: Validate

After writing the file, check the tool result for LSP diagnostics (errors and warnings). This
plugin bundles an LSP server (`rwx lsp serve`) that automatically validates RWX config files —
diagnostics appear inline in the Write/Edit tool result. Do not use IDE or MCP tools to fetch
diagnostics; they are already included in the tool output.

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

### Step 8: Automated review

Tell the user: "Launching a sub-agent with fresh context to review the migration. This
reviewer has no knowledge of the decisions made during migration — it will read both files
from scratch and check for gaps."

Before spawning the subagent, fetch the reference docs and read the review procedure so you
can include them in the prompt:
- Curl the RWX reference and GHA-to-RWX mapping (same URLs from step 5)
- Read `skills/review-gha-migration/SKILL.md`

Then spawn the reviewer using the Task tool with `subagent_type: "general-purpose"` and a
prompt that includes:
1. The full contents of the review procedure (from the SKILL.md you just read)
2. The full contents of both reference docs (from the curls you just ran)
3. The file paths to review

Structure the prompt like this:

```
You are reviewing an RWX config that was migrated from a GitHub Actions workflow.
Your job is to catch problems the implementer missed. Approach this as a skeptical
reviewer, not as someone defending prior work.

## Review Procedure
<paste the review procedure from SKILL.md here>

## RWX Reference
<paste the RWX reference doc here>

## GHA-to-RWX Mapping Reference
<paste the GHA mapping doc here>

## Files to Review
- Source GHA workflow: <path from step 1>
- Generated RWX config: <path from step 6>
```

Replace the placeholders with the actual content and paths.

Wait for the subagent to complete. If the review found blocking issues, fix them before
continuing.

### Step 9: Summarize

Provide a final summary to the user that covers both the migration and the review:
- What the original workflow did
- How the RWX version is structured differently (and why it's better)
- The DAG shape: which tasks run in parallel vs sequentially
- The review verdict and any issues found (or confirmation that it passed)
- Any `# TODO:` items that need manual attention
- Secrets that need to be configured in RWX Cloud
- Estimated parallelism improvement (e.g., "6 sequential steps → 3-level DAG")

Let the user know they can re-run the review independently at any time with
`/rwx:review-gha-migration`.
