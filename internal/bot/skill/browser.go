package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	infrafiles "github.com/garyellow/axle/internal/infrastructure/files"
	"github.com/garyellow/axle/internal/usecase/browserdsl"
)

const (
	defaultBrowserWait   = 1500 * time.Millisecond
	maxBrowserTextChars  = 12000
	defaultExtractTarget = browserdsl.DefaultExtractTarget
	maxBrowserWait       = browserdsl.MaxWait
	unsafeBrowserEnvKey  = "AXLE_ALLOW_UNSAFE_BROWSER"
)

// BrowserStep represents one browser automation step.
type BrowserStep = browserdsl.Step

// BrowserPlan is a parsed browser script.
type BrowserPlan = browserdsl.Plan

// BrowserRunResult contains the outputs from a browser automation run.
type BrowserRunResult struct {
	URL           string        `json:"url,omitempty"`
	ExtractedText string        `json:"extracted_text,omitempty"`
	Screenshots   []string      `json:"screenshots,omitempty"`
	ArtifactDir   string        `json:"artifact_dir,omitempty"`
	ExecutedSteps []BrowserStep `json:"executed_steps,omitempty"`
}

// Summary returns a concise human-readable summary.
func (r BrowserRunResult) Summary() string {
	var parts []string
	if r.URL != "" {
		parts = append(parts, "URL: "+r.URL)
	}
	if r.ExtractedText != "" {
		parts = append(parts, "Extracted: "+truncateBrowserText(strings.ReplaceAll(r.ExtractedText, "\n", " "), 400))
	}
	if len(r.Screenshots) > 0 {
		parts = append(parts, "Screenshots: "+strings.Join(r.Screenshots, ", "))
	}
	if r.ArtifactDir != "" {
		parts = append(parts, "Artifacts: "+r.ArtifactDir)
	}
	if len(parts) == 0 {
		return "Browser run completed."
	}
	return strings.Join(parts, "\n")
}

// ParseBrowserScript parses the small Axle browser DSL.
func ParseBrowserScript(input string) (BrowserPlan, error) {
	return browserdsl.ParseScript(input)
}

// RunBrowserScript parses and executes a browser script.
func RunBrowserScript(ctx context.Context, workspace, script string) (BrowserRunResult, error) {
	plan, err := ParseBrowserScript(script)
	if err != nil {
		return BrowserRunResult{}, err
	}
	return RunBrowserPlan(ctx, workspace, plan)
}

