package evals

import (
	"fmt"
	"strings"
	"testing"
)

// TB is the subset of testing.TB used by assertion checks.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
}

// ConfigAssertion is a named check against a parsed RWX config.
type ConfigAssertion struct {
	Name  string
	Check func(TB, *RWXConfig)
}

// AssertConfig runs a set of named assertions against all RWX configs in workDir.
// It loads and merges configs, then runs each assertion as a subtest.
func AssertConfig(t *testing.T, workDir string, assertions []ConfigAssertion) {
	t.Helper()

	configs, err := LoadRWXConfigs(workDir)
	if err != nil {
		t.Fatalf("loading RWX configs: %v", err)
	}

	// Merge all configs into one for assertion purposes.
	merged := &RWXConfig{}
	for _, cfg := range configs {
		merged.Tasks = append(merged.Tasks, cfg.Tasks...)
	}

	for _, a := range assertions {
		t.Run(a.Name, func(t *testing.T) {
			a.Check(t, merged)
		})
	}
}

// --- Assertion constructors ---

// HasTask asserts a task with the given key exists.
func HasTask(key string) ConfigAssertion {
	return ConfigAssertion{
		Name: "has_task_" + key,
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			if cfg.Task(key) == nil {
				t.Errorf("expected task %q to exist, got tasks: %v", key, cfg.TaskKeys())
			}
		},
	}
}

// HasPackage asserts some task calls a package with the given prefix.
func HasPackage(callPrefix string) ConfigAssertion {
	return ConfigAssertion{
		Name: "has_package_" + sanitizeName(callPrefix),
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			if !cfg.HasTaskWithCall(callPrefix) {
				var calls []string
				for _, task := range cfg.Tasks {
					if task.Call != "" {
						calls = append(calls, task.Call)
					}
				}
				t.Errorf("expected a task calling %q, got calls: %v", callPrefix, calls)
			}
		},
	}
}

// HasRunContaining asserts at least one task's run field contains the substring.
func HasRunContaining(substr string) ConfigAssertion {
	return ConfigAssertion{
		Name: "has_run_" + sanitizeName(substr),
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			if len(cfg.TasksWithRun(substr)) == 0 {
				var runs []string
				for _, task := range cfg.Tasks {
					if r := task.RunString(); r != "" {
						runs = append(runs, task.Key+": "+firstLine(r))
					}
				}
				t.Errorf("expected a task with run containing %q, got: %v", substr, runs)
			}
		},
	}
}

// TaskDependsOn asserts that the task with taskKey lists dep in its use array.
func TaskDependsOn(taskKey, dep string) ConfigAssertion {
	return ConfigAssertion{
		Name: "task_" + taskKey + "_depends_on_" + dep,
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			task := cfg.Task(taskKey)
			if task == nil {
				t.Errorf("task %q does not exist", taskKey)
				return
			}
			if !cfg.DependsOn(taskKey, dep) {
				t.Errorf("expected task %q to depend on %q, got use: %v", taskKey, dep, task.Use)
			}
		},
	}
}

// HasService asserts that some task has a background process matching the substring.
func HasService(substr string) ConfigAssertion {
	return ConfigAssertion{
		Name: "has_service_" + sanitizeName(substr),
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			if !cfg.HasBackgroundProcess(substr) {
				t.Errorf("expected a background process matching %q, found none", substr)
			}
		},
	}
}

// HasEnvVar asserts that some task references the given env var.
// Checks task-level env blocks and inline assignments in run fields
// (e.g. "export FOO=" or "FOO=").
func HasEnvVar(envKey string) ConfigAssertion {
	return ConfigAssertion{
		Name: "has_env_" + sanitizeName(envKey),
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			for _, task := range cfg.Tasks {
				if _, ok := task.Env[envKey]; ok {
					return
				}
				if strings.Contains(task.RunString(), envKey+"=") {
					return
				}
			}
			t.Errorf("expected some task to have env var %q, found none", envKey)
		},
	}
}

// HasSecretRef asserts that some task references the given secret name.
// Matches both GHA-style (${{ secrets.<name> }}) and RWX vault-style
// (${{ vaults.<vault>.secrets.<name> }}) references in env, with, and run fields.
func HasSecretRef(secretName string) ConfigAssertion {
	return ConfigAssertion{
		Name: "has_secret_" + sanitizeName(secretName),
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			needle := "secrets." + secretName
			for _, task := range cfg.Tasks {
				if strings.Contains(task.RunString(), needle) {
					return
				}
				for _, v := range task.Env {
					if s, ok := v.(string); ok && strings.Contains(s, needle) {
						return
					}
				}
				for _, v := range task.With {
					if s, ok := v.(string); ok && strings.Contains(s, needle) {
						return
					}
				}
			}
			t.Errorf("expected some task to reference secret %q, found none", secretName)
		},
	}
}

// HasConditional asserts that the task with the given key has an if field.
func HasConditional(taskKey string) ConfigAssertion {
	return ConfigAssertion{
		Name: "task_" + taskKey + "_has_conditional",
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			task := cfg.Task(taskKey)
			if task == nil {
				t.Errorf("task %q does not exist", taskKey)
				return
			}
			if task.If == "" {
				t.Errorf("expected task %q to have a conditional (if field), but it was empty", taskKey)
			}
		},
	}
}

// MinTaskCount asserts the config has at least n tasks.
func MinTaskCount(n int) ConfigAssertion {
	return ConfigAssertion{
		Name: fmt.Sprintf("min_task_count_%d", n),
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			if len(cfg.Tasks) < n {
				t.Errorf("expected at least %d tasks, got %d: %v", n, len(cfg.Tasks), cfg.TaskKeys())
			}
		},
	}
}

// Either passes if at least one of the given assertions passes.
func Either(name string, assertions ...ConfigAssertion) ConfigAssertion {
	return ConfigAssertion{
		Name: name,
		Check: func(t TB, cfg *RWXConfig) {
			t.Helper()
			var names []string
			for _, a := range assertions {
				probe := &probeTB{}
				a.Check(probe, cfg)
				if !probe.failed {
					return
				}
				names = append(names, a.Name)
			}
			t.Errorf("none of the alternatives passed: %v", names)
		},
	}
}

// probeTB captures assertion failures without propagating them.
// Used by Either() at runtime to test alternatives without failing the real test.
type probeTB struct{ failed bool }

func (m *probeTB) Helper()               {}
func (m *probeTB) Errorf(string, ...any)  { m.failed = true }

func sanitizeName(s string) string {
	r := strings.NewReplacer(
		"/", "_", " ", "_", ".", "_", "-", "_",
		"$", "", "{", "", "}", "",
	)
	return r.Replace(s)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
