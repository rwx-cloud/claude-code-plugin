package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rwx-cloud/skills/evals"
)

// setupWorkDir creates a temporary directory with the given fixture copied
// into .github/workflows/. Returns the path to the temp directory.
func setupWorkDir(t *testing.T, fixtureName string) string {
	t.Helper()

	tmpDir := t.TempDir()

	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("creating workflow dir: %v", err)
	}

	src := filepath.Join("testdata", "fixtures", fixtureName)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixtureName, err)
	}

	dst := filepath.Join(workflowDir, filepath.Base(fixtureName))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("writing fixture to work dir: %v", err)
	}

	return tmpDir
}

// assertSkillUsed checks that the given skill name appears in the result's SkillUses.
func assertSkillUsed(t *testing.T, result *evals.ExecutionResult, skillName string) {
	t.Helper()

	skills := result.SkillUses()
	for _, s := range skills {
		if s == skillName {
			return
		}
	}
	t.Errorf("expected skill %q to be used, got skills: %v", skillName, skills)
}

// assertToolUsed checks that the given tool name appears in the result's ToolNames.
func assertToolUsed(t *testing.T, result *evals.ExecutionResult, toolName string) {
	t.Helper()

	tools := result.ToolNames()
	for _, tool := range tools {
		if tool == toolName {
			return
		}
	}
	t.Errorf("expected tool %q to be used, got tools: %v", toolName, tools)
}

// assertRWXConfigExists verifies that at least one .rwx/*.yml file was created.
func assertRWXConfigExists(t *testing.T, workDir string) {
	t.Helper()

	pattern := filepath.Join(workDir, ".rwx", "*.yml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("globbing for RWX configs: %v", err)
	}
	if len(matches) == 0 {
		t.Error("expected .rwx/*.yml to exist, but no files found")
	}
}

// assertRWXConfigValid runs rwx lint on all .rwx/*.yml files.
func assertRWXConfigValid(t *testing.T, ctx context.Context, workDir string) {
	t.Helper()

	pattern := filepath.Join(workDir, ".rwx", "*.yml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("globbing for RWX configs: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no .rwx/*.yml files found to validate")
	}

	for _, f := range matches {
		runValidation(t, ctx, "rwx", "lint", f)
	}
}

// runValidation runs a command and fails the test if it exits non-zero.
func runValidation(t *testing.T, ctx context.Context, name string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("%s %v failed: %v\noutput: %s", name, args, err, output)
	}
}

// assertOutputMentions checks that Claude's text output contains the given
// substring (case-insensitive). Useful for verifying that reviews identify
// specific issues.
func assertOutputMentions(t *testing.T, result *evals.ExecutionResult, substr string) {
	t.Helper()

	text := strings.ToLower(result.TextOutput())
	if !strings.Contains(text, strings.ToLower(substr)) {
		t.Errorf("expected Claude output to mention %q, but it did not", substr)
	}
}

// evalContext returns a context with a 5-minute timeout.
func evalContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)
	return ctx
}
