package integration

import (
	"fmt"
	"testing"

	"github.com/rwx-cloud/skills/evals"
)

func runGHAMigrationEval(t *testing.T, fixtureName string, invariants []evals.ConfigAssertion) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping eval test in short mode")
	}

	fixturePath := "gha/" + fixtureName
	workDir := setupWorkDir(t, fixturePath)
	ctx := evalContext(t)

	prompt := fmt.Sprintf(
		"Migrate the GitHub Actions workflow at .github/workflows/%s to RWX",
		fixtureName,
	)

	result, err := evals.RunClaude(ctx, prompt, workDir)
	if err != nil {
		t.Fatalf("RunClaude failed: %v", err)
	}

	assertSkillUsed(t, result, "rwx:migrate-from-gha")
	assertRWXConfigExists(t, workDir)
	assertRWXConfigValid(t, ctx, workDir)
	evals.AssertConfig(t, workDir, invariants)
	evals.AssertNoRegression(t, result)
}

// installsGo matches either package name the agent might use for Go installation.
func installsGo() evals.ConfigAssertion {
	return evals.Either("installs_go",
		evals.HasPackage("golang/install"),
		evals.HasPackage("go/install"),
	)
}

// clonesRepo matches either a git/clone package or a git clone run command.
func clonesRepo() evals.ConfigAssertion {
	return evals.Either("clones_repo",
		evals.HasPackage("git/clone"),
		evals.HasRunContaining("git clone"),
	)
}

// simple-ci.yml: checkout → setup-go 1.23 → go mod download → go test → go vet
func TestMigrateGHASimpleCI(t *testing.T) {
	runGHAMigrationEval(t, "simple-ci.yml", []evals.ConfigAssertion{
		clonesRepo(),
		installsGo(),
		evals.HasRunContaining("go test"),
		evals.HasRunContaining("go vet"),
	})
}

// matrix-ci.yml: matrix (go 1.22, 1.23) + postgres service + cache + env vars + race tests
func TestMigrateGHAMatrixCI(t *testing.T) {
	runGHAMigrationEval(t, "matrix-ci.yml", []evals.ConfigAssertion{
		clonesRepo(),
		installsGo(),
		evals.Either("runs_postgres",
			evals.HasService("postgres"),
			evals.HasRunContaining("postgres"),
		),
		evals.HasEnvVar("DATABASE_URL"),
		evals.HasRunContaining("go test"),
		evals.HasRunContaining("go vet"),
	})
}

// multi-job-ci.yml: lint/test → build (needs both) → deploy (needs build, conditional, secrets)
func TestMigrateGHAMultiJobCI(t *testing.T) {
	runGHAMigrationEval(t, "multi-job-ci.yml", []evals.ConfigAssertion{
		clonesRepo(),
		installsGo(),
		evals.HasRunContaining("golangci-lint"),
		evals.HasRunContaining("go test"),
		evals.HasRunContaining("go build"),
		evals.HasSecretRef("DEPLOY_TOKEN"),
		// At least: clone, go-install, lint, test, build, deploy
		evals.MinTaskCount(6),
	})
}
