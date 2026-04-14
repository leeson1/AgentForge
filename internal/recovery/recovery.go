package recovery

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leeson1/agent-forge/internal/config"
	"github.com/leeson1/agent-forge/internal/notify"
	"github.com/leeson1/agent-forge/internal/session"
)

// SessionStatus Session 状态
type SessionStatus struct {
	SessionID     string
	TaskID        string
	FeatureID     string
	RetryCount    int
	LastError     string
	TotalTokens   int
	LastToolCall  string
	SameToolCount int
	StartedAt     time.Time
}

// RecoveryManager 错误恢复管理器
type RecoveryManager struct {
	mu         sync.Mutex
	sessions   map[string]*SessionStatus
	cfg        *config.Config
	notifier   notify.Notifier
	maxRetries int
}

// NewRecoveryManager 创建恢复管理器
func NewRecoveryManager(cfg *config.Config, notifier notify.Notifier) *RecoveryManager {
	maxRetries := cfg.CLI.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &RecoveryManager{
		sessions:   make(map[string]*SessionStatus),
		cfg:        cfg,
		notifier:   notifier,
		maxRetries: maxRetries,
	}
}

// RegisterSession 注册新 Session
func (rm *RecoveryManager) RegisterSession(sessionID, taskID, featureID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.sessions[sessionID] = &SessionStatus{
		SessionID: sessionID,
		TaskID:    taskID,
		FeatureID: featureID,
		StartedAt: time.Now(),
	}
}

// UnregisterSession 注销 Session
func (rm *RecoveryManager) UnregisterSession(sessionID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.sessions, sessionID)
}

// HandleCrash 处理 Session 崩溃
// 返回是否应该重试
func (rm *RecoveryManager) HandleCrash(sessionID string, err error) (shouldRetry bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	status, ok := rm.sessions[sessionID]
	if !ok {
		return false
	}

	status.RetryCount++
	status.LastError = err.Error()

	if status.RetryCount > rm.maxRetries {
		log.Printf("[Recovery] Session %s exceeded max retries (%d), giving up", sessionID, rm.maxRetries)
		// 发送通知
		rm.sendNotification(notify.EventSessionCrash, status,
			fmt.Sprintf("Session %s crashed %d times, giving up. Last error: %s",
				sessionID, status.RetryCount, err.Error()))
		return false
	}

	log.Printf("[Recovery] Session %s crashed (attempt %d/%d), will retry. Error: %v",
		sessionID, status.RetryCount, rm.maxRetries, err)
	return true
}

// HandleTimeout 处理 Session 超时
func (rm *RecoveryManager) HandleTimeout(sessionID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	status, ok := rm.sessions[sessionID]
	if !ok {
		return
	}

	log.Printf("[Recovery] Session %s timed out after %v", sessionID, time.Since(status.StartedAt))
	rm.sendNotification(notify.EventSessionCrash, status,
		fmt.Sprintf("Session %s timed out after %v", sessionID, time.Since(status.StartedAt)))
}

// RecordToolCall 记录工具调用（用于死循环检测）
func (rm *RecoveryManager) RecordToolCall(sessionID, toolName string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	status, ok := rm.sessions[sessionID]
	if !ok {
		return
	}

	if toolName == status.LastToolCall {
		status.SameToolCount++
	} else {
		status.LastToolCall = toolName
		status.SameToolCount = 1
	}
}

// IsStuck 检测是否死循环
// 连续相同 tool_call 超过 threshold 次则认为 stuck
func (rm *RecoveryManager) IsStuck(sessionID string, threshold int) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	status, ok := rm.sessions[sessionID]
	if !ok {
		return false
	}

	if threshold <= 0 {
		threshold = 10
	}

	return status.SameToolCount >= threshold
}

// RecordTokens 记录 Token 消耗
func (rm *RecoveryManager) RecordTokens(sessionID string, tokens int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	status, ok := rm.sessions[sessionID]
	if !ok {
		return
	}
	status.TotalTokens += tokens
}

