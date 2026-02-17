package integration

import (
	"context"
	"fmt"
	"io/fs"
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
		runValidation(t, ctx, workDir, "rwx", "lint", f)
	}
}

// runValidation runs a command in the given directory and fails the test if it exits non-zero.
func runValidation(t *testing.T, ctx context.Context, dir string, name string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
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

// saveClaudeOutput writes the raw Claude JSON output to tmp/ for CI artifact collection.
func saveClaudeOutput(t *testing.T, result *evals.ExecutionResult) {
	t.Helper()

	dir := filepath.Join("..", "tmp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Logf("WARNING: could not create output dir %s: %v", dir, err)
		return
	}
	path := filepath.Join(dir, "claude-output-"+t.Name()+".json")
	if err := os.WriteFile(path, result.RawOutput, 0o644); err != nil {
		t.Logf("WARNING: could not save Claude output to %s: %v", path, err)
	}

	writeRWXInfo(t, result)
}

// writeRWXInfo writes Claude usage stats to $RWX_INFO so they appear in the RWX UI.
// Each metric is written to a separate file so newlines don't interfere with rendering.
func writeRWXInfo(t *testing.T, result *evals.ExecutionResult) {
	t.Helper()

	infoDir := os.Getenv("RWX_INFO")
	if infoDir == "" {
		return
	}

	evt := result.ResultEvent()
	if evt == nil {
		t.Log("WARNING: no result event found, skipping RWX_INFO")
		return
	}

	var usage evals.TokenUsage
	if evt.Usage != nil {
		usage = *evt.Usage
	}

	prefix := t.Name() + "-"
	entries := []struct {
		key   string
		value string
	}{
		{"input_tokens", fmt.Sprintf("%d", usage.InputTokens)},
		{"cache_creation_input_tokens", fmt.Sprintf("%d", usage.CacheCreationInputTokens)},
		{"cache_read_input_tokens", fmt.Sprintf("%d", usage.CacheReadInputTokens)},
		{"output_tokens", fmt.Sprintf("%d", usage.OutputTokens)},
	}

	// Only report cost when using an API key (OAuth tokens don't report cost).
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		entries = append(entries, struct {
			key   string
			value string
		}{"total_cost_usd", fmt.Sprintf("$%.4f", evt.TotalCostUSD)})
	}

	for _, e := range entries {
		path := filepath.Join(infoDir, prefix+e.key)
		content := fmt.Sprintf("%s: %s", e.key, e.value)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Logf("WARNING: could not write RWX_INFO %s: %v", e.key, err)
		}
	}
}

// evalContext returns a context with a 15-minute timeout.
func evalContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// setupProjectDir copies an entire fixture directory tree into a temp dir,
// preserving directory structure. Unlike setupWorkDir which targets
// .github/workflows/, this copies to the project root.
func setupProjectDir(t *testing.T, fixtureName string) string {
	t.Helper()

	tmpDir := t.TempDir()

	srcRoot := filepath.Join("testdata", "fixtures", "projects", fixtureName)
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}

		dst := filepath.Join(tmpDir, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copying fixture %s: %v", fixtureName, err)
	}

	return tmpDir
}

// --- Shared assertion helpers ---

// clonesRepo matches either a git/clone package or a git clone run command.
func clonesRepo() evals.ConfigAssertion {
	return evals.Either("clones_repo",
		evals.HasPackage("git/clone"),
		evals.HasRunContaining("git clone"),
	)
}

// installsGo matches package names the agent might use for Go installation.
func installsGo() evals.ConfigAssertion {
	return evals.Either("installs_go",
		evals.HasPackage("golang/install"),
		evals.HasPackage("go/install"),
		evals.HasPackage("rwx/tool-versions"),
	)
}

// installsNode matches common Node.js installation patterns.
func installsNode() evals.ConfigAssertion {
	return evals.Either("installs_node",
		evals.HasPackage("nodejs/install"),
		evals.HasPackage("node/install"),
	)
}

// installsRust matches common Rust installation patterns.
func installsRust() evals.ConfigAssertion {
	return evals.Either("installs_rust",
		evals.HasPackage("rust/install"),
		evals.HasRunContaining("rustup"),
	)
}

// installsPython matches common Python installation patterns.
func installsPython() evals.ConfigAssertion {
	return evals.Either("installs_python",
		evals.HasPackage("python/install"),
		evals.HasRunContaining("python"),
	)
}
