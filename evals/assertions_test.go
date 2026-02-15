package evals

import (
	"strings"
	"testing"
)

// testConfig is a realistic RWX config used across all assertion tests.
const testConfigYAML = `
tasks:
  - key: code
    call: git/clone 2.0.2
    with:
      repository: https://github.com/example/repo.git
      ref: ${{ init.commit-sha }}
      github-token: ${{ github.token }}

  - key: go
    call: golang/install 1.2.0
    with:
      go-version: "1.26"

  - key: mod-download
    call: golang/mod-download 1.0.0
    use: [code, go]

  - key: test
    use: [code, go, mod-download]
    env:
      DATABASE_URL: postgres://localhost:5432/testdb
      API_KEY: ${{ secrets.API_KEY }}
    background-processes:
      - key: postgres
        run: pg_ctl start
        ready-check: pg_isready
    run: |
      go test -race ./...
      go vet ./...

  - key: deploy
    use: [test]
    if: github.ref == 'refs/heads/main'
    env:
      DEPLOY_TOKEN: ${{ secrets.DEPLOY_TOKEN }}
    run: ./deploy.sh
`

func mustParseTestConfig(t *testing.T) *RWXConfig {
	t.Helper()
	cfg, err := ParseRWXConfig([]byte(testConfigYAML))
	if err != nil {
		t.Fatalf("parsing test config: %v", err)
	}
	return cfg
}

// shouldPass runs an assertion and fails if it doesn't pass.
func shouldPass(t *testing.T, cfg *RWXConfig, a ConfigAssertion) {
	t.Helper()
	a.Check(t, cfg)
}

// shouldFail runs an assertion and fails the test if it unexpectedly passes.
// It uses a probeTB to capture the expected failure without propagating it.
func shouldFail(t *testing.T, cfg *RWXConfig, a ConfigAssertion) {
	t.Helper()
	probe := &probeTB{}
	a.Check(probe, cfg)
	if !probe.failed {
		t.Errorf("expected assertion %q to fail, but it passed", a.Name)
	}
}

func TestParseRWXConfig(t *testing.T) {
	cfg := mustParseTestConfig(t)

	if len(cfg.Tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(cfg.Tasks))
	}

	keys := cfg.TaskKeys()
	expected := []string{"code", "go", "mod-download", "test", "deploy"}
	for i, k := range expected {
		if keys[i] != k {
			t.Errorf("task %d: expected key %q, got %q", i, k, keys[i])
		}
	}

	// Verify call parsing.
	if cfg.Task("code").Call != "git/clone 2.0.2" {
		t.Errorf("unexpected call: %s", cfg.Task("code").Call)
	}

	// Verify use parsing.
	test := cfg.Task("test")
	if len(test.Use) != 3 {
		t.Errorf("expected 3 deps for test, got %d", len(test.Use))
	}

	// Verify env parsing.
	if dbURL, ok := test.EnvString("DATABASE_URL"); !ok || dbURL != "postgres://localhost:5432/testdb" {
		t.Errorf("unexpected DATABASE_URL: %s", dbURL)
	}

	// Verify background-processes parsing.
	if len(test.BackgroundProcesses) != 1 || test.BackgroundProcesses[0].Key != "postgres" {
		t.Errorf("expected postgres background process, got: %+v", test.BackgroundProcesses)
	}

	// Verify if parsing.
	deploy := cfg.Task("deploy")
	if deploy.If == "" {
		t.Error("expected deploy to have a conditional")
	}

	// Verify multiline run parsing.
	run := test.RunString()
	if !strings.Contains(run, "go test") || !strings.Contains(run, "go vet") {
		t.Errorf("expected test run to contain 'go test' and 'go vet', got: %s", run)
	}
}

func TestHasTask_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, HasTask("test"))
	shouldPass(t, cfg, HasTask("deploy"))
}

func TestHasTask_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, HasTask("nonexistent"))
}

func TestHasPackage_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, HasPackage("git/clone"))
	shouldPass(t, cfg, HasPackage("golang/install"))
	shouldPass(t, cfg, HasPackage("golang/mod-download"))
}

func TestHasPackage_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, HasPackage("nodejs/install"))
}

func TestHasRunContaining_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, HasRunContaining("go test"))
	shouldPass(t, cfg, HasRunContaining("go vet"))
	shouldPass(t, cfg, HasRunContaining("deploy.sh"))
}

func TestHasRunContaining_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, HasRunContaining("npm install"))
}

func TestTaskDependsOn_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, TaskDependsOn("test", "code"))
	shouldPass(t, cfg, TaskDependsOn("test", "go"))
	shouldPass(t, cfg, TaskDependsOn("test", "mod-download"))
	shouldPass(t, cfg, TaskDependsOn("deploy", "test"))
}

func TestTaskDependsOn_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, TaskDependsOn("test", "deploy"))
	shouldFail(t, cfg, TaskDependsOn("nonexistent", "code"))
}

func TestHasService_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, HasService("postgres"))
	shouldPass(t, cfg, HasService("pg_ctl"))
}

func TestHasService_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, HasService("redis"))
}

