package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leeson1/agent-forge/internal/session"
	"github.com/leeson1/agent-forge/internal/store"
	"github.com/leeson1/agent-forge/internal/task"
	"github.com/leeson1/agent-forge/internal/template"
)

func setupWorkerTest(t *testing.T, mockScriptContent string) (*Worker, string, string) {
	t.Helper()

	baseDir := t.TempDir()
	repoDir := initTestRepo(t)

	mockScript := filepath.Join(baseDir, "mock-claude-worker.sh")
	if err := os.WriteFile(mockScript, []byte(mockScriptContent), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	taskStore := store.NewTaskStore(baseDir)
	sessionStore := store.NewSessionStore(baseDir)
	logStore := store.NewLogStore(baseDir)

	tsk := task.NewTask("test-task", "Test", "desc", "default", task.TaskConfig{WorkspaceDir: repoDir})
	if err := taskStore.Create(tsk); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	config := session.ExecutorConfig{
		ClaudePath: mockScript,
		MaxTurns:   10,
		Timeout:    15 * time.Second,
		MaxRetries: 1,
	}
	executor := session.NewExecutor(baseDir, config)

	worker := NewWorker(executor, taskStore, sessionStore, logStore)
	return worker, baseDir, repoDir
}

func TestWorker_RunSuccess(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null
FEATURE_FILE="$PWD/feature.txt"
echo "implemented" > "$FEATURE_FILE"
git add "$FEATURE_FILE"
git commit -m "feat: implement worker feature" >/dev/null
echo '{"type":"system","subtype":"init","session_id":"w-test"}'
echo '{"type":"assistant","message":{"id":"msg_01","role":"assistant","content":[{"type":"text","text":"Feature implemented."}],"usage":{"input_tokens":50,"output_tokens":30,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"session_id":"w-test"}'
echo '{"type":"result","subtype":"success","is_error":false,"duration_ms":1000,"num_turns":1,"result":"done","stop_reason":"end_turn","total_cost_usd":0.02,"session_id":"w-test","usage":{"input_tokens":50,"output_tokens":30,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	worker, _, workDir := setupWorkerTest(t, mockScript)
	baseCommit, err := session.HeadCommit(workDir)
	if err != nil {
		t.Fatalf("HeadCommit failed: %v", err)
	}

	result := worker.Run(WorkerConfig{
		TaskID:          "test-task",
		TaskName:        "Test Project",
		Feature:         task.Feature{ID: "F001", Category: "functional", Description: "Setup project", Steps: []string{"Step 1", "Step 2"}},
		BatchNum:        0,
		SessionNumber:   1,
		WorkDir:         workDir,
		ProgressContent: "Initial progress",
		PendingFeatures: "- F002: Next feature",
		Branch:          "agentforge/test-task/F001",
		BaseCommit:      baseCommit,
	})

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}
	if result.HeadCommit == "" {
		t.Fatal("Expected head commit to be recorded")
	}
	if len(result.ChangedFiles) == 0 {
		t.Fatal("Expected changed files to be recorded")
	}
}

func TestWorker_RunFailure_NoCommit(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null
echo '{"type":"system","subtype":"init","session_id":"w-nocommit"}'
echo '{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"w-nocommit","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	worker, _, workDir := setupWorkerTest(t, mockScript)
	baseCommit, err := session.HeadCommit(workDir)
	if err != nil {
		t.Fatalf("HeadCommit failed: %v", err)
	}

	result := worker.Run(WorkerConfig{
		TaskID:        "test-task",
		TaskName:      "Test",
		Feature:       task.Feature{ID: "F001", Category: "functional", Description: "test"},
		WorkDir:       workDir,
		SessionNumber: 1,
		Branch:        "agentforge/test-task/F001",
		BaseCommit:    baseCommit,
	})

	if result.Success {
		t.Fatal("Expected no-commit worker to fail")
	}
	if !strings.Contains(result.Error, "without creating a commit") {
		t.Fatalf("Unexpected error: %s", result.Error)
	}
}

func TestWorker_RunFailure_DirtyTree(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null
echo "committed" > "$PWD/feature.txt"
git add "$PWD/feature.txt"
git commit -m "feat: committed file" >/dev/null
echo "dirty" > "$PWD/untracked.txt"
echo '{"type":"system","subtype":"init","session_id":"w-dirty"}'
echo '{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"w-dirty","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	worker, _, workDir := setupWorkerTest(t, mockScript)
	baseCommit, err := session.HeadCommit(workDir)
	if err != nil {
		t.Fatalf("HeadCommit failed: %v", err)
	}

	result := worker.Run(WorkerConfig{
		TaskID:        "test-task",
		TaskName:      "Test",
		Feature:       task.Feature{ID: "F001", Category: "functional", Description: "test"},
		WorkDir:       workDir,
		SessionNumber: 1,
		Branch:        "agentforge/test-task/F001",
		BaseCommit:    baseCommit,
	})

	if result.Success {
		t.Fatal("Expected dirty-tree worker to fail")
	}
	if !strings.Contains(result.Error, "left the worktree dirty") {
		t.Fatalf("Unexpected error: %s", result.Error)
	}
}

func TestWorker_RunFailure_Validator(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null
echo "implemented" > "$PWD/feature.txt"
git add "$PWD/feature.txt"
git commit -m "feat: validator case" >/dev/null
echo '{"type":"system","subtype":"init","session_id":"w-validator"}'
echo '{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"w-validator","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	worker, _, workDir := setupWorkerTest(t, mockScript)
	baseCommit, err := session.HeadCommit(workDir)
	if err != nil {
		t.Fatalf("HeadCommit failed: %v", err)
	}

	result := worker.Run(WorkerConfig{
		TaskID:        "test-task",
		TaskName:      "Test",
		Feature:       task.Feature{ID: "F001", Category: "functional", Description: "test"},
		WorkDir:       workDir,
		SessionNumber: 1,
		Branch:        "agentforge/test-task/F001",
		BaseCommit:    baseCommit,
		Template: &template.Template{
			Config:          template.TemplateConfig{ID: "validator", Name: "Validator"},
			InitializerPrompt: template.DefaultInitializerPrompt,
			WorkerPrompt:      template.DefaultWorkerPrompt,
			ValidatorScript: "#!/bin/bash\nexit 1\n",
		},
	})

	if result.Success {
		t.Fatal("Expected validator failure")
	}
	if !strings.Contains(result.Error, "validator failed") {
		t.Fatalf("Unexpected error: %s", result.Error)
	}
}

func TestFormatPendingFeatures(t *testing.T) {
	features := []task.Feature{
		{ID: "F001", Description: "Setup", DependsOn: []string{}},
		{ID: "F002", Description: "Core", DependsOn: []string{"F001"}},
	}

	result := FormatPendingFeatures(features)
	if result == "" {
		t.Error("Result should not be empty")
	}
	if !strings.Contains(result, "F001") || !strings.Contains(result, "F002") {
		t.Errorf("Result should contain feature IDs: %s", result)
	}
}

func TestFormatPendingFeatures_Empty(t *testing.T) {
	result := FormatPendingFeatures(nil)
	if result != "No pending features." {
		t.Errorf("got %q, want 'No pending features.'", result)
	}
}
