# RWX Skills

Skills for working with [RWX](https://www.rwx.com).

<!-- prettier-ignore -->
> [!IMPORTANT]
> This repository is under active development.

## Installation

### Claude Code

In Claude Code, run:

```
/plugin marketplace add rwx-cloud/skills
```

### Universal (npx)

```
npx skills add rwx-cloud/skills
```

### Codex

Codex automatically reads `AGENTS.md` from the repo root and discovers the MCP
server from `.codex/config.toml`. Clone the repo and ensure `rwx` is on PATH.

### Cursor / GitHub Copilot

The `.vscode/mcp.json` config is picked up automatically when the repo is open.
Copilot also reads `.github/copilot-instructions.md` for skill context.

## Skills

### `/rwx`

Generates or modifies an RWX CI/CD config. Analyzes project structure, selects
appropriate packages, and produces an optimized config with DAG parallelism,
content-based caching, and RWX packages. If an existing `.rwx/*.yml` config is
present, modifies it in place.

```
/rwx CI pipeline with tests and deploy
```

The skill will:

1. Analyze project structure and any existing `.rwx/*.yml` configs
2. Fetch the latest RWX reference documentation
3. Generate (or update) an optimized config at `.rwx/<name>.yml`
4. Validate via `rwx lint` and fix any errors
5. Summarize the DAG shape, packages used, and next steps

### Migration Skills

#### `/migrate-from-gha`

Migrates a GitHub Actions workflow to RWX.

```
/migrate-from-gha .github/workflows/ci.yml
```

#### `/review-gha-migration`

Reviews a generated RWX config against the original GitHub Actions workflow.

```
/review-gha-migration .rwx/ci.yml
```

## Architecture

- **Skills** (`skills/*/SKILL.md`) — Agent-neutral procedural playbooks. Each
  skill includes `curl` commands for fetching the latest reference documentation
  directly.
- **MCP** (`.mcp.json`) — Connects to `rwx mcp serve` for package lookups,
  server-side translation, and on-demand docs. Optional — skills work
  standalone.
- **LSP** (`.lsp.json`) — Connects to the RWX language server for real-time
  validation of `.rwx/*.yml` files (Claude Code).
- **CLI Validation** — `rwx lint <file>` provides validation for all agents via
  the command line.

## Agent Config

| File                              | Agent(s)        | Purpose                             |
| --------------------------------- | --------------- | ----------------------------------- |
| `AGENTS.md`                       | Codex, OpenCode | Vendor-neutral agent instructions   |
| `.claude-plugin/plugin.json`      | Claude Code     | Plugin discovery                    |
| `.claude/settings.local.json`     | Claude Code     | Permissions and MCP config          |
| `.lsp.json`                       | Claude Code     | Real-time LSP validation            |
| `.mcp.json`                       | Claude Code     | MCP server config                   |
| `.codex/config.toml`              | Codex           | MCP server config (TOML)            |
| `.vscode/mcp.json`                | Cursor, Copilot | MCP server config (VS Code)         |
| `.github/copilot-instructions.md` | GitHub Copilot  | Agent instructions                  |
| `marketplace.json`                | Universal       | Skill registry for `npx skills add` |

## Evals

The `evals/` directory contains a Go-based evaluation framework that tests
skills end-to-end by running Claude headlessly against fixtures. See
[`evals/README.md`](evals/README.md) for details.

## Requirements

- [RWX CLI](https://www.rwx.com/docs/rwx/getting-started/installing-the-cli)
  (`rwx` on PATH)

## License

MIT
