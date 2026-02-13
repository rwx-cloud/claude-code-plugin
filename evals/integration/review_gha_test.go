package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rwx-cloud/skills/evals"
)

// setupGHAReviewWorkDir creates a temp dir with both a GHA workflow and a pre-baked RWX config.
func setupGHAReviewWorkDir(t *testing.T, ghaFixture, rwxFixture string) string {
	t.Helper()

	workDir := setupWorkDir(t, "gha/"+ghaFixture)

	rwxDir := filepath.Join(workDir, ".rwx")
	if err := os.MkdirAll(rwxDir, 0o755); err != nil {
		t.Fatalf("creating .rwx dir: %v", err)
	}

	src := filepath.Join("testdata", "fixtures", "gha", rwxFixture)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading RWX fixture %s: %v", rwxFixture, err)
	}

	dst := filepath.Join(rwxDir, rwxFixture)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("writing RWX fixture to work dir: %v", err)
	}

	return workDir
}

func TestReviewGHASimpleMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping eval test in short mode")
	}

	workDir := setupGHAReviewWorkDir(t, "simple-ci.yml", "simple-ci-rwx.yml")
	ctx := evalContext(t)

	result, err := evals.RunClaude(ctx, "/rwx:review-gha-migration .rwx/simple-ci-rwx.yml", workDir)
	if err != nil {
		t.Fatalf("RunClaude failed: %v", err)
	}
	saveClaudeOutput(t, result)

	assertSkillUsed(t, result, "rwx:review-gha-migration")
	assertToolUsed(t, result, "Bash")

	// The pre-baked RWX config is intentionally missing "go vet" —
	// verify the review catches it.
	assertOutputMentions(t, result, "go vet")

	evals.AssertNoRegression(t, result)
}
