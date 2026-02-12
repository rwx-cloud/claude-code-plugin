# RWX Skills

Skills for working with [RWX](https://www.rwx.com).

## Available Skills

Each skill is defined in `skills/<name>/SKILL.md`. Read the skill's SKILL.md for the full procedure before executing it.

- **migrate-from-gha**: Migrate a GitHub Actions workflow to RWX. See `skills/migrate-from-gha/SKILL.md`.
- **review-gha-migration**: Review a generated RWX config against the original workflow. See `skills/review-gha-migration/SKILL.md`.

## Requirements

- The `rwx` CLI must be installed and on PATH.
- After writing RWX configs, validate with `rwx lint .rwx/<name>.yml`.