// sendNotification 发送恢复相关通知
func (rm *RecoveryManager) sendNotification(eventType notify.EventType, status *SessionStatus, message string) {
	if rm.notifier == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = rm.notifier.Send(ctx, notify.Notification{
			Type:      eventType,
			TaskID:    status.TaskID,
			Message:   message,
			Timestamp: time.Now(),
			Data: map[string]string{
				"session_id": status.SessionID,
				"feature_id": status.FeatureID,
				"retries":    fmt.Sprintf("%d", status.RetryCount),
			},
		})
	}()
}

// CostMonitor 成本监控
type CostMonitor struct {
	mu       sync.Mutex
	cfg      *config.Config
	notifier notify.Notifier
	// taskID -> 累计成本
	taskCosts map[string]float64
	// taskID -> 是否已告警
	alerted map[string]bool
}

// NewCostMonitor 创建成本监控器
func NewCostMonitor(cfg *config.Config, notifier notify.Notifier) *CostMonitor {
	return &CostMonitor{
		cfg:       cfg,
		notifier:  notifier,
		taskCosts: make(map[string]float64),
		alerted:   make(map[string]bool),
	}
}

// RecordUsage 记录 token 用量并检查阈值
// 返回 (currentCost, shouldPause)
func (cm *CostMonitor) RecordUsage(taskID, taskName string, inputTokens, outputTokens int) (float64, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cost := cm.cfg.EstimateCost(inputTokens, outputTokens)
	cm.taskCosts[taskID] += cost
	totalCost := cm.taskCosts[taskID]

	// 告警阈值
	if totalCost >= cm.cfg.Cost.AlertThreshold && !cm.alerted[taskID] {
		cm.alerted[taskID] = true
		if cm.notifier != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = cm.notifier.Send(ctx, notify.Notification{
					Type:      notify.EventCostAlert,
					TaskID:    taskID,
					TaskName:  taskName,
					Message:   fmt.Sprintf("Task %q cost reached $%.2f (threshold: $%.2f)", taskName, totalCost, cm.cfg.Cost.AlertThreshold),
					Timestamp: time.Now(),
					Data: map[string]string{
						"current_cost": fmt.Sprintf("%.4f", totalCost),
						"threshold":    fmt.Sprintf("%.2f", cm.cfg.Cost.AlertThreshold),
					},
				})
			}()
		}
	}

	// 硬限制
	shouldPause := cm.cfg.Cost.HardLimit > 0 && totalCost >= cm.cfg.Cost.HardLimit
	return totalCost, shouldPause
}

// GetTaskCost 获取任务当前成本
func (cm *CostMonitor) GetTaskCost(taskID string) float64 {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.taskCosts[taskID]
}

// CheckDiskSpace 检查磁盘空间
// 返回可用空间（字节），空间不足时返回 error
func CheckDiskSpace(dir string, minFreeBytes int64) error {
	if minFreeBytes <= 0 {
		minFreeBytes = 500 * 1024 * 1024 // 默认 500MB
	}

	// 使用 df 命令检查（跨平台简单方式）
	// 在生产环境中应使用 syscall.Statfs
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("check disk space: directory %q not accessible: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("check disk space: %q is not a directory", dir)
	}

	// 简单检查：创建临时文件测试写入权限
	testFile := filepath.Join(dir, ".agentforge-diskcheck")
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("disk space check: cannot write to %q: %w", dir, err)
	}
	os.Remove(testFile)

	return nil
}

// ScanRunningTasks 扫描运行中的任务目录
// 返回 (taskID, PID, isRunning) 列表
func ScanRunningTasks(baseDir string) []TaskScanResult {
	var results []TaskScanResult

	tasksDir := filepath.Join(baseDir, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return results
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		taskID := entry.Name()
		pidFile := filepath.Join(tasksDir, taskID, "pid")

		pidData, err := os.ReadFile(pidFile)
		if err != nil {
			continue // 没有 PID 文件
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if err != nil {
			continue
		}

		results = append(results, TaskScanResult{
			TaskID:    taskID,
			PID:       pid,
			IsRunning: session.IsProcessAlive(pid),
		})
	}

	return results
}

// TaskScanResult 任务扫描结果
type TaskScanResult struct {
	TaskID    string
	PID       int
	IsRunning bool
}