// RunBrowserPlan executes a parsed browser plan via Safari on macOS.
func RunBrowserPlan(ctx context.Context, workspace string, plan BrowserPlan) (BrowserRunResult, error) {
	if err := validateBrowserPlan(plan); err != nil {
		return BrowserRunResult{}, err
	}
	if os.Getenv(unsafeBrowserEnvKey) != "1" {
		return BrowserRunResult{}, fmt.Errorf("browser automation 預設停用；如需自行承擔風險啟用，請設定 %s=1", unsafeBrowserEnvKey)
	}
	if runtime.GOOS != "darwin" {
		return BrowserRunResult{}, fmt.Errorf("browser automation 目前僅支援 macOS Safari")
	}

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	artifactRel := filepath.Join(".axle", "browser", runID)
	root, _, err := infrafiles.OpenWorkspaceRoot(workspace)
	if err != nil {
		return BrowserRunResult{}, err
	}
	defer root.Close()

	artifactDir, err := infrafiles.EnsureRootedDirectory(root, artifactRel, artifactRel, 0755)
	if err != nil {
		return BrowserRunResult{}, fmt.Errorf("建立 browser artifact 目錄失敗: %w", err)
	}

	result := BrowserRunResult{
		ArtifactDir:   artifactRel,
		ExecutedSteps: make([]BrowserStep, 0, len(plan.Steps)),
	}

	for i, step := range plan.Steps {
		select {
		case <-ctx.Done():
			return BrowserRunResult{}, context.Canceled
		default:
		}

		switch step.Action {
		case "open":
			safeURL, err := preflightBrowserNavigation(ctx, step.Arg)
			if err != nil {
				return BrowserRunResult{}, err
			}
			if err := safariOpenURL(ctx, safeURL); err != nil {
				return BrowserRunResult{}, err
			}
			currentURL, err := validateSafariCurrentPage(ctx)
			if err != nil {
				return BrowserRunResult{}, err
			}
			result.URL = currentURL
			if err := sleepContext(ctx, defaultBrowserWait); err != nil {
				return BrowserRunResult{}, err
			}
			currentURL, err = validateSafariCurrentPage(ctx)
			if err != nil {
				return BrowserRunResult{}, err
			}
			result.URL = currentURL

		case "wait":
			d, _ := time.ParseDuration(step.Arg)
			if err := validateBrowserWaitDuration(d); err != nil {
				return BrowserRunResult{}, err
			}
			if err := sleepContext(ctx, d); err != nil {
				return BrowserRunResult{}, err
			}

		case "extract":
			currentURL, err := validateSafariCurrentPage(ctx)
			if err != nil {
				return BrowserRunResult{}, err
			}
			result.URL = currentURL
			text, err := safariExtract(ctx, step.Arg)
			if err != nil {
				return BrowserRunResult{}, err
			}
			result.ExtractedText = truncateBrowserText(text, maxBrowserTextChars)

		case "screenshot":
			currentURL, err := validateSafariCurrentPage(ctx)
			if err != nil {
				return BrowserRunResult{}, err
			}
			result.URL = currentURL
			relPath := step.Arg
			if relPath == "" {
				relPath = filepath.Join(artifactRel, fmt.Sprintf("step-%d.png", i+1))
			}
			relPath, err = normalizeBrowserScreenshotPath(artifactRel, relPath)
			if err != nil {
				return BrowserRunResult{}, err
			}

			tmp, err := os.CreateTemp("", "axle-browser-*.png")
			if err != nil {
				return BrowserRunResult{}, fmt.Errorf("建立暫存 screenshot 檔案失敗: %w", err)
			}
			tmpPath := tmp.Name()
			if err := tmp.Close(); err != nil {
				os.Remove(tmpPath)
				return BrowserRunResult{}, fmt.Errorf("建立暫存 screenshot 檔案失敗: %w", err)
			}
			if err := safariScreenshot(ctx, tmpPath); err != nil {
				os.Remove(tmpPath)
				return BrowserRunResult{}, err
			}
			data, err := os.ReadFile(tmpPath)
			os.Remove(tmpPath)
			if err != nil {
				return BrowserRunResult{}, fmt.Errorf("讀取暫存 screenshot 失敗: %w", err)
			}
			if err := infrafiles.WriteRootedFile(root, relPath, relPath, data, 0644); err != nil {
				return BrowserRunResult{}, err
			}
			result.Screenshots = append(result.Screenshots, relPath)
		}

		result.ExecutedSteps = append(result.ExecutedSteps, step)
	}

	if err := writeBrowserResult(root, artifactDir, result); err != nil {
		return BrowserRunResult{}, err
	}
	return result, nil
}

func validateBrowserPlan(plan BrowserPlan) error {
	return browserdsl.ValidatePlan(plan)
}

func validateBrowserWaitDuration(d time.Duration) error {
	return browserdsl.ValidateWaitDuration(d)
}

func normalizeBrowserScreenshotPath(artifactRel, relPath string) (string, error) {
	return browserdsl.NormalizeScreenshotPath(artifactRel, relPath)
}

func safariOpenURL(ctx context.Context, rawURL string) error {
	_, err := runAppleScript(ctx,
		`tell application "Safari" to activate`,
		`tell application "Safari" to open location `+appleQuote(rawURL),
	)
	if err != nil {
		return fmt.Errorf("開啟 Safari 頁面失敗: %w", err)
	}
	return nil
}

