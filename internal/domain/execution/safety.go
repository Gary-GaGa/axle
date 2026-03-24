package execution

import (
	"regexp"
	"strings"
)

// DangerLevel classifies how dangerous a shell command is.
type DangerLevel int

const (
	DangerNone DangerLevel = iota
	DangerWarning
	DangerBlocked
)

// SafetyReport describes the result of a command safety evaluation.
type SafetyReport struct {
	Level   DangerLevel
	Reasons []string
}

var dangerPatterns = []struct {
	Pattern *regexp.Regexp
	Level   DangerLevel
	Reason  string
}{
	{regexp.MustCompile(`(?i)\brm\s+(-[a-zA-Z]*f[a-zA-Z]*)?\s*/\s*$`), DangerBlocked, "刪除根目錄"},
	{regexp.MustCompile(`(?i)\bmkfs\b`), DangerBlocked, "格式化磁碟"},
	{regexp.MustCompile(`(?i)\bdd\b.*\bof=/dev/`), DangerBlocked, "覆寫裝置"},
	{regexp.MustCompile(`(?i)>\s*/dev/[sh]d[a-z]`), DangerBlocked, "覆寫裝置"},
	{regexp.MustCompile(`(?i):\(\)\{ :\|:& \};:`), DangerBlocked, "Fork bomb"},
	{regexp.MustCompile(`(?i)\brm\b`), DangerWarning, "刪除檔案/目錄"},
	{regexp.MustCompile(`(?i)\brmdir\b`), DangerWarning, "刪除目錄"},
	{regexp.MustCompile(`(?i)\bgit\s+(reset|clean|push\s+.*--force|push\s+.*-f)\b`), DangerWarning, "Git 破壞性操作"},
	{regexp.MustCompile(`(?i)\bchmod\b`), DangerWarning, "變更權限"},
	{regexp.MustCompile(`(?i)\bchown\b`), DangerWarning, "變更擁有者"},
	{regexp.MustCompile(`(?i)\bkill\b`), DangerWarning, "終止程序"},
	{regexp.MustCompile(`(?i)\bsudo\b`), DangerWarning, "提權操作"},
	{regexp.MustCompile(`(?i)\bmv\b`), DangerWarning, "移動/重命名"},
	{regexp.MustCompile(`(?i)>\s*[^|]`), DangerWarning, "檔案覆寫 (>)"},
	{regexp.MustCompile(`(?i)\bcurl\b.*\|\s*(bash|sh)\b`), DangerWarning, "管道執行遠端腳本"},
	{regexp.MustCompile(`(?i)\bwget\b.*\|\s*(bash|sh)\b`), DangerWarning, "管道執行遠端腳本"},
	{regexp.MustCompile(`(?i)\bdrop\s+(table|database)\b`), DangerWarning, "刪除資料表/資料庫"},
	{regexp.MustCompile(`(?i)\btruncate\b`), DangerWarning, "清空資料表"},
	{regexp.MustCompile(`(?i)\bdelete\s+from\b`), DangerWarning, "刪除資料"},
}

// ClassifyCommand analyses a shell command string and returns the highest danger level found.
func ClassifyCommand(cmd string) SafetyReport {
	cmd = strings.TrimSpace(cmd)
	maxLevel := DangerNone
	var reasons []string

	for _, dp := range dangerPatterns {
		if dp.Pattern.MatchString(cmd) {
			if dp.Level > maxLevel {
				maxLevel = dp.Level
			}
			reasons = append(reasons, dp.Reason)
		}
	}

	return SafetyReport{Level: maxLevel, Reasons: reasons}
}
