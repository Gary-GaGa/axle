package skill

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const ghTimeout = 30 * time.Second

// GHTimeout returns the timeout for gh CLI operations.
func GHTimeout() time.Duration { return ghTimeout }

// ghRun executes a `gh` CLI command in the given workspace.
func ghRun(ctx context.Context, workspace string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, ghTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("`gh %s` 失敗:\n%s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "(無輸出)", nil
	}
	return result, nil
}

// GHCheckInstalled checks if gh CLI is available.
func GHCheckInstalled() error {
	_, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("`gh` CLI 未安裝，請先執行 `brew install gh` 並完成認證")
	}
	return nil
}

// GHPRList lists open pull requests.
func GHPRList(ctx context.Context, workspace string) (string, error) {
	return ghRun(ctx, workspace, "pr", "list", "--limit", "10")
}

// GHPRView shows the current PR details.
func GHPRView(ctx context.Context, workspace string) (string, error) {
	return ghRun(ctx, workspace, "pr", "view")
}

// GHPRCreate creates a new pull request.
func GHPRCreate(ctx context.Context, workspace, title, body string) (string, error) {
	return ghRun(ctx, workspace, "pr", "create", "--title", title, "--body", body)
}

// GHIssueList lists open issues.
func GHIssueList(ctx context.Context, workspace string) (string, error) {
	return ghRun(ctx, workspace, "issue", "list", "--limit", "10")
}

// GHIssueView shows a specific issue.
func GHIssueView(ctx context.Context, workspace, number string) (string, error) {
	return ghRun(ctx, workspace, "issue", "view", number)
}

// GHCIStatus shows recent workflow runs.
func GHCIStatus(ctx context.Context, workspace string) (string, error) {
	return ghRun(ctx, workspace, "run", "list", "--limit", "5")
}

// GHRepoView shows repository info.
func GHRepoView(ctx context.Context, workspace string) (string, error) {
	return ghRun(ctx, workspace, "repo", "view")
}
