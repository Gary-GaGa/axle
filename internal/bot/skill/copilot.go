package skill

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const copilotTimeout = 5 * time.Minute // safety limit for Copilot CLI

// RunCopilot executes the Copilot CLI with a prompt and returns the response
// as one or more Telegram-safe message chunks.
//
// Context limit prevention:
//   - Prompts exceeding MaxPromptChars are silently truncated before sending
//   - Responses are split into ≤ MaxResponseChars chunks by SplitMessage
//   - --add-dir restricts file access to the workspace sandbox
//   - A 5-minute timeout prevents the process from hanging forever
func RunCopilot(ctx context.Context, workspace, model, prompt string) ([]string, error) {
	if len(prompt) > MaxPromptChars {
		prompt = prompt[:MaxPromptChars]
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return nil, fmt.Errorf("workspace 解析失敗: %w", err)
	}
	if model == "" {
		model = DefaultModel
	}

	ctx, cancel := context.WithTimeout(ctx, copilotTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "copilot",
		"--model", model,
		"-p", prompt,
		"--allow-all-tools",  // required for non-interactive mode
		"--allow-all-paths",  // needed when workspace is not home dir
		"--silent",           // output only the agent response
		"--no-color",         // no ANSI codes in output
		"--add-dir", absWorkspace,
	)
	cmd.Dir = absWorkspace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		switch ctx.Err() {
		case context.Canceled:
			return nil, context.Canceled
		case context.DeadlineExceeded:
			return nil, fmt.Errorf("⏱ Copilot 執行超時（限制 %s）", copilotTimeout)
		}
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("Copilot 執行失敗: %s", errMsg)
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		result = "（Copilot 無回應）"
	}
	return SplitMessage(result), nil
}
