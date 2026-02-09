# RWX Claude Code Plugin

Claude Code plugin for interacting with [RWX](https://www.rwx.com).

> [!IMPORTANT]
> This plugin is under active development and is not yet fully supported.

## Installation

This repository is not yet available via a Claude marketplace, but you can install directly from this repo:

```
claude plugin install --from-repo https://github.com/rwx-cloud/claude-code-plugin
```

## Skills

### `/rwx:migrate-from-gha`

Migrates a GitHub Actions workflow to RWX.

```
/rwx:migrate-from-gha .github/workflows/ci.yml
```

The skill will:

1. Read and analyze the source workflow
2. Translate triggers, jobs, and steps into RWX config
3. Optimize for RWX strengths — parallel DAG, content-based caching, package substitution
4. Write the output to `.rwx/<name>.yml`
5. Validate via the RWX language server and fix any errors
6. Explain what changed and why

## Architecture

The plugin has three layers:

- **Skill** (`skills/migrate-from-gha/SKILL.md`) — Procedural playbook that drives the migration. Fetches the latest GHA-to-RWX reference content at invocation time via `!curl`.
- **MCP** (`.mcp.json`) — Connects to `rwx mcp serve` for package lookups, server-side translation, and on-demand docs. Optional — the skill works standalone without MCP.
- **LSP** (`.lsp.json`) — Connects to the RWX language server for real-time validation of generated `.rwx/*.yml` files.

## Requirements

- [Claude Code](https://claude.ai/code)
- [RWX CLI](https://www.rwx.com/docs/rwx/getting-started/installing-the-cli) (`rwx` on PATH)

## License

MIT
