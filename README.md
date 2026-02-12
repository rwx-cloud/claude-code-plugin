# RWX Skills

Skills for working with [RWX](https://www.rwx.com).

> [!IMPORTANT]
> This repository is under active development and is not yet fully supported.

## Installation

### Claude Code

```
claude plugin install --from-repo https://github.com/rwx-cloud/skills
```

### Universal (npx)

```
npx skills add rwx-cloud/skills
```

### Codex

Codex automatically reads `AGENTS.md` from the repo root and discovers the MCP server from `.codex/config.toml`. Clone the repo and ensure `rwx` is on PATH.

### Cursor / GitHub Copilot

The `.vscode/mcp.json` config is picked up automatically when the repo is open. Copilot also reads `.github/copilot-instructions.md` for skill context.

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
5. Validate via `rwx lint` and fix any errors
6. Run an automated review to catch gaps
7. Summarize the migration and next steps

### `/rwx:review-gha-migration`

Reviews a generated RWX config against the original GitHub Actions workflow.

```
/rwx:review-gha-migration .rwx/ci.yml
```

## Architecture

- **Skills** (`skills/*/SKILL.md`) — Agent-neutral procedural playbooks. Each skill references documentation via `references/` files that point to canonical URLs.
- **MCP** (`.mcp.json`) — Connects to `rwx mcp serve` for package lookups, server-side translation, and on-demand docs. Optional — skills work standalone.
- **LSP** (`.lsp.json`) — Connects to the RWX language server for real-time validation of `.rwx/*.yml` files (Claude Code).
- **CLI Validation** — `rwx lint <file>` provide validation for all agents via the command line.

## Agent Config

| File | Agent(s) | Purpose |
|------|----------|---------|
| `AGENTS.md` | Codex, OpenCode | Vendor-neutral agent instructions |
| `.claude-plugin/plugin.json` | Claude Code | Plugin discovery |
| `.claude/settings.local.json` | Claude Code | Permissions and MCP config |
| `.lsp.json` | Claude Code | Real-time LSP validation |
| `.mcp.json` | Claude Code | MCP server config |
| `.codex/config.toml` | Codex | MCP server config (TOML) |
| `.vscode/mcp.json` | Cursor, Copilot | MCP server config (VS Code) |
| `.github/copilot-instructions.md` | GitHub Copilot | Agent instructions |
| `marketplace.json` | Universal | Skill registry for `npx skills add` |

## Requirements

- [RWX CLI](https://www.rwx.com/docs/rwx/getting-started/installing-the-cli) (`rwx` on PATH)

## License

MIT
