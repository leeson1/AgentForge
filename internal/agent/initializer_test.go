package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leeson1/agent-forge/internal/session"
	"github.com/leeson1/agent-forge/internal/store"
	"github.com/leeson1/agent-forge/internal/task"
)

// setupInitializerTest 创建测试环境
func setupInitializerTest(t *testing.T, mockScriptContent string) (*Initializer, *task.Task, string) {
	t.Helper()

	baseDir := t.TempDir()
	workDir := t.TempDir()

	// 创建 mock claude 脚本
	mockScript := filepath.Join(baseDir, "mock-claude-init.sh")
	if err := os.WriteFile(mockScript, []byte(mockScriptContent), 0755); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	// 创建 stores
	taskStore := store.NewTaskStore(baseDir)
	sessionStore := store.NewSessionStore(baseDir)
	logStore := store.NewLogStore(baseDir)

	// 创建 executor
	config := session.ExecutorConfig{
		ClaudePath: mockScript,
		MaxTurns:   10,
		Timeout:    15 * time.Second,
		MaxRetries: 1,
	}
	executor := session.NewExecutor(baseDir, config)

	// 创建任务
	taskConfig := task.TaskConfig{
		MaxParallelWorkers: 2,
		SessionTimeout:     "10m",
		WorkspaceDir:       workDir,
	}
	tsk := task.NewTask("test-task-1", "Test Project", "A test project for unit testing", "default", taskConfig)
	if err := taskStore.Create(tsk); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	initializer := NewInitializer(executor, taskStore, sessionStore, logStore)
	return initializer, tsk, workDir
}

func TestInitializer_RunSuccess(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null

WORK_DIR="$PWD"

cat > "$WORK_DIR/feature_list.json" << 'FEATURE_EOF'
{
  "features": [
    {
      "id": "F001",
      "category": "functional",
      "description": "Setup project structure",
      "steps": ["Create directories", "Initialize configs"],
      "depends_on": [],
      "batch": null,
      "passes": false
    },
    {
      "id": "F002",
      "category": "functional",
      "description": "Implement core logic",
      "steps": ["Write code", "Add tests"],
      "depends_on": ["F001"],
      "batch": null,
      "passes": false
    }
  ]
}
FEATURE_EOF

cat > "$WORK_DIR/init.sh" << 'INIT_EOF'
#!/bin/bash
echo "Initializing..."
INIT_EOF
chmod +x "$WORK_DIR/init.sh"

echo "Project: Test Project
Status: Initialization complete." > "$WORK_DIR/progress.txt"

echo '{"type":"system","subtype":"init","session_id":"test-init"}'
echo '{"type":"assistant","message":{"id":"msg_01","role":"assistant","content":[{"type":"text","text":"I have created the required files."}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"session_id":"test-init"}'
echo '{"type":"result","subtype":"success","is_error":false,"duration_ms":2000,"num_turns":2,"result":"Files created successfully","stop_reason":"end_turn","total_cost_usd":0.05,"session_id":"test-init","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	initializer, tsk, _ := setupInitializerTest(t, mockScript)

	result, err := initializer.Run(tsk, nil)
	if err != nil {
		t.Fatalf("Initializer.Run failed: %v", err)
	}

	if result.FeatureList == nil {
		t.Fatal("FeatureList should not be nil")
	}
	if len(result.FeatureList.Features) != 2 {
		t.Errorf("Features count: got %d, want 2", len(result.FeatureList.Features))
	}
	if result.Session == nil {
		t.Fatal("Session should not be nil")
	}
	if result.Error != "" {
		t.Errorf("Error should be empty, got: %s", result.Error)
	}

	if tsk.Status != task.StatusPlanning {
		t.Errorf("Task status: got %s, want %s", tsk.Status, task.StatusPlanning)
	}
	if tsk.Progress.FeaturesTotal != 2 {
		t.Errorf("FeaturesTotal: got %d, want 2", tsk.Progress.FeaturesTotal)
	}
	if tsk.Progress.TotalSessions != 1 {
		t.Errorf("TotalSessions: got %d, want 1", tsk.Progress.TotalSessions)
	}
}

