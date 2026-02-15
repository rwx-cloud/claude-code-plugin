package evals

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var update = flag.Bool("update", false, "update baseline snapshots")

// ClaudeEvent is a top-level event from Claude's --output-format json output.
type ClaudeEvent struct {
	Type         string           `json:"type"`
	Subtype      string           `json:"subtype,omitempty"`
	Message      ClaudeMessage    `json:"message"`
	DurationMS   float64          `json:"duration_ms"`
	TotalCostUSD float64          `json:"total_cost_usd,omitempty"`
	Usage        *TokenUsage      `json:"usage,omitempty"`
	ModelUsage   *ModelTokenUsage `json:"model_usage,omitempty"`
	Skills       []string         `json:"skills,omitempty"`
}

// ClaudeMessage contains a role and array of content items (lazily parsed).
type ClaudeMessage struct {
	Role    string            `json:"role"`
	Content []json.RawMessage `json:"content"`
}

// ContentItem is a parsed content block from a Claude message.
type ContentItem struct {
	Type  string          `json:"type"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	Text  string          `json:"text,omitempty"`
}

// SkillInput is the parsed input for a Skill tool invocation.
type SkillInput struct {
	Skill string `json:"skill"`
	Args  string `json:"args,omitempty"`
}

// TokenUsage tracks token counts from a result event.
type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// ModelTokenUsage tracks per-model token usage.
type ModelTokenUsage map[string]TokenUsage

// ExecutionResult holds the parsed output from a Claude headless run.
type ExecutionResult struct {
	Events    []ClaudeEvent
	RawOutput []byte
	Prompt    string
}

// ResultEvent returns the final "result" event, or nil if not found.
func (r *ExecutionResult) ResultEvent() *ClaudeEvent {
	for i := len(r.Events) - 1; i >= 0; i-- {
		if r.Events[i].Type == "result" {
			return &r.Events[i]
		}
	}
	return nil
}

// ToolUses extracts all tool_use content items from all messages.
func (r *ExecutionResult) ToolUses() []ContentItem {
	var items []ContentItem
	for _, event := range r.Events {
		for _, raw := range event.Message.Content {
			var item ContentItem
			if err := json.Unmarshal(raw, &item); err == nil && item.Type == "tool_use" {
				items = append(items, item)
			}
		}
	}
	return items
}

// ToolNames returns deduplicated tool names from all tool_use items.
func (r *ExecutionResult) ToolNames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, item := range r.ToolUses() {
		if item.Name != "" && !seen[item.Name] {
			seen[item.Name] = true
			names = append(names, item.Name)
		}
	}
	return names
}

// SkillUses extracts skill names from Skill tool invocations and slash commands.
// It detects both model-initiated invocations (via the Skill tool) and
// CLI-initiated invocations (via slash command prompts like "/rwx:migrate-from-gha").
func (r *ExecutionResult) SkillUses() []string {
	seen := make(map[string]bool)
	var skills []string

	// Detect model-initiated skill invocations (Skill tool_use).
	for _, item := range r.ToolUses() {
		if item.Name == "Skill" && item.Input != nil {
			var si SkillInput
			if err := json.Unmarshal(item.Input, &si); err == nil && si.Skill != "" && !seen[si.Skill] {
				seen[si.Skill] = true
				skills = append(skills, si.Skill)
			}
		}
	}

	// Detect CLI-initiated skill invocations (slash command prompts).
	// When the prompt starts with "/", the CLI expands the skill directly
	// without going through the Skill tool. Cross-reference against the
	// init event's skills list to verify the skill was actually registered.
	if r.Prompt != "" && strings.HasPrefix(r.Prompt, "/") {
		name := strings.SplitN(r.Prompt[1:], " ", 2)[0]
		if name != "" && !seen[name] && r.isRegisteredSkill(name) {
			seen[name] = true
			skills = append(skills, name)
		}
	}

	return skills
}

// isRegisteredSkill checks whether the given name appears in the init event's
// skills list, confirming the plugin was loaded and the skill was available.
func (r *ExecutionResult) isRegisteredSkill(name string) bool {
	for _, evt := range r.Events {
		if evt.Type == "system" && evt.Subtype == "init" {
			for _, s := range evt.Skills {
				if s == name {
					return true
				}
			}
			return false
		}
	}
	return false
}

// TextOutput returns all text content from assistant messages, concatenated.
func (r *ExecutionResult) TextOutput() string {
	var parts []string
	for _, event := range r.Events {
		if event.Message.Role != "assistant" {
			continue
		}
		for _, raw := range event.Message.Content {
			var item ContentItem
			if err := json.Unmarshal(raw, &item); err == nil && item.Type == "text" && item.Text != "" {
				parts = append(parts, item.Text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// Summary produces a Baseline from the execution result.
// Returns an error if no result event is found (e.g., Claude crashed mid-run).
func (r *ExecutionResult) Summary() (Baseline, error) {
	b := Baseline{
		ToolsUsed:  r.ToolNames(),
		SkillsUsed: r.SkillUses(),
	}
	evt := r.ResultEvent()
	if evt == nil {
		return b, fmt.Errorf("no result event found in Claude output (Claude may have crashed mid-run)")
	}
	b.ExecutionTimeMS = int(evt.DurationMS)
	if evt.Usage != nil {
		b.InputTokens = evt.Usage.InputTokens
		b.CacheCreationInputTokens = evt.Usage.CacheCreationInputTokens
		b.CacheReadInputTokens = evt.Usage.CacheReadInputTokens
		b.OutputTokens = evt.Usage.OutputTokens
	}
	return b, nil
}

// repoRoot walks up from the current working directory to find the repository
// root. It prefers stable workspace markers so task caching does not depend on
// including .git metadata in RWX filters.
func repoRoot() (string, error) {
	if root := os.Getenv("SKILLS_REPO_ROOT"); root != "" {
		if looksLikeRepoRoot(root) {
			return root, nil
		}
		return "", fmt.Errorf("SKILLS_REPO_ROOT=%q does not look like the skills repository root", root)
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if looksLikeRepoRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root (looked for workspace markers: skills/ and evals/)")
		}
		dir = parent
	}
}

func looksLikeRepoRoot(dir string) bool {
	// Primary detection: repository layout expected by these evals.
	if isDir(filepath.Join(dir, "skills")) && isDir(filepath.Join(dir, "evals")) {
		return true
	}

	// Fallback for nonstandard layouts where only .git is available.
	return isDir(filepath.Join(dir, ".git"))
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// DefaultMaxBudgetUSD is the default spend cap per Claude invocation.
// Override with the EVALS_MAX_BUDGET_USD environment variable.
const DefaultMaxBudgetUSD = "5.00"

// RunClaude runs Claude headlessly with the given prompt and working directory.
// It returns the parsed execution result.
func RunClaude(ctx context.Context, prompt string, workDir string) (*ExecutionResult, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, fmt.Errorf("finding repo root: %w", err)
	}

	maxBudget := DefaultMaxBudgetUSD
	if v := os.Getenv("EVALS_MAX_BUDGET_USD"); v != "" {
		maxBudget = v
	}

	args := []string{
		"--print",
		"--output-format", "json",
		"--no-session-persistence",
		"--verbose",
		"--model", "sonnet",
		"--plugin-dir", root,
		"--max-budget-usd", maxBudget,
	}

	// Only skip permission checks in CI or when explicitly opted in.
	// This prevents accidental unrestricted access in local development.
	if os.Getenv("CI") != "" || os.Getenv("EVALS_SKIP_PERMISSIONS") != "" {
		args = append(args, "--dangerously-skip-permissions")
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude exited with error: %w\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}

	raw := bytes.Clone(stdout.Bytes())

	var events []ClaudeEvent
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, fmt.Errorf("parsing claude output: %w\nraw output: %s", err, stdout.String())
	}

	return &ExecutionResult{Events: events, RawOutput: raw, Prompt: prompt}, nil
}
