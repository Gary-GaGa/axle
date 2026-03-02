package skill

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	upgradePlanTimeout  = 3 * time.Minute
	upgradeApplyTimeout = 5 * time.Minute
	upgradeBuildTimeout = 2 * time.Minute
)

// UpgradePlan asks Copilot to analyze a feature request and return a plan.
func UpgradePlan(ctx context.Context, srcDir, model, request string) (string, error) {
	prompt := fmt.Sprintf(`你是 Axle 專案的開發者。使用者提出了以下功能需求：

%s

請分析 Axle 原始碼，產出一份簡潔的升級計畫，格式如下：
1. 列出需要修改/新增的檔案
2. 每個檔案的變更摘要（一行）
3. 預估風險（低/中/高）

注意：
- 不要輸出任何程式碼，只輸出計畫
- 使用繁體中文
- 保持精簡`, request)

	ctx, cancel := context.WithTimeout(ctx, upgradePlanTimeout)
	defer cancel()

	chunks, err := RunCopilot(ctx, srcDir, model, prompt)
	if err != nil {
		return "", fmt.Errorf("規劃失敗: %w", err)
	}
	return strings.Join(chunks, ""), nil
}

// UpgradeApply asks Copilot to implement the changes in the source directory.
func UpgradeApply(ctx context.Context, srcDir, model, request, plan string) (string, error) {
	prompt := fmt.Sprintf(`你是 Axle 專案的開發者。請根據以下需求與計畫，直接修改程式碼。

## 需求
%s

## 計畫
%s

## 規則
- 直接修改/建立檔案，不要只輸出程式碼片段
- 保持現有程式風格與架構
- 所有 Bot 回應使用繁體中文
- 修改完成後輸出一份變更摘要`, request, plan)

	ctx, cancel := context.WithTimeout(ctx, upgradeApplyTimeout)
	defer cancel()

	chunks, err := RunCopilot(ctx, srcDir, model, prompt)
	if err != nil {
		return "", fmt.Errorf("開發失敗: %w", err)
	}
	return strings.Join(chunks, ""), nil
}

// UpgradeBuild compiles the project and runs vet. Returns nil on success.
func UpgradeBuild(ctx context.Context, srcDir string) error {
	ctx, cancel := context.WithTimeout(ctx, upgradeBuildTimeout)
	defer cancel()

	// go vet
	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go vet 失敗:\n%s", string(out))
	}

	// go build
	binPath := filepath.Join(srcDir, "axle")
	cmd = exec.CommandContext(ctx, "go", "build", "-o", binPath, "./cmd/axle")
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build 失敗:\n%s", string(out))
	}

	return nil
}

// UpgradeTest runs tests on key packages and returns results.
func UpgradeTest(ctx context.Context, srcDir string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, upgradeBuildTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-cover",
		"./internal/app/...", "./internal/bot/skill/...")
	cmd.Dir = srcDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("測試失敗:\n%s", string(out))
	}
	return string(out), nil
}

// UpgradeCommit stages all changes and creates a git commit with the given message.
func UpgradeCommit(ctx context.Context, srcDir, version, summary string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// git add -A
	cmd := exec.CommandContext(ctx, "git", "add", "-A")
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add 失敗: %s", string(out))
	}

	// Check if there are changes to commit
	cmd = exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	cmd.Dir = srcDir
	if err := cmd.Run(); err == nil {
		return fmt.Errorf("沒有檔案變更需要提交")
	}

	// git commit
	msg := fmt.Sprintf("feat(self-upgrade): %s [v%s]\n\nCo-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>", summary, version)
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", msg)
	cmd.Dir = srcDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit 失敗: %s", string(out))
	}

	return nil
}

// UpgradeBackupBinary copies the current binary to axle.bak for rollback.
func UpgradeBackupBinary(srcDir string) error {
	binPath := filepath.Join(srcDir, "axle")
	bakPath := filepath.Join(srcDir, "axle.bak")

	data, err := os.ReadFile(binPath)
	if err != nil {
		return fmt.Errorf("讀取 binary 失敗: %w", err)
	}
	if err := os.WriteFile(bakPath, data, 0755); err != nil {
		return fmt.Errorf("備份 binary 失敗: %w", err)
	}
	return nil
}

// UpgradeRollbackBinary restores axle.bak to axle.
func UpgradeRollbackBinary(srcDir string) error {
	binPath := filepath.Join(srcDir, "axle")
	bakPath := filepath.Join(srcDir, "axle.bak")

	data, err := os.ReadFile(bakPath)
	if err != nil {
		return fmt.Errorf("讀取備份失敗: %w", err)
	}
	if err := os.WriteFile(binPath, data, 0755); err != nil {
		return fmt.Errorf("還原 binary 失敗: %w", err)
	}
	return nil
}

// BumpVersion reads version.go in the source directory, increments the patch number,
// and writes the file back. Returns the new version string.
func BumpVersion(srcDir string) (string, error) {
	versionFile := filepath.Join(srcDir, "internal", "app", "version.go")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("讀取 version.go 失敗: %w", err)
	}

	content := string(data)
	// Find version string like "0.10.0"
	idx := bytes.Index(data, []byte(`Version = "`))
	if idx == -1 {
		return "", fmt.Errorf("找不到 Version 常數")
	}

	start := idx + len(`Version = "`)
	end := strings.Index(content[start:], `"`)
	if end == -1 {
		return "", fmt.Errorf("Version 格式錯誤")
	}

	oldVer := content[start : start+end]

	// Parse major.minor.patch
	var major, minor, patch int
	if _, err := fmt.Sscanf(oldVer, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return "", fmt.Errorf("版本號解析失敗: %w", err)
	}

	patch++
	newVer := fmt.Sprintf("%d.%d.%d", major, minor, patch)

	newContent := strings.Replace(content, `Version = "`+oldVer+`"`, `Version = "`+newVer+`"`, 1)
	if err := os.WriteFile(versionFile, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("寫入 version.go 失敗: %w", err)
	}

	return newVer, nil
}
