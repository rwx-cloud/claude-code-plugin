package integration

import (
	"testing"

	"github.com/rwx-cloud/skills/evals"
)

func runCreateRWXEval(t *testing.T, fixtureName string, prompt string, invariants []evals.ConfigAssertion) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping eval test in short mode")
	}

	workDir := setupProjectDir(t, fixtureName)
	ctx := evalContext(t)

	result, err := evals.RunClaude(ctx, prompt, workDir)
	if err != nil {
		t.Fatalf("RunClaude failed: %v", err)
	}
	saveClaudeOutput(t, result)

	assertSkillUsed(t, result, "rwx:rwx")
	assertRWXConfigExists(t, workDir)
	assertRWXConfigValid(t, ctx, workDir)
	evals.AssertConfig(t, workDir, invariants)
	evals.AssertNoRegression(t, result)
}

func TestCreateRWXGoBasic(t *testing.T) {
	runCreateRWXEval(t, "go-basic", "/rwx:rwx", []evals.ConfigAssertion{
		clonesRepo(),
		installsGo(),
		evals.HasRunContaining("go test"),
	})
}

func TestCreateRWXNodeBasic(t *testing.T) {
	runCreateRWXEval(t, "node-basic", "/rwx:rwx", []evals.ConfigAssertion{
		clonesRepo(),
		installsNode(),
		evals.Either("runs_tests",
			evals.HasRunContaining("npm test"),
			evals.HasRunContaining("npm run test"),
		),
		evals.MinTaskCount(3),
	})
}

func TestCreateRWXPythonBasic(t *testing.T) {
	runCreateRWXEval(t, "python-basic", "/rwx:rwx", []evals.ConfigAssertion{
		clonesRepo(),
		evals.Either("runs_pytest",
			evals.HasRunContaining("pytest"),
			evals.HasRunContaining("python -m pytest"),
		),
		evals.MinTaskCount(3),
	})
}

func TestCreateRWXRustBasic(t *testing.T) {
	runCreateRWXEval(t, "rust-basic", "/rwx:rwx", []evals.ConfigAssertion{
		clonesRepo(),
		evals.HasRunContaining("cargo test"),
		evals.MinTaskCount(3),
	})
}

func TestCreateRWXGoPostgres(t *testing.T) {
	runCreateRWXEval(t, "go-postgres", "/rwx:rwx", []evals.ConfigAssertion{
		clonesRepo(),
		installsGo(),
		evals.Either("runs_postgres",
			evals.HasService("postgres"),
			evals.HasRunContaining("postgres"),
		),
		evals.HasEnvVar("DATABASE_URL"),
		evals.HasRunContaining("go test"),
	})
}

func TestCreateRWXGoRedis(t *testing.T) {
	runCreateRWXEval(t, "go-redis", "/rwx:rwx", []evals.ConfigAssertion{
		clonesRepo(),
		installsGo(),
		evals.Either("runs_redis",
			evals.HasService("redis"),
			evals.HasRunContaining("redis"),
		),
		evals.HasRunContaining("go test"),
	})
}

func TestCreateRWXGoMultiStage(t *testing.T) {
	runCreateRWXEval(t, "go-multi-stage",
		"/rwx:rwx CI pipeline with linting, testing, and building",
		[]evals.ConfigAssertion{
			clonesRepo(),
			installsGo(),
			evals.HasRunContaining("go test"),
			evals.HasRunContaining("go build"),
			evals.Either("runs_lint",
				evals.HasRunContaining("golangci-lint"),
				evals.HasRunContaining("go vet"),
			),
			evals.MinTaskCount(5),
		})
}

func TestCreateRWXGoDeploy(t *testing.T) {
	runCreateRWXEval(t, "go-deploy",
		"/rwx:rwx CI/CD pipeline with tests, build, and conditional deploy to production using DEPLOY_TOKEN secret",
		[]evals.ConfigAssertion{
			clonesRepo(),
			installsGo(),
			evals.HasRunContaining("go test"),
			evals.HasSecretRef("DEPLOY_TOKEN"),
			evals.MinTaskCount(5),
		})
}

func TestCreateRWXGoMatrix(t *testing.T) {
	runCreateRWXEval(t, "go-matrix",
		"/rwx:rwx CI pipeline that tests against Go versions 1.22 and 1.26",
		[]evals.ConfigAssertion{
			clonesRepo(),
			installsGo(),
			evals.HasRunContaining("go test"),
			evals.MinTaskCount(3),
		})
}

func TestCreateRWXDockerBuild(t *testing.T) {
	runCreateRWXEval(t, "docker-build",
		"/rwx:rwx CI pipeline that builds a Docker image",
		[]evals.ConfigAssertion{
			clonesRepo(),
			evals.Either("builds_docker",
				evals.HasRunContaining("docker build"),
				evals.HasRunContaining("docker"),
			),
			evals.MinTaskCount(3),
		})
}

func TestCreateRWXToolVersions(t *testing.T) {
	runCreateRWXEval(t, "tool-versions", "/rwx:rwx", []evals.ConfigAssertion{
		clonesRepo(),
		evals.HasPackage("rwx/tool-versions"),
		evals.HasRunContaining("go test"),
	})
}

func TestCreateRWXMonorepo(t *testing.T) {
	runCreateRWXEval(t, "monorepo",
		"/rwx:rwx CI pipeline for this monorepo",
		[]evals.ConfigAssertion{
			clonesRepo(),
			evals.MinTaskCount(4),
			evals.HasRunContaining("go test"),
		})
}

func TestCreateRWXNodeFullstack(t *testing.T) {
	runCreateRWXEval(t, "node-fullstack",
		"Create an RWX CI/CD config for this project",
		[]evals.ConfigAssertion{
			clonesRepo(),
			installsNode(),
			evals.HasRunContaining("test"),
			evals.MinTaskCount(4),
		})
}
