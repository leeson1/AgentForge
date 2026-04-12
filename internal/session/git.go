package session

import (
	"fmt"
	"os/exec"
	"strings"
)

// HeadCommit 返回当前仓库 HEAD commit。
func HeadCommit(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// HasCommitAdvanced 判断当前 HEAD 是否已经超出给定 base commit。
func HasCommitAdvanced(repoDir, baseCommit string) (bool, string, error) {
	headCommit, err := HeadCommit(repoDir)
	if err != nil {
		return false, "", err
	}
	return headCommit != strings.TrimSpace(baseCommit), headCommit, nil
}

// IsWorktreeClean 判断工作区是否干净。
func IsWorktreeClean(repoDir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git status failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)) == "", nil
}

// ChangedFilesSince 返回 base..HEAD 之间修改的文件列表。
func ChangedFilesSince(repoDir, baseCommit string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseCommit+"..HEAD")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	if strings.TrimSpace(string(output)) == "" {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
