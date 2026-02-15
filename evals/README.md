# Skill Evals

End-to-end evaluations that run Claude headlessly against test fixtures, verify
that the correct skills are invoked, and assert that outputs are valid.

## How evals work

Each eval test:

1. Sets up a temporary working directory with fixture files (e.g. a
   `.github/workflows/` YAML)
2. Runs Claude headlessly via `claude --print --output-format json`, piping a
   task prompt via stdin
3. Parses the JSON output to extract tool uses, skill invocations, and token
   usage
4. Asserts correctness — the right skill was used, valid configs were produced,
   semantic invariants hold, etc.
5. Compares performance against a saved baseline to catch regressions in token
   usage or execution time

## Project structure

```
evals/
├── go.mod                  # Go module (one external dep: yaml.v3)
├── claude.go               # Headless Claude runner + JSON output types
├── baseline.go             # Load/save/compare performance snapshots
├── rwx.go                  # RWX config parser for structural assertions
├── assertions.go           # Composable semantic checks
├── assertions_test.go      # Unit tests for parser + assertions
├── integration/            # Eval tests (run Claude end-to-end)
│   ├── helpers_test.go     # Shared test helpers
│   ├── migrate_gha_test.go # GitHub Actions migration evals
│   ├── review_gha_test.go  # GitHub Actions review evals
│   └── testdata/
│       ├── fixtures/
│       │   └── gha/        # GitHub Actions fixtures
│       └── baselines/      # Performance snapshots (auto-generated)
└── README.md
```

Unit tests (`assertions_test.go`) and eval tests (`integration/`) are separated
so CI can run them independently — unit tests are fast and need no external
tools, while eval tests call Claude and run in parallel as dynamic tasks.

## Running tests

### In CI

Note, right now CI runs with `.rwx/evals.yml` require access to a vault in the
RWX organization, so only RWX employees can run evals in CI.

The CI pipeline is helpful for parallelizing evals, but it does require a
short-lived OAuth token. Run `/bin/setup_claude_skill_ci.sh` to sync your Claude
auth settings to the vault, then run `rwx run .rwx/evals.yml` to kick off evals
in CI.

Evals do not currently run automatically on pull request, and are intended to be
one part of a larger manual effort to ensure confidence in changes.

### Locally

Any contributor can run evals locally, regardless of access to the RWX
organization in CI.

```bash
# Unit tests (parser + assertions, no external deps)
go test -v $(go list ./... | grep -v /integration)

# Eval tests in short mode (skips Claude calls, just verifies compilation)
go test -v -short ./integration/

# Run all evals end-to-end
go test -v -timeout 600s ./integration/

# Run a single eval
go test -v -run TestMigrateGHASimpleCI -timeout 600s ./integration/

# Generate/update baselines
go test -v -update -timeout 600s ./integration/
```

## Semantic assertions

Beyond linting and skill detection, evals verify that the agent's output
preserves the _meaning_ of the source workflow. Each fixture defines structural
invariants — things the agent should always get right regardless of how it
phrases the config.

Assertions are composable and run as subtests:

```go
runGHAMigrationEval(t, "matrix-ci.yml", []evals.ConfigAssertion{
    evals.HasPackage("golang/install"),
    evals.HasService("postgres"),
    evals.HasEnvVar("DATABASE_URL"),
    evals.HasRunContaining("go test"),
})
```

Available assertion constructors:

| Assertion                  | Checks                                                                   |
| -------------------------- | ------------------------------------------------------------------------ |
| `HasTask(key)`             | A task with the given key exists                                         |
| `HasPackage(prefix)`       | Some task calls a package matching the prefix                            |
| `HasRunContaining(substr)` | Some task's `run:` field contains the substring                          |
| `TaskDependsOn(task, dep)` | Task lists dep in its `use:` array                                       |
| `HasService(substr)`       | Some task has a background process matching the substring                |
| `HasEnvVar(key)`           | Some task has the env var in `env:` or as an inline assignment in `run:` |
| `HasSecretRef(name)`       | Some task references the secret in `env:`, `with:`, or `run:`            |
| `HasConditional(task)`     | Task has an `if:` field                                                  |
| `MinTaskCount(n)`          | Config has at least n tasks                                              |
| `Either(name, ...)`        | Passes if any of the given assertions passes (logical OR combinator)     |

For review evals, `assertOutputMentions(t, result, substr)` checks that Claude's
text output mentions a specific issue (case-insensitive).

## Adding a new eval

### For an existing provider (e.g. GitHub Actions)

1. Add a fixture YAML to `integration/testdata/fixtures/gha/`
2. Add a test function in the appropriate `*_gha_test.go` file in `integration/`
3. Define semantic invariants — what must the generated config always contain?
4. Run the eval once with `-update` to generate its baseline

### For a new provider

1. Create a new fixture directory: `integration/testdata/fixtures/<provider>/`
2. Add fixture files for that provider's CI config format
3. Create `migrate_<provider>_test.go` and/or `review_<provider>_test.go` in
   `integration/`
4. Follow the same pattern: setup work dir → run Claude → assert skill +
   invariants → check regression

## Baselines

Baselines are JSON snapshots stored in `integration/testdata/baselines/`. They
track:

- `input_tokens` / `output_tokens` — token usage
- `execution_time_ms` — wall clock time
- `tools_used` / `skills_used` — which tools and skills were invoked

On each eval run (without `-update`), current metrics are compared against the
baseline. Allowed regression thresholds:

| Metric              | Max increase |
| ------------------- | ------------ |
| `input_tokens`      | 20%          |
| `output_tokens`     | 30%          |
| `execution_time_ms` | 50%          |

If no baseline exists and `-update` isn't set, the test logs a warning and
passes (first run).

To regenerate baselines after intentional changes:

```bash
go test -v -update -timeout 600s ./integration/
```
