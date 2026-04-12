package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/leeson1/agent-forge/internal/store"
	"github.com/leeson1/agent-forge/internal/task"
)

func initCLITestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s: %v", args, out, err)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s: %v", args, out, err)
		}
	}
	return dir
}

func TestRunCommandExecutesPipeline(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("AGENT_FORGE_HOME", "")

	repoDir := initCLITestRepo(t)

	mockClaude := filepath.Join(homeDir, "mock-claude.sh")
	script := `#!/bin/bash
PROMPT="$(cat)"
if printf '%s' "$PROMPT" | grep -q "Initializer Agent"; then
  cat > "$PWD/feature_list.json" <<'EOF'
{
  "features": [
    {
      "id": "F001",
      "category": "functional",
      "description": "Create feature file",
      "steps": ["Write a file", "Commit it"],
      "depends_on": [],
      "batch": null,
      "passes": false
    }
  ]
}
EOF
  cat > "$PWD/init.sh" <<'EOF'
#!/bin/bash
echo init
EOF
  chmod +x "$PWD/init.sh"
  echo "Initialization complete." > "$PWD/progress.txt"
  echo '{"type":"system","subtype":"init","session_id":"init-session"}'
  echo '{"type":"result","subtype":"success","is_error":false,"result":"init done","session_id":"init-session","usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
  exit 0
fi

FEATURE_NAME="$(basename "$PWD")"
echo "$FEATURE_NAME" > "$PWD/$FEATURE_NAME.txt"
git add "$PWD/$FEATURE_NAME.txt"
git commit -m "feat: $FEATURE_NAME" >/dev/null
echo '{"type":"system","subtype":"init","session_id":"worker-session"}'
echo '{"type":"result","subtype":"success","is_error":false,"result":"worker done","session_id":"worker-session","usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`
	if err := os.WriteFile(mockClaude, []byte(script), 0755); err != nil {
		t.Fatalf("write mock claude: %v", err)
	}

	configDir := filepath.Join(homeDir, ".agent-forge")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configJSON := `{"cli":{"claude_path":"` + mockClaude + `","default_timeout":"30s"}}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := store.Init(); err != nil {
		t.Fatalf("store.Init failed: %v", err)
	}
	taskStore := store.NewTaskStore(store.BaseDir())
	taskID := "task-cli-run"
	tsk := task.NewTask(taskID, "CLI Task", "desc", "default", task.TaskConfig{
		MaxParallelWorkers: 1,
		SessionTimeout:     "30s",
		WorkspaceDir:       repoDir,
	})
	if err := taskStore.Create(tsk); err != nil {
		t.Fatalf("create task: %v", err)
	}

	cmd := newRunCmd()
	if err := cmd.Flags().Set("follow", "false"); err != nil {
		t.Fatalf("set follow flag: %v", err)
	}
	if err := runRun(cmd, []string{taskID}); err != nil {
		t.Fatalf("runRun failed: %v", err)
	}

	updated, err := taskStore.Get(taskID)
	if err != nil {
		t.Fatalf("Get task failed: %v", err)
	}
	if updated.Status != task.StatusCompleted {
		t.Fatalf("Expected completed task, got %s", updated.Status)
	}

	sessionStore := store.NewSessionStore(store.BaseDir())
	sessions, err := sessionStore.List(taskID)
	if err != nil {
		t.Fatalf("List sessions failed: %v", err)
	}
	if len(sessions) < 2 {
		t.Fatalf("Expected initializer and worker sessions, got %d", len(sessions))
	}

	if _, err := os.Stat(filepath.Join(repoDir, "F001.txt")); err != nil {
		t.Fatalf("Expected merged feature file in repo: %v", err)
	}
}
