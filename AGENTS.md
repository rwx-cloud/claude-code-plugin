# RWX Skills

Skills for working with [RWX](https://www.rwx.com).

## Requirements

The `rwx` CLI must be on your PATH. Install it from
[RWX docs](https://www.rwx.com/docs/rwx/getting-started/installing-the-cli).

## Available Skills

### migrate-from-gha

Migrates a GitHub Actions workflow to an optimized RWX config with DAG
parallelism, content-based caching, and RWX packages.

Usage: invoke with a path to a GitHub Actions workflow file, e.g.
`.github/workflows/ci.yml`

### review-gha-migration

Reviews a generated RWX config against the original GitHub Actions workflow to
catch semantic gaps, missing steps, and optimization opportunities.

Usage: invoke with a path to an RWX config file, e.g. `.rwx/ci.yml`

## Validation

After writing or modifying RWX config files (`.rwx/*.yml`), validate them by
running:

    rwx lint .rwx/<name>.yml

## Skill Procedures

Full step-by-step procedures are in the `skills/` directory:

- `skills/migrate-from-gha/SKILL.md`
- `skills/review-gha-migration/SKILL.md`

Each skill directory also contains a `references/` folder with pointers to the
latest RWX documentation.
