package skill

import (
	"bufio"
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
		"--allow-all-tools", // required for non-interactive mode
		"--silent",          // output only the agent response
		"--no-color",        // no ANSI codes in output
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

// RunCopilotStream executes the Copilot CLI and calls onUpdate with accumulated
// output as content streams in line-by-line. Returns the final complete output.
func RunCopilotStream(ctx context.Context, workspace, model, prompt string, onUpdate func(accumulated string)) (string, error) {
	if len(prompt) > MaxPromptChars {
		prompt = prompt[:MaxPromptChars]
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("workspace 解析失敗: %w", err)
	}
	if model == "" {
		model = DefaultModel
	}

	// Use copilot timeout, but respect shorter parent deadline
	timeout := copilotTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "copilot",
		"--model", model,
		"-p", prompt,
		"--allow-all-tools",
		"--silent",
		"--no-color",
		"--add-dir", absWorkspace,
	)
	cmd.Dir = absWorkspace

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe 失敗: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("Copilot 啟動失敗: %w", err)
	}
	// Ensure process is cleaned up even if scanner panics or goroutine is interrupted
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait() // reap zombie
		}
	}()

	var accumulated strings.Builder
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line

	for scanner.Scan() {
		line := scanner.Text()
		if accumulated.Len() > 0 {
			accumulated.WriteString("\n")
		}
		accumulated.WriteString(line)
		if onUpdate != nil {
			onUpdate(accumulated.String())
		}
	}

	if err := cmd.Wait(); err != nil {
		switch ctx.Err() {
		case context.Canceled:
			return "", context.Canceled
		case context.DeadlineExceeded:
			return "", fmt.Errorf("⏱ Copilot 執行超時（限制 %s）", copilotTimeout)
		}
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("Copilot 執行失敗: %s", errMsg)
	}

	result := strings.TrimSpace(accumulated.String())
	if result == "" {
		result = "（Copilot 無回應）"
	}
	return result, nil
}
