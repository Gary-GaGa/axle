package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBrowserWait   = 1500 * time.Millisecond
	maxBrowserTextChars  = 12000
	defaultExtractTarget = "body"
)

// BrowserStep represents one browser automation step.
type BrowserStep struct {
	Action string `json:"action"`
	Arg    string `json:"arg,omitempty"`
}

// BrowserPlan is a parsed browser script.
type BrowserPlan struct {
	Steps []BrowserStep `json:"steps"`
}

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
	lines := strings.Split(input, "\n")
	plan := BrowserPlan{Steps: make([]BrowserStep, 0, len(lines))}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		action := strings.ToLower(parts[0])
		arg := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))

		switch action {
		case "open":
			if arg == "" {
				return BrowserPlan{}, fmt.Errorf("browser script: open 缺少 URL")
			}
			parsed, err := url.Parse(strings.TrimSpace(arg))
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return BrowserPlan{}, fmt.Errorf("browser script: 無效 URL %q", arg)
			}
			if err := validateBrowserTarget(parsed); err != nil {
				return BrowserPlan{}, err
			}
		case "wait":
			if arg == "" {
				return BrowserPlan{}, fmt.Errorf("browser script: wait 缺少 duration")
			}
			if _, err := time.ParseDuration(strings.TrimSpace(arg)); err != nil {
				return BrowserPlan{}, fmt.Errorf("browser script: wait duration 無效: %w", err)
			}
		case "extract":
			if strings.TrimSpace(arg) == "" {
				arg = defaultExtractTarget
			}
		case "screenshot":
			// empty path is allowed; a default artifact path will be generated
		default:
			return BrowserPlan{}, fmt.Errorf("browser script: 不支援的指令 %q", action)
		}

		plan.Steps = append(plan.Steps, BrowserStep{Action: action, Arg: strings.TrimSpace(arg)})
	}

	if len(plan.Steps) == 0 {
		return BrowserPlan{}, fmt.Errorf("browser script 不能為空")
	}
	return plan, nil
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
	if runtime.GOOS != "darwin" {
		return BrowserRunResult{}, fmt.Errorf("browser automation 目前僅支援 macOS Safari")
	}

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	artifactRel := filepath.Join(".axle", "browser", runID)
	artifactDir, err := resolveAndValidate(workspace, artifactRel)
	if err != nil {
		return BrowserRunResult{}, err
	}
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
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
			absPath, err := resolveAndValidate(workspace, relPath)
			if err != nil {
				return BrowserRunResult{}, err
			}
			if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
				return BrowserRunResult{}, fmt.Errorf("建立 screenshot 目錄失敗: %w", err)
			}
			if err := safariScreenshot(ctx, absPath); err != nil {
				return BrowserRunResult{}, err
			}
			result.Screenshots = append(result.Screenshots, relPath)
		}

		result.ExecutedSteps = append(result.ExecutedSteps, step)
	}

	if err := writeBrowserResult(artifactDir, result); err != nil {
		return BrowserRunResult{}, err
	}
	return result, nil
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

func writeBrowserResult(artifactDir string, result BrowserRunResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 browser 結果失敗: %w", err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "result.json"), data, 0644); err != nil {
		return fmt.Errorf("寫入 browser 結果失敗: %w", err)
	}
	return nil
}

func validateBrowserTarget(parsed *url.URL) error {
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return fmt.Errorf("browser script: 無效 URL %q", parsed.String())
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || !strings.Contains(host, ".") {
		return fmt.Errorf("browser script: 禁止存取本機或內網位址 %q", parsed.String())
	}
	if isRebindingHost(host) {
		return fmt.Errorf("browser script: 禁止存取可能映射到本機的網域 %q", parsed.String())
	}

	if ip := net.ParseIP(host); ip != nil {
		if isPrivateBrowserIP(ip) {
			return fmt.Errorf("browser script: 禁止存取本機或內網位址 %q", parsed.String())
		}
	}
	return nil
}

func validateBrowserTargetResolved(ctx context.Context, parsed *url.URL) error {
	if err := validateBrowserTarget(parsed); err != nil {
		return err
	}

	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil {
		return fmt.Errorf("browser script: 無法驗證目標位址 %q", parsed.String())
	}
	for _, addr := range addrs {
		if isPrivateBrowserIP(addr.IP) {
			return fmt.Errorf("browser script: 禁止存取本機或內網位址 %q", parsed.String())
		}
	}
	return nil
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

func isPrivateBrowserIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	v4 := ip.To4()
	if v4 == nil {
		return false
	}

	switch {
	case v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127:
		return true
	case v4[0] == 169 && v4[1] == 254:
		return true
	case v4[0] == 198 && (v4[1] == 18 || v4[1] == 19):
		return true
	default:
		return false
	}
}

func isRebindingHost(host string) bool {
	switch {
	case strings.HasSuffix(host, ".nip.io"),
		strings.HasSuffix(host, ".sslip.io"),
		strings.HasSuffix(host, ".xip.io"):
		return true
	default:
		return false
	}
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