func TestInitializer_RunFailure_SessionFailed(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null
echo '{"type":"system","subtype":"init","session_id":"test-fail"}'
echo '{"type":"result","subtype":"error","is_error":true,"result":"Something went wrong","session_id":"test-fail","usage":{"input_tokens":10,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
exit 1
`

	initializer, tsk, _ := setupInitializerTest(t, mockScript)

	result, err := initializer.Run(tsk, nil)
	if err == nil {
		t.Fatal("Expected error from Initializer.Run")
	}

	if result == nil {
		t.Fatal("Result should not be nil even on failure")
	}
	if result.Error == "" {
		t.Error("Result.Error should contain error message")
	}
	if tsk.Status != task.StatusFailed {
		t.Errorf("Task status: got %s, want %s", tsk.Status, task.StatusFailed)
	}
}

func TestInitializer_RunFailure_MissingFeatureList(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null

WORK_DIR="$PWD"

cat > "$WORK_DIR/init.sh" << 'EOF'
#!/bin/bash
echo "init"
EOF
chmod +x "$WORK_DIR/init.sh"
echo "progress" > "$WORK_DIR/progress.txt"

echo '{"type":"system","subtype":"init","session_id":"test-missing"}'
echo '{"type":"result","subtype":"success","is_error":false,"duration_ms":1000,"num_turns":1,"result":"done","stop_reason":"end_turn","total_cost_usd":0.01,"session_id":"test-missing","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	initializer, tsk, _ := setupInitializerTest(t, mockScript)

	_, err := initializer.Run(tsk, nil)
	if err == nil {
		t.Fatal("Expected error for missing feature_list.json")
	}
	if tsk.Status != task.StatusFailed {
		t.Errorf("Task status: got %s, want %s", tsk.Status, task.StatusFailed)
	}
}

func TestInitializer_RunFailure_MissingInitSh(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null

WORK_DIR="$PWD"

cat > "$WORK_DIR/feature_list.json" << 'EOF'
{"features":[{"id":"F001","category":"functional","description":"test","steps":["step"],"depends_on":[],"batch":null,"passes":false}]}
EOF
echo "progress" > "$WORK_DIR/progress.txt"

echo '{"type":"system","subtype":"init","session_id":"test-noinit"}'
echo '{"type":"result","subtype":"success","is_error":false,"duration_ms":1000,"num_turns":1,"result":"done","stop_reason":"end_turn","total_cost_usd":0.01,"session_id":"test-noinit","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	initializer, tsk, _ := setupInitializerTest(t, mockScript)

	_, err := initializer.Run(tsk, nil)
	if err == nil {
		t.Fatal("Expected error for missing init.sh")
	}
	if tsk.Status != task.StatusFailed {
		t.Errorf("Task status: got %s, want %s", tsk.Status, task.StatusFailed)
	}
}

func TestInitializer_RunFailure_NonExecutableInitSh(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null

WORK_DIR="$PWD"

cat > "$WORK_DIR/feature_list.json" << 'EOF'
{"features":[{"id":"F001","category":"functional","description":"test","steps":["step"],"depends_on":[],"batch":null,"passes":false}]}
EOF

echo "#!/bin/bash" > "$WORK_DIR/init.sh"
echo "progress" > "$WORK_DIR/progress.txt"

echo '{"type":"system","subtype":"init","session_id":"test-noexec"}'
echo '{"type":"result","subtype":"success","is_error":false,"duration_ms":1000,"num_turns":1,"result":"done","stop_reason":"end_turn","total_cost_usd":0.01,"session_id":"test-noexec","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	initializer, tsk, _ := setupInitializerTest(t, mockScript)

	_, err := initializer.Run(tsk, nil)
	if err == nil {
		t.Fatal("Expected error for non-executable init.sh")
	}
	if tsk.Status != task.StatusFailed {
		t.Errorf("Task status: got %s, want %s", tsk.Status, task.StatusFailed)
	}
}

func TestInitializer_RunFailure_InvalidFeatureList(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null

WORK_DIR="$PWD"

cat > "$WORK_DIR/feature_list.json" << 'EOF'
{"features":[
  {"id":"F001","category":"functional","description":"a","steps":["s"],"depends_on":["F002"],"batch":null,"passes":false},
  {"id":"F002","category":"functional","description":"b","steps":["s"],"depends_on":["F001"],"batch":null,"passes":false}
]}
EOF

cat > "$WORK_DIR/init.sh" << 'INIT_EOF'
#!/bin/bash
echo "init"
INIT_EOF
chmod +x "$WORK_DIR/init.sh"

echo "progress" > "$WORK_DIR/progress.txt"

echo '{"type":"system","subtype":"init","session_id":"test-invalid"}'
echo '{"type":"result","subtype":"success","is_error":false,"duration_ms":1000,"num_turns":1,"result":"done","stop_reason":"end_turn","total_cost_usd":0.01,"session_id":"test-invalid","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	initializer, tsk, _ := setupInitializerTest(t, mockScript)

	_, err := initializer.Run(tsk, nil)
	if err == nil {
		t.Fatal("Expected error for invalid feature_list.json (circular dependency)")
	}
	if tsk.Status != task.StatusFailed {
		t.Errorf("Task status: got %s, want %s", tsk.Status, task.StatusFailed)
	}
}

func TestInitializer_InvalidStateTransition(t *testing.T) {
	mockScript := `#!/bin/bash
cat > /dev/null
echo '{"type":"result","subtype":"success","is_error":false,"session_id":"x","usage":{"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}'
`

	initializer, tsk, _ := setupInitializerTest(t, mockScript)

	// 设为 completed，不能 transition 到 initializing
	tsk.Status = task.StatusCompleted

	_, err := initializer.Run(tsk, nil)
	if err == nil {
		t.Fatal("Expected error for invalid state transition")
	}
}

func TestValidateFeatureList_Valid(t *testing.T) {
	workDir := t.TempDir()
	fl := task.FeatureList{
		Features: []task.Feature{
			{ID: "F001", Category: "functional", Description: "test", Steps: []string{"s"}, DependsOn: []string{}},
			{ID: "F002", Category: "functional", Description: "test2", Steps: []string{"s"}, DependsOn: []string{"F001"}},
		},
	}
	data, _ := json.MarshalIndent(fl, "", "  ")
	os.WriteFile(filepath.Join(workDir, "feature_list.json"), data, 0644)

	init := &Initializer{}
	result, err := init.validateFeatureList(workDir)
	if err != nil {
		t.Fatalf("validateFeatureList failed: %v", err)
	}
	if len(result.Features) != 2 {
		t.Errorf("Features count: got %d, want 2", len(result.Features))
	}
}

func TestValidateFeatureList_NotFound(t *testing.T) {
	init := &Initializer{}
	_, err := init.validateFeatureList(t.TempDir())
	if err == nil {
		t.Fatal("Expected error for missing feature_list.json")
	}
}

func TestValidateFeatureList_Empty(t *testing.T) {
	workDir := t.TempDir()
	fl := task.FeatureList{Features: []task.Feature{}}
	data, _ := json.MarshalIndent(fl, "", "  ")
	os.WriteFile(filepath.Join(workDir, "feature_list.json"), data, 0644)

	init := &Initializer{}
	_, err := init.validateFeatureList(workDir)
	if err == nil {
		t.Fatal("Expected error for empty feature list")
	}
}

func TestValidateInitScript_Valid(t *testing.T) {
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "init.sh"), []byte("#!/bin/bash\necho init\n"), 0755)

	init := &Initializer{}
	if err := init.validateInitScript(workDir); err != nil {
		t.Fatalf("validateInitScript failed: %v", err)
	}
}

func TestValidateInitScript_NotFound(t *testing.T) {
	init := &Initializer{}
	if err := init.validateInitScript(t.TempDir()); err == nil {
		t.Fatal("Expected error for missing init.sh")
	}
}

func TestValidateInitScript_NotExecutable(t *testing.T) {
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "init.sh"), []byte("#!/bin/bash\n"), 0644)

	init := &Initializer{}
	if err := init.validateInitScript(workDir); err == nil {
		t.Fatal("Expected error for non-executable init.sh")
	}
}

func TestValidateProgressFile_Valid(t *testing.T) {
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "progress.txt"), []byte("Project: Test\nStatus: OK\n"), 0644)

	init := &Initializer{}
	if _, err := init.validateProgressFile(workDir); err != nil {
		t.Fatalf("validateProgressFile failed: %v", err)
	}
}

func TestValidateProgressFile_NotFound(t *testing.T) {
	init := &Initializer{}
	if _, err := init.validateProgressFile(t.TempDir()); err == nil {
		t.Fatal("Expected error for missing progress.txt")
	}
}

func TestValidateProgressFile_Empty(t *testing.T) {
	workDir := t.TempDir()
	os.WriteFile(filepath.Join(workDir, "progress.txt"), []byte(""), 0644)

	init := &Initializer{}
	if _, err := init.validateProgressFile(workDir); err == nil {
		t.Fatal("Expected error for empty progress.txt")
	}
}
