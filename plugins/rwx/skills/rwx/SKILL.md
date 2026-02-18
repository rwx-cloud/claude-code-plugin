---
name: rwx
description:
  Can be used when understanding, creating or modifying an RWX CI/CD config.
argument-hint: [optional description, e.g. "CI pipeline with tests and deploy"]
allowed-tools:
  Bash(rwx lint *), Bash(rwx docs *), Bash(rwx logs *), Bash(rwx artifacts *),
  Bash(rwx results *), Bash(rwx * --help)
---

## Generate or Modify RWX Config

You have been tasked with creating, modifying, or understanding/explaining an
RWX CI/CD config for this project.

Fetch the reference docs index with:

    rwx docs pull /docs/rwx/migrating/rwx-reference

If you encounter a question not covered by these references, use
`rwx docs search "<query>"` to find the relevant documentation page, then
`rwx docs pull` the result.

When making changes, you can run validation on the config:

    rwx lint .rwx/<name>.yml

If the user chooses, you can kick off an actual run on RWX:

    rwx run .rwx/<name>.yml --wait

When the run finishes, results will be shown, and you can iterate in that
fashion until the run passes.

No git push is required to invoke a run from the RWX CLI.
