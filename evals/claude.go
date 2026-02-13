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
	Type       string          `json:"type"`
	Message    ClaudeMessage   `json:"message"`
	DurationMS float64         `json:"duration_ms"`
	Usage      *TokenUsage     `json:"usage,omitempty"`
	ModelUsage *ModelTokenUsage `json:"model_usage,omitempty"`
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
	InputTokens                int `json:"input_tokens"`
	CacheCreationInputTokens   int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens       int `json:"cache_read_input_tokens"`
	OutputTokens               int `json:"output_tokens"`
}

// ModelTokenUsage tracks per-model token usage.
type ModelTokenUsage map[string]TokenUsage

// ExecutionResult holds the parsed output from a Claude headless run.
type ExecutionResult struct {
	Events    []ClaudeEvent
	RawOutput []byte
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

// SkillUses extracts skill names from Skill tool invocations.
func (r *ExecutionResult) SkillUses() []string {
	seen := make(map[string]bool)
	var skills []string
	for _, item := range r.ToolUses() {
		if item.Name == "Skill" && item.Input != nil {
			var si SkillInput
			if err := json.Unmarshal(item.Input, &si); err == nil && si.Skill != "" && !seen[si.Skill] {
				seen[si.Skill] = true
				skills = append(skills, si.Skill)
			}
		}
	}
	return skills
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

// repoRoot walks up from the current working directory to find the repository root
// (identified by the presence of a .git directory).
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root (no .git directory found)")
		}
		dir = parent
	}
}

// RunClaude runs Claude headlessly with the given prompt and working directory.
// It returns the parsed execution result.
func RunClaude(ctx context.Context, prompt string, workDir string) (*ExecutionResult, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, fmt.Errorf("finding repo root: %w", err)
	}

	args := []string{
		"--print",
		"--output-format", "json",
		"--no-session-persistence",
		"--verbose",
		"--model", "sonnet",
		"--plugin-dir", root,
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

	return &ExecutionResult{Events: events, RawOutput: raw}, nil
}
