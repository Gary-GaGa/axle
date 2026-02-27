package skill

import (
	"strings"
	"testing"
)

func TestHTMLToText(t *testing.T) {
	html := "<html><body><p>Hello <b>World</b></p></body></html>"
	text := htmlToText(html)
	if !strings.Contains(text, "Hello") || !strings.Contains(text, "World") {
		t.Errorf("htmlToText = %q", text)
	}
}

func TestHTMLToText_Empty(t *testing.T) {
	text := htmlToText("")
	if text != "" {
		t.Errorf("expected empty, got %q", text)
	}
}

func TestHTMLToText_ScriptStyle(t *testing.T) {
	html := `<html><head><style>body{}</style></head><body><script>alert(1)</script><p>Content</p></body></html>`
	text := htmlToText(html)
	if strings.Contains(text, "alert") || strings.Contains(text, "body{}") {
		t.Errorf("should strip script/style: %q", text)
	}
	if !strings.Contains(text, "Content") {
		t.Error("should contain Content")
	}
}

func TestGetAttr(t *testing.T) {
	// Test the getAttr helper with a simulated token
	// This is indirectly tested through parseDDGHTML, but let's test a basic parse
	html := `<a href="https://example.com">Link</a>`
	text := htmlToText(html)
	if !strings.Contains(text, "Link") {
		t.Errorf("expected Link in %q", text)
	}
}

func TestExtractText(t *testing.T) {
	// extractText takes *html.Node, test via htmlToText
	text := htmlToText("<div><p>First</p><p>Second</p></div>")
	if !strings.Contains(text, "First") || !strings.Contains(text, "Second") {
		t.Errorf("htmlToText = %q", text)
	}
}

func TestExtractDDGURL(t *testing.T) {
	// DuckDuckGo URL format
	url := extractDDGURL("//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com&rut=abc")
	if url != "https://example.com" {
		t.Errorf("extractDDGURL = %q", url)
	}

	// Direct URL
	url = extractDDGURL("https://example.com")
	if url != "https://example.com" {
		t.Errorf("direct URL = %q", url)
	}

	// Empty
	url = extractDDGURL("")
	if url != "" {
		t.Errorf("empty = %q", url)
	}
}
