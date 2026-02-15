package evals

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Baseline holds a performance snapshot for an eval test.
type Baseline struct {
	InputTokens              int      `json:"input_tokens"`
	CacheCreationInputTokens int      `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int      `json:"cache_read_input_tokens"`
	OutputTokens             int      `json:"output_tokens"`
	ExecutionTimeMS          int      `json:"execution_time_ms"`
	ToolsUsed                []string `json:"tools_used"`
	SkillsUsed               []string `json:"skills_used"`
}

func baselinesDir() string {
	return filepath.Join("testdata", "baselines")
}

func baselinePath(testName string) string {
	return filepath.Join(baselinesDir(), testName+".json")
}

// LoadBaseline reads a baseline from testdata/baselines/<testName>.json.
// Returns nil if the file does not exist.
func LoadBaseline(testName string) (*Baseline, error) {
	data, err := os.ReadFile(baselinePath(testName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading baseline: %w", err)
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parsing baseline: %w", err)
	}
	return &b, nil
}

// SaveBaseline writes a baseline to testdata/baselines/<testName>.json.
func SaveBaseline(testName string, b Baseline) error {
	if err := os.MkdirAll(baselinesDir(), 0o755); err != nil {
		return fmt.Errorf("creating baselines dir: %w", err)
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling baseline: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(baselinePath(testName), data, 0o644); err != nil {
		return fmt.Errorf("writing baseline: %w", err)
	}
	return nil
}

// AssertNoRegression compares the current result against a saved baseline.
// In -update mode, it saves the new baseline. Otherwise, it checks that
// metrics haven't regressed beyond allowed thresholds.
func AssertNoRegression(t *testing.T, result *ExecutionResult) {
	t.Helper()

	current, err := result.Summary()
	if err != nil {
		t.Fatalf("extracting summary: %v", err)
	}

	if *update {
		if err := SaveBaseline(t.Name(), current); err != nil {
			t.Fatalf("saving baseline: %v", err)
		}
		t.Logf("updated baseline for %s", t.Name())
		return
	}

	prev, err2 := LoadBaseline(t.Name())
	if err2 != nil {
		t.Fatalf("loading baseline: %v", err2)
	}
	if prev == nil {
		t.Logf("WARNING: no baseline found for %s — skipping regression check (run with -update to create)", t.Name())
		return
	}

	checkThreshold(t, "input_tokens", prev.InputTokens, current.InputTokens, 0.20)
	checkThreshold(t, "cache_read_input_tokens", prev.CacheReadInputTokens, current.CacheReadInputTokens, 0.20)
	checkThreshold(t, "output_tokens", prev.OutputTokens, current.OutputTokens, 0.30)
	checkThreshold(t, "execution_time_ms", prev.ExecutionTimeMS, current.ExecutionTimeMS, 0.50)
}

func checkThreshold(t *testing.T, metric string, baseline, current int, maxIncrease float64) {
	t.Helper()
	if baseline == 0 {
		return
	}
	diff := current - baseline
	increase := float64(diff) / float64(baseline)
	// Skip percentage-based checks when the absolute difference is trivially
	// small — LLM token counts naturally fluctuate by small amounts between
	// runs, and percentage thresholds are unreliable for small baselines.
	if diff <= 50 {
		return
	}
	if increase > maxIncrease {
		t.Errorf("%s regressed: baseline=%d, current=%d (%.0f%% increase, max allowed %.0f%%)",
			metric, baseline, current, increase*100, maxIncrease*100)
	}
}
