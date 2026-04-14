package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/leeson1/agent-forge/internal/config"
	"github.com/leeson1/agent-forge/internal/notify"
)

func testConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.CLI.MaxRetries = 3
	cfg.Cost.AlertThreshold = 5.0
	cfg.Cost.HardLimit = 20.0
	return cfg
}

func TestRecoveryManager_HandleCrash(t *testing.T) {
	rm := NewRecoveryManager(testConfig(), notify.NoopNotifier{})
	rm.RegisterSession("S1", "T1", "F001")

	// 第 1 次崩溃 -> 应重试
	if !rm.HandleCrash("S1", fmt.Errorf("crash 1")) {
		t.Error("First crash should allow retry")
	}

	// 第 2 次
	if !rm.HandleCrash("S1", fmt.Errorf("crash 2")) {
		t.Error("Second crash should allow retry")
	}

	// 第 3 次
	if !rm.HandleCrash("S1", fmt.Errorf("crash 3")) {
		t.Error("Third crash should allow retry")
	}

	// 第 4 次 -> 超过限制
	if rm.HandleCrash("S1", fmt.Errorf("crash 4")) {
		t.Error("Fourth crash should NOT allow retry (exceeded max)")
	}
}

func TestRecoveryManager_HandleCrash_UnknownSession(t *testing.T) {
	rm := NewRecoveryManager(testConfig(), notify.NoopNotifier{})
	if rm.HandleCrash("unknown", fmt.Errorf("err")) {
		t.Error("Unknown session should not retry")
	}
}

func TestRecoveryManager_UnregisterSession(t *testing.T) {
	rm := NewRecoveryManager(testConfig(), notify.NoopNotifier{})
	rm.RegisterSession("S1", "T1", "F001")
	rm.UnregisterSession("S1")

	if rm.HandleCrash("S1", fmt.Errorf("err")) {
		t.Error("Unregistered session should not retry")
	}
}

func TestRecoveryManager_IsStuck(t *testing.T) {
	rm := NewRecoveryManager(testConfig(), notify.NoopNotifier{})
	rm.RegisterSession("S1", "T1", "F001")

	// 记录不同工具调用
	rm.RecordToolCall("S1", "read_file")
	rm.RecordToolCall("S1", "write_file")
	if rm.IsStuck("S1", 5) {
		t.Error("Should not be stuck with different tool calls")
	}

	// 记录连续相同工具调用
	for i := 0; i < 5; i++ {
		rm.RecordToolCall("S1", "read_file")
	}
	if !rm.IsStuck("S1", 5) {
		t.Error("Should be stuck after 5 same tool calls")
	}
}

func TestRecoveryManager_IsStuck_DefaultThreshold(t *testing.T) {
	rm := NewRecoveryManager(testConfig(), notify.NoopNotifier{})
	rm.RegisterSession("S1", "T1", "F001")

	for i := 0; i < 10; i++ {
		rm.RecordToolCall("S1", "same_tool")
	}
	if !rm.IsStuck("S1", 0) {
		t.Error("Should be stuck with default threshold (10)")
	}
}

func TestCostMonitor_RecordUsage(t *testing.T) {
	cfg := testConfig()
	cm := NewCostMonitor(cfg, notify.NoopNotifier{})

	// 少量 token，不应暂停
	cost, pause := cm.RecordUsage("T1", "Test", 1000, 500)
	if pause {
		t.Error("Should not pause for small usage")
	}
	if cost <= 0 {
		t.Error("Cost should be positive")
	}
}

func TestCostMonitor_AlertThreshold(t *testing.T) {
	cfg := testConfig()
	cfg.Cost.AlertThreshold = 0.001 // 极低阈值
	cm := NewCostMonitor(cfg, notify.NoopNotifier{})

	_, _ = cm.RecordUsage("T1", "Test", 100000, 100000)
	// 检查已触发告警（内部状态）
	// 这里主要测试不 panic
}

func TestCostMonitor_HardLimit(t *testing.T) {
	cfg := testConfig()
	cfg.Cost.HardLimit = 0.001 // 极低限制
	cm := NewCostMonitor(cfg, notify.NoopNotifier{})

	_, pause := cm.RecordUsage("T1", "Test", 1000000, 1000000)
	if !pause {
		t.Error("Should pause when exceeding hard limit")
	}
}

func TestCostMonitor_GetTaskCost(t *testing.T) {
	cfg := testConfig()
	cm := NewCostMonitor(cfg, notify.NoopNotifier{})

	cm.RecordUsage("T1", "Test", 1000, 500)
	cost := cm.GetTaskCost("T1")
	if cost <= 0 {
		t.Error("Should have recorded cost")
	}

	cost2 := cm.GetTaskCost("T2")
	if cost2 != 0 {
		t.Error("Unknown task should have 0 cost")
	}
}

func TestCheckDiskSpace(t *testing.T) {
	dir := t.TempDir()
	err := CheckDiskSpace(dir, 1024) // 1KB
	if err != nil {
		t.Errorf("CheckDiskSpace should pass for temp dir: %v", err)
	}
}

func TestCheckDiskSpace_NonExistent(t *testing.T) {
	err := CheckDiskSpace("/nonexistent/dir", 1024)
	if err == nil {
		t.Error("Should error for nonexistent directory")
	}
}

func TestScanRunningTasks(t *testing.T) {
	dir := t.TempDir()

	// 创建模拟的任务目录
	taskDir := filepath.Join(dir, "tasks", "task-1")
	os.MkdirAll(taskDir, 0755)

	// 写入当前进程 PID（存活）
	os.WriteFile(filepath.Join(taskDir, "pid"), []byte(fmt.Sprintf("%d", os.Getpid())), 0644)

	// 创建另一个带无效 PID 的任务
	taskDir2 := filepath.Join(dir, "tasks", "task-2")
	os.MkdirAll(taskDir2, 0755)
	os.WriteFile(filepath.Join(taskDir2, "pid"), []byte("999999999"), 0644)

	results := ScanRunningTasks(dir)
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// 找到 task-1
	var found bool
	var deadFound bool
	for _, r := range results {
		if r.TaskID == "task-1" {
			found = true
			if r.PID != os.Getpid() {
				t.Errorf("Expected PID %d, got %d", os.Getpid(), r.PID)
			}
			if !r.IsRunning {
				t.Error("Expected current process to be reported as running")
			}
		}
		if r.TaskID == "task-2" {
			deadFound = true
			if r.IsRunning {
				t.Error("Expected invalid PID to be reported as not running")
			}
		}
	}
	if !found {
		t.Error("task-1 not found in results")
	}
	if !deadFound {
		t.Error("task-2 not found in results")
	}
}

func TestScanRunningTasks_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results := ScanRunningTasks(dir)
	if len(results) != 0 {
		t.Errorf("Expected 0 results from empty dir, got %d", len(results))
	}
}
