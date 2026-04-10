package template

import (
	"testing"
	"time"
)

func TestRunHook_EmptyScript(t *testing.T) {
	result := RunHook("", HookEnv{}, 10*time.Second)
	if !result.Success {
		t.Error("Empty script should succeed")
	}
}

func TestRunHook_SimpleScript(t *testing.T) {
	script := `#!/bin/bash
echo "hello from hook"
echo "TASK_ID=$TASK_ID"
`
	env := HookEnv{
		TaskID:       "task-123",
		SessionID:    "sess-456",
		WorkspaceDir: t.TempDir(),
	}

	result := RunHook(script, env, 10*time.Second)
	if !result.Success {
		t.Fatalf("Hook should succeed: %v", result.Error)
	}
	if result.Output == "" {
		t.Error("Output should not be empty")
	}
}

func TestRunHook_FailingScript(t *testing.T) {
	script := `#!/bin/bash
exit 1
`
	env := HookEnv{
		WorkspaceDir: t.TempDir(),
	}

	result := RunHook(script, env, 10*time.Second)
	if result.Success {
		t.Error("Failing script should not succeed")
	}
	if result.Error == nil {
		t.Error("Error should not be nil")
	}
}

func TestRunHook_Timeout(t *testing.T) {
	script := `#!/bin/bash
sleep 60
`
	env := HookEnv{
		WorkspaceDir: t.TempDir(),
	}

	result := RunHook(script, env, 500*time.Millisecond)
	if result.Success {
		t.Error("Timed out script should not succeed")
	}
	if result.Error == nil {
		t.Error("Error should not be nil for timeout")
	}
}

func TestRunHook_EnvVars(t *testing.T) {
	script := `#!/bin/bash
echo "task=$TASK_ID session=$SESSION_ID feature=$FEATURE_ID batch=$BATCH_NUM custom=$CUSTOM_VAR"
`
	env := HookEnv{
		TaskID:       "T1",
		SessionID:    "S1",
		WorkspaceDir: t.TempDir(),
		FeatureID:    "F001",
		BatchNum:     2,
		Extra: map[string]string{
			"CUSTOM_VAR": "hello",
		},
	}

	result := RunHook(script, env, 10*time.Second)
	if !result.Success {
		t.Fatalf("Hook should succeed: %v", result.Error)
	}

	if result.Output == "" {
		t.Error("Output should contain env vars")
	}
}

func TestRunSessionStartHook(t *testing.T) {
	tmpl := &Template{
		HookStartScript: `#!/bin/bash
echo "session start hook"
`,
	}
	env := HookEnv{
		TaskID:       "T1",
		WorkspaceDir: t.TempDir(),
	}

	result := RunSessionStartHook(tmpl, env)
	if !result.Success {
		t.Fatalf("Session start hook should succeed: %v", result.Error)
	}
}

func TestRunSessionEndHook_NoScript(t *testing.T) {
	tmpl := &Template{}
	env := HookEnv{WorkspaceDir: t.TempDir()}

	result := RunSessionEndHook(tmpl, env)
	if !result.Success {
		t.Error("No script should succeed")
	}
}

func TestRunValidator(t *testing.T) {
	tmpl := &Template{
		ValidatorScript: `#!/bin/bash
echo "[PASS] All checks passed"
exit 0
`,
	}
	env := HookEnv{
		WorkspaceDir: t.TempDir(),
	}

	result := RunValidator(tmpl, env)
	if !result.Success {
		t.Fatalf("Validator should succeed: %v", result.Error)
	}
}