func safariExtract(ctx context.Context, selector string) (string, error) {
	if selector == "" {
		selector = defaultExtractTarget
	}
	js := `(() => {
  const selector = ` + strconv.Quote(selector) + `;
  const target = selector === "body" ? document.body : document.querySelector(selector);
  return target ? target.innerText : "";
})()`

	out, err := runAppleScript(ctx,
		`tell application "Safari" to activate`,
		`tell application "Safari"`,
		`  if (count of windows) = 0 then error "Safari 沒有開啟頁面"`,
		`  return do JavaScript `+appleQuote(js)+` in current tab of front window`,
		`end tell`,
	)
	if err != nil {
		return "", fmt.Errorf("擷取瀏覽器內容失敗: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func safariScreenshot(ctx context.Context, absPath string) error {
	bounds, err := runAppleScript(ctx,
		`tell application "Safari" to activate`,
		`tell application "Safari"`,
		`  if (count of windows) = 0 then error "Safari 沒有開啟頁面"`,
		`  set {x1, y1, x2, y2} to bounds of front window`,
		`  return (x1 as text) & "," & (y1 as text) & "," & ((x2 - x1) as text) & "," & ((y2 - y1) as text)`,
		`end tell`,
	)
	if err != nil {
		return fmt.Errorf("取得 Safari 視窗位置失敗: %w", err)
	}

	args := strings.Split(strings.TrimSpace(bounds), ",")
	if len(args) != 4 {
		return fmt.Errorf("無法解析 Safari 視窗範圍: %q", bounds)
	}

	cmd := exec.CommandContext(ctx, "screencapture", "-x", "-R", strings.Join(args, ","), absPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("擷取 screenshot 失敗: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func writeBrowserResult(root *os.Root, artifactDir string, result BrowserRunResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 browser 結果失敗: %w", err)
	}
	resultPath := filepath.Join(artifactDir, "result.json")
	if err := infrafiles.WriteRootedFile(root, resultPath, resultPath, data, 0644); err != nil {
		return fmt.Errorf("寫入 browser 結果失敗: %w", err)
	}
	return nil
}

func validateBrowserTarget(parsed *url.URL) error {
	return browserdsl.ValidateTarget(parsed)
}

func validateBrowserTargetResolved(_ context.Context, parsed *url.URL) error {
	return validateBrowserTarget(parsed)
}

func preflightBrowserNavigation(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("browser script: 無效 URL %q", rawURL)
	}
	if err := validateBrowserTargetResolved(ctx, parsed); err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("browser script: redirect 次數過多")
			}
			return validateBrowserTargetResolved(ctx, req.URL)
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("browser script: 建立預檢請求失敗: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("browser script: URL 預檢失敗: %w", err)
	}
	_ = resp.Body.Close()
	if resp.Request != nil && resp.Request.URL != nil {
		if err := validateBrowserTargetResolved(ctx, resp.Request.URL); err != nil {
			return "", err
		}
		return resp.Request.URL.String(), nil
	}
	return parsed.String(), nil
}

func validateSafariCurrentPage(ctx context.Context) (string, error) {
	currentURL, err := safariCurrentURL(ctx)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(strings.TrimSpace(currentURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("無法驗證 Safari 目前頁面: %q", currentURL)
	}
	if err := validateBrowserTargetResolved(ctx, parsed); err != nil {
		return "", err
	}
	return parsed.String(), nil
}

func safariCurrentURL(ctx context.Context) (string, error) {
	out, err := runAppleScript(ctx,
		`tell application "Safari" to activate`,
		`tell application "Safari"`,
		`  if (count of windows) = 0 then error "Safari 沒有開啟頁面"`,
		`  return URL of current tab of front window`,
		`end tell`,
	)
	if err != nil {
		return "", fmt.Errorf("取得 Safari 目前 URL 失敗: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func runAppleScript(ctx context.Context, lines ...string) (string, error) {
	args := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		args = append(args, "-e", line)
	}
	cmd := exec.CommandContext(ctx, "osascript", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func appleQuote(s string) string {
	return strconv.Quote(s)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Canceled
	case <-timer.C:
		return nil
	}
}

func truncateBrowserText(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n\n⚠️ Browser 擷取內容過長，已截斷。"
}
