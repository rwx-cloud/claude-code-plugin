package evals

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RWXConfig represents a parsed RWX configuration file.
type RWXConfig struct {
	Tasks []RWXTask `yaml:"tasks"`
}

// RWXTask represents a single task in an RWX config.
// Fields use flexible types where LLM-generated output may produce maps instead
// of scalars. The real schema is enforced by rwx lint; this struct is for
// eval assertions only.
type RWXTask struct {
	Key                 string         `yaml:"key"`
	Call                string         `yaml:"call,omitempty"`
	Run                 any            `yaml:"run,omitempty"`
	Use                 FlexStrings    `yaml:"use,omitempty"`
	With                map[string]any `yaml:"with,omitempty"`
	Env                 map[string]any `yaml:"env,omitempty"`
	If                  string         `yaml:"if,omitempty"`
	Filter              FlexStrings    `yaml:"filter,omitempty"`
	Parallel            any            `yaml:"parallel,omitempty"`
	BackgroundProcesses []BGProcess    `yaml:"background-processes,omitempty"`
	Outputs             any            `yaml:"outputs,omitempty"`
}

// RunString returns the run field as a string. If Claude generated a map
// instead of a string, it returns the fmt.Sprint representation.
func (t *RWXTask) RunString() string {
	if s, ok := t.Run.(string); ok {
		return s
	}
	if t.Run == nil {
		return ""
	}
	return fmt.Sprintf("%v", t.Run)
}

// EnvString returns the value of an env var as a string.
func (t *RWXTask) EnvString(key string) (string, bool) {
	v, ok := t.Env[key]
	if !ok {
		return "", false
	}
	if s, ok := v.(string); ok {
		return s, true
	}
	return fmt.Sprintf("%v", v), true
}

// FlexStrings handles YAML fields that can be either a single string or a list of strings.
type FlexStrings []string

func (f *FlexStrings) UnmarshalYAML(unmarshal func(any) error) error {
	var list []string
	if err := unmarshal(&list); err == nil {
		*f = list
		return nil
	}
	var single string
	if err := unmarshal(&single); err == nil {
		*f = []string{single}
		return nil
	}
	return fmt.Errorf("expected string or list of strings")
}

// BGProcess represents a background process (service) in an RWX task.
type BGProcess struct {
	Key        string `yaml:"key"`
	Run        any    `yaml:"run,omitempty"`
	ReadyCheck any    `yaml:"ready-check,omitempty"`
}

// RunString returns the run field as a string.
func (bp *BGProcess) RunString() string {
	if s, ok := bp.Run.(string); ok {
		return s
	}
	if bp.Run == nil {
		return ""
	}
	return fmt.Sprintf("%v", bp.Run)
}

// ParseRWXConfig parses an RWX config from raw YAML bytes.
func ParseRWXConfig(data []byte) (*RWXConfig, error) {
	var cfg RWXConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing RWX config: %w", err)
	}
	return &cfg, nil
}

// LoadRWXConfigs finds and parses all .rwx/*.yml files in the given directory.
func LoadRWXConfigs(workDir string) ([]*RWXConfig, error) {
	pattern := filepath.Join(workDir, ".rwx", "*.yml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing for RWX configs: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .rwx/*.yml files found in %s", workDir)
	}

	var configs []*RWXConfig
	for _, f := range matches {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}
		cfg, err := ParseRWXConfig(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", f, err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// Task returns the first task matching the given key, or nil.
func (c *RWXConfig) Task(key string) *RWXTask {
	for i := range c.Tasks {
		if c.Tasks[i].Key == key {
			return &c.Tasks[i]
		}
	}
	return nil
}

// TaskKeys returns all task keys in order.
func (c *RWXConfig) TaskKeys() []string {
	keys := make([]string, len(c.Tasks))
	for i, t := range c.Tasks {
		keys[i] = t.Key
	}
	return keys
}

// HasTaskWithCall returns true if any task uses the given package call prefix.
// For example, HasTaskWithCall("golang/install") matches "golang/install 1.2.0".
func (c *RWXConfig) HasTaskWithCall(callPrefix string) bool {
	for _, t := range c.Tasks {
		if t.Call == callPrefix || strings.HasPrefix(t.Call, callPrefix+" ") {
			return true
		}
	}
	return false
}

// TasksWithRun returns all tasks whose run field contains the given substring.
func (c *RWXConfig) TasksWithRun(substr string) []RWXTask {
	var matches []RWXTask
	for _, t := range c.Tasks {
		if strings.Contains(t.RunString(), substr) {
			matches = append(matches, t)
		}
	}
	return matches
}

// HasBackgroundProcess returns true if any task has a background process
// whose key or run field contains the given substring.
func (c *RWXConfig) HasBackgroundProcess(substr string) bool {
	for _, t := range c.Tasks {
		for _, bp := range t.BackgroundProcesses {
			if strings.Contains(bp.Key, substr) || strings.Contains(bp.RunString(), substr) {
				return true
			}
		}
	}
	return false
}

// DependsOn returns true if the task with the given key lists dep in its use array.
func (c *RWXConfig) DependsOn(taskKey, dep string) bool {
	t := c.Task(taskKey)
	if t == nil {
		return false
	}
	for _, u := range t.Use {
		if u == dep {
			return true
		}
	}
	return false
}
