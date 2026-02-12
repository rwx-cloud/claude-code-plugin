---
name: migrate-from-gha
description: Migrate a GitHub Actions workflow to RWX. Translates triggers, jobs, steps into an optimized RWX config with DAG parallelism, content-based caching, and RWX packages.
argument-hint: [path/to/.github/workflows/ci.yml]
---

## Quick Reference

Read the cheat sheet before starting: [GHA Cheat Sheet](references/gha-cheat-sheet.md)

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

Fetch the full reference documentation now. Read these reference files and then fetch their
contents:
- [RWX Reference](references/rwx-reference.md)
- [GHA-to-RWX Mapping](references/gha-reference.md)

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

After writing the file, validate the generated config:

    rwx lint .rwx/<name>.yml

If there are diagnostics:

- Read the diagnostic messages
- Fix the issues in the generated config
- Re-check diagnostics after each fix until the file is clean

Common issues the validator will catch:

- Invalid YAML structure
- Unknown task keys or properties
- Outdated package versions (the validator will suggest updates)
- Missing required fields

You can also initiate test runs locally without pushing the code. See `rwx run --help` for documentation.

### Step 8: Automated review

Tell the user: "Launching a review of the migration. This reviewer has no knowledge of the
decisions made during migration — it will read both files from scratch and check for gaps."

First, read the reference docs and the review procedure so you can include them:
- Fetch the contents from the URLs in [RWX Reference](references/rwx-reference.md) and [GHA-to-RWX Mapping](references/gha-reference.md)
- Read the review procedure at [review-gha-migration/SKILL.md](../review-gha-migration/SKILL.md)

**If you have the ability to spawn a subagent** (e.g., Claude Code's Task tool), do so for an
independent review with fresh context. Spawn the reviewer using a general-purpose subagent with
a prompt that includes:
1. The full contents of the review procedure (from the SKILL.md you just read)
2. The full contents of both reference docs (from the fetches you just ran)
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

**Otherwise**, perform the review inline by reading and following the review procedure from
[review-gha-migration/SKILL.md](../review-gha-migration/SKILL.md).

Wait for the review to complete. If the review found blocking issues, fix them before
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