func TestHasEnvVar_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, HasEnvVar("DATABASE_URL"))
	shouldPass(t, cfg, HasEnvVar("API_KEY"))
	shouldPass(t, cfg, HasEnvVar("DEPLOY_TOKEN"))
}

func TestHasEnvVar_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, HasEnvVar("MISSING_VAR"))
}

func TestHasSecretRef_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, HasSecretRef("API_KEY"))
	shouldPass(t, cfg, HasSecretRef("DEPLOY_TOKEN"))
}

func TestHasSecretRef_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, HasSecretRef("NONEXISTENT"))
}

func TestHasConditional_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, HasConditional("deploy"))
}

func TestHasConditional_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, HasConditional("test"))
	shouldFail(t, cfg, HasConditional("nonexistent"))
}

func TestMinTaskCount_Pass(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldPass(t, cfg, MinTaskCount(1))
	shouldPass(t, cfg, MinTaskCount(5))
}

func TestMinTaskCount_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, MinTaskCount(10))
}

func TestSecretRefInWith(t *testing.T) {
	// Verify HasSecretRef finds secrets in the with field too.
	yaml := `
tasks:
  - key: clone
    call: git/clone 2.0.2
    with:
      github-token: ${{ secrets.GH_TOKEN }}
`
	cfg, err := ParseRWXConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	shouldPass(t, cfg, HasSecretRef("GH_TOKEN"))
}

func TestEither_FirstPasses(t *testing.T) {
	cfg := mustParseTestConfig(t)
	// First option matches — git/clone exists as a package.
	shouldPass(t, cfg, Either("clones_repo",
		HasPackage("git/clone"),
		HasRunContaining("git clone"),
	))
}

func TestEither_SecondPasses(t *testing.T) {
	cfg := mustParseTestConfig(t)
	// First option fails (no nodejs/install), second matches (deploy.sh run).
	shouldPass(t, cfg, Either("has_deploy",
		HasPackage("nodejs/install"),
		HasRunContaining("deploy.sh"),
	))
}

func TestEither_Fail(t *testing.T) {
	cfg := mustParseTestConfig(t)
	shouldFail(t, cfg, Either("neither_matches",
		HasPackage("nodejs/install"),
		HasRunContaining("npm install"),
	))
}

func TestFlexStrings_SingleString(t *testing.T) {
	yaml := `
tasks:
  - key: deploy
    use: test
`
	cfg, err := ParseRWXConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	task := cfg.Task("deploy")
	if len(task.Use) != 1 || task.Use[0] != "test" {
		t.Errorf("expected Use=[test], got %v", task.Use)
	}
}

func TestHasTaskWithCall_Boundary(t *testing.T) {
	cfg, err := ParseRWXConfig([]byte(`
tasks:
  - key: exact
    call: git/clone
  - key: versioned
    call: git/clone 2.0.2
  - key: similar
    call: git/clone-extra 1.0.0
`))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if !cfg.HasTaskWithCall("git/clone") {
		t.Fatal("expected git/clone to match exact and versioned calls")
	}
	if cfg.HasTaskWithCall("git/clon") {
		t.Fatal("expected partial prefix git/clon to not match")
	}
	if cfg.HasTaskWithCall("git/clone-ex") {
		t.Fatal("expected similar prefix git/clone-ex to not match")
	}
}

func TestSecretRefInRun(t *testing.T) {
	yaml := `
tasks:
  - key: deploy
    run: |
      curl -H "Authorization: ${{ secrets.DEPLOY_TOKEN }}" https://example.com
`
	cfg, err := ParseRWXConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	shouldPass(t, cfg, HasSecretRef("DEPLOY_TOKEN"))
	shouldFail(t, cfg, HasSecretRef("OTHER_TOKEN"))
}

func TestHasEnvVarInRun(t *testing.T) {
	yaml := `
tasks:
  - key: test
    run: |
      export DATABASE_URL=postgres://localhost/test
      go test ./...
`
	cfg, err := ParseRWXConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	shouldPass(t, cfg, HasEnvVar("DATABASE_URL"))
	shouldFail(t, cfg, HasEnvVar("MISSING_VAR"))
}

func TestSecretRefInVaultSyntax(t *testing.T) {
	yaml := `
tasks:
  - key: eval
    env:
      API_KEY: ${{ vaults.team.secrets.API_KEY }}
`
	cfg, err := ParseRWXConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	shouldPass(t, cfg, HasSecretRef("API_KEY"))
}

func TestEmptyConfig(t *testing.T) {
	cfg := &RWXConfig{}

	if cfg.Task("anything") != nil {
		t.Error("expected nil task from empty config")
	}
	if len(cfg.TaskKeys()) != 0 {
		t.Error("expected no keys from empty config")
	}
	if cfg.HasTaskWithCall("any") {
		t.Error("expected no calls in empty config")
	}
	if cfg.HasBackgroundProcess("any") {
		t.Error("expected no background processes in empty config")
	}
	if cfg.DependsOn("a", "b") {
		t.Error("expected no dependencies in empty config")
	}
}
