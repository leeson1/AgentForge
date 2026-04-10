package template

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// HookEnv Hook 执行时注入的环境变量
type HookEnv struct {
	TaskID       string
	SessionID    string
	WorkspaceDir string
	FeatureID    string
	BatchNum     int
	Extra        map[string]string // 模板自定义变量
}

// HookResult Hook 执行结果
type HookResult struct {
	Success  bool
	Output   string
	Duration time.Duration
	Error    error
}

// RunHook 执行 hook 脚本
func RunHook(script string, env HookEnv, timeout time.Duration) *HookResult {
	if script == "" {
		return &HookResult{Success: true}
	}

	start := time.Now()

	// 将脚本写入临时文件
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, fmt.Sprintf("agentforge-hook-%s-%d.sh", env.SessionID, time.Now().UnixNano()))
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return &HookResult{
			Error:    fmt.Errorf("write hook script: %w", err),
			Duration: time.Since(start),
		}
	}
	defer os.Remove(scriptPath)

	// 设置超时
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = env.WorkspaceDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// 杀死整个进程组
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	// 注入环境变量
	cmd.Env = append(os.Environ(), buildEnvVars(env)...)

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		return &HookResult{
			Output:   string(output),
			Duration: duration,
			Error:    fmt.Errorf("hook execution failed: %w\nOutput: %s", err, string(output)),
		}
	}

	return &HookResult{
		Success:  true,
		Output:   string(output),
		Duration: duration,
	}
}

// RunSessionStartHook 执行 session start hook
func RunSessionStartHook(tmpl *Template, env HookEnv) *HookResult {
	return RunHook(tmpl.HookStartScript, env, 30*time.Second)
}

// RunSessionEndHook 执行 session end hook
func RunSessionEndHook(tmpl *Template, env HookEnv) *HookResult {
	return RunHook(tmpl.HookEndScript, env, 30*time.Second)
}

// RunValidator 执行验证脚本
func RunValidator(tmpl *Template, env HookEnv) *HookResult {
	return RunHook(tmpl.ValidatorScript, env, 120*time.Second)
}

// buildEnvVars 构建环境变量列表
func buildEnvVars(env HookEnv) []string {
	vars := []string{
		"TASK_ID=" + env.TaskID,
		"SESSION_ID=" + env.SessionID,
		"WORKSPACE_DIR=" + env.WorkspaceDir,
		"FEATURE_ID=" + env.FeatureID,
		fmt.Sprintf("BATCH_NUM=%d", env.BatchNum),
	}
	for k, v := range env.Extra {
		vars = append(vars, k+"="+v)
	}
	return vars
}
