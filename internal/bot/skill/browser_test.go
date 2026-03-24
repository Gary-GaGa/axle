package skill

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	infrafiles "github.com/garyellow/axle/internal/infrastructure/files"
)

func TestParseBrowserScript(t *testing.T) {
	script := `
# comment
open https://1.1.1.1
wait 2s
extract main
screenshot docs/example-home.png
`

	plan, err := ParseBrowserScript(script)
	if err != nil {
		t.Fatalf("ParseBrowserScript: %v", err)
	}
	if len(plan.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Action != "open" || plan.Steps[0].Arg != "https://1.1.1.1" {
		t.Fatalf("unexpected open step: %+v", plan.Steps[0])
	}
	if plan.Steps[2].Action != "extract" || plan.Steps[2].Arg != "main" {
		t.Fatalf("unexpected extract step: %+v", plan.Steps[2])
	}
}

func TestParseBrowserScriptDefaultsAndErrors(t *testing.T) {
	plan, err := ParseBrowserScript("open https://1.1.1.1\nextract\nscreenshot")
	if err != nil {
		t.Fatalf("ParseBrowserScript default extract: %v", err)
	}
	if plan.Steps[1].Arg != "body" {
		t.Fatalf("expected default body extract, got %q", plan.Steps[1].Arg)
	}

	if _, err := ParseBrowserScript("wait nope"); err == nil || !strings.Contains(err.Error(), "duration") {
		t.Fatalf("expected duration error, got %v", err)
	}
	if _, err := ParseBrowserScript("wait 0s"); err == nil || !strings.Contains(err.Error(), "超過上限") {
		t.Fatalf("expected non-positive wait error, got %v", err)
	}
	if _, err := ParseBrowserScript("wait -1s"); err == nil || !strings.Contains(err.Error(), "超過上限") {
		t.Fatalf("expected negative wait error, got %v", err)
	}
	if _, err := ParseBrowserScript("wait 31s"); err == nil || !strings.Contains(err.Error(), "超過上限") {
		t.Fatalf("expected wait cap error, got %v", err)
	}
	if _, err := ParseBrowserScript("open file:///tmp/test"); err == nil || !strings.Contains(err.Error(), "無效 URL") {
		t.Fatalf("expected invalid url error, got %v", err)
	}
	if _, err := ParseBrowserScript("open http://localhost:3000"); err == nil || !strings.Contains(err.Error(), "public IP") {
		t.Fatalf("expected localhost block, got %v", err)
	}
	if _, err := ParseBrowserScript("open http://192.168.1.10"); err == nil || !strings.Contains(err.Error(), "禁止存取") {
		t.Fatalf("expected private IP block, got %v", err)
	}
	if _, err := ParseBrowserScript("open https://example.com"); err == nil || !strings.Contains(err.Error(), "public IP") {
		t.Fatalf("expected hostname to be rejected, got %v", err)
	}
	if _, err := ParseBrowserScript("extract body"); err == nil || !strings.Contains(err.Error(), "第一個步驟必須是 open") {
		t.Fatalf("expected first-open validation error, got %v", err)
	}
	if _, err := ParseBrowserScript("open https://1.1.1.1\nscreenshot ../oops.png"); err == nil {
		t.Fatal("expected invalid screenshot path to be rejected")
	}
}

func TestNormalizeBrowserScreenshotPath(t *testing.T) {
	got, err := normalizeBrowserScreenshotPath(filepath.Join(".axle", "browser", "run-1"), ".axle/browser/run-1/page.png")
	if err != nil {
		t.Fatalf("normalizeBrowserScreenshotPath() error = %v", err)
	}
	if got != filepath.Join(".axle", "browser", "run-1", "page.png") {
		t.Fatalf("unexpected normalized path = %q", got)
	}

	got, err = normalizeBrowserScreenshotPath(filepath.Join(".axle", "browser", "run-1"), "docs/page.png")
	if err != nil {
		t.Fatalf("normalizeBrowserScreenshotPath() relative path error = %v", err)
	}
	if got != filepath.Join(".axle", "browser", "run-1", "docs", "page.png") {
		t.Fatalf("unexpected remapped path = %q", got)
	}
}

func TestValidateBrowserWaitDuration(t *testing.T) {
	if err := validateBrowserWaitDuration(maxBrowserWait + time.Second); err == nil || !strings.Contains(err.Error(), "超過上限") {
		t.Fatalf("expected wait cap error, got %v", err)
	}
	if err := validateBrowserWaitDuration(0); err == nil || !strings.Contains(err.Error(), "超過上限") {
		t.Fatalf("expected non-positive wait error, got %v", err)
	}
}

func TestRunBrowserPlanDisabledByDefault(t *testing.T) {
	_, err := RunBrowserPlan(context.Background(), t.TempDir(), BrowserPlan{
		Steps: []BrowserStep{{Action: "open", Arg: "https://1.1.1.1"}},
	})
	if err == nil || !strings.Contains(err.Error(), unsafeBrowserEnvKey) {
		t.Fatalf("expected browser automation to be disabled by default, got %v", err)
	}
}

func TestBrowserRunResultSummary(t *testing.T) {
	summary := (BrowserRunResult{
		URL:           "https://example.com",
		ExtractedText: "hello world",
		Screenshots:   []string{"docs/example.png"},
		ArtifactDir:   ".axle/browser/run-1",
	}).Summary()

	if !strings.Contains(summary, "https://example.com") || !strings.Contains(summary, "example.png") {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func TestWriteBrowserResultRejectsHardLink(t *testing.T) {
	workspace := t.TempDir()
	root, _, err := infrafiles.OpenWorkspaceRoot(workspace)
	if err != nil {
		t.Fatalf("OpenWorkspaceRoot: %v", err)
	}
	defer root.Close()

	artifactDir, err := infrafiles.EnsureRootedDirectory(root, ".axle/browser/run-1", ".axle/browser/run-1", 0755)
	if err != nil {
		t.Fatalf("EnsureRootedDirectory: %v", err)
	}

	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Link(outside, filepath.Join(workspace, artifactDir, "result.json")); err != nil {
		t.Skipf("hard links unsupported: %v", err)
	}

	err = writeBrowserResult(root, artifactDir, BrowserRunResult{ArtifactDir: artifactDir})
	if err == nil {
		t.Fatal("expected hard link browser result write to be rejected")
	}
}
