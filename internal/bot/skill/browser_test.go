package skill

import (
	"strings"
	"testing"
)

func TestParseBrowserScript(t *testing.T) {
	script := `
# comment
open https://example.com
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
	if plan.Steps[0].Action != "open" || plan.Steps[0].Arg != "https://example.com" {
		t.Fatalf("unexpected open step: %+v", plan.Steps[0])
	}
	if plan.Steps[2].Action != "extract" || plan.Steps[2].Arg != "main" {
		t.Fatalf("unexpected extract step: %+v", plan.Steps[2])
	}
}

func TestParseBrowserScriptDefaultsAndErrors(t *testing.T) {
	plan, err := ParseBrowserScript("open https://example.com\nextract\nscreenshot")
	if err != nil {
		t.Fatalf("ParseBrowserScript default extract: %v", err)
	}
	if plan.Steps[1].Arg != "body" {
		t.Fatalf("expected default body extract, got %q", plan.Steps[1].Arg)
	}

	if _, err := ParseBrowserScript("wait nope"); err == nil || !strings.Contains(err.Error(), "duration") {
		t.Fatalf("expected duration error, got %v", err)
	}
	if _, err := ParseBrowserScript("open file:///tmp/test"); err == nil || !strings.Contains(err.Error(), "無效 URL") {
		t.Fatalf("expected invalid url error, got %v", err)
	}
	if _, err := ParseBrowserScript("open http://localhost:3000"); err == nil || !strings.Contains(err.Error(), "禁止存取") {
		t.Fatalf("expected localhost block, got %v", err)
	}
	if _, err := ParseBrowserScript("open http://192.168.1.10"); err == nil || !strings.Contains(err.Error(), "禁止存取") {
		t.Fatalf("expected private IP block, got %v", err)
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
