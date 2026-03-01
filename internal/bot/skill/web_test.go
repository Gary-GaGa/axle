package skill

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestHTMLToText(t *testing.T) {
	h := "<html><body><p>Hello <b>World</b></p></body></html>"
	text := htmlToText(h)
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
	h := `<html><head><style>body{}</style></head><body><script>alert(1)</script><p>Content</p></body></html>`
	text := htmlToText(h)
	if strings.Contains(text, "alert") || strings.Contains(text, "body{}") {
		t.Errorf("should strip script/style: %q", text)
	}
	if !strings.Contains(text, "Content") {
		t.Error("should contain Content")
	}
}

func TestHTMLToText_BlockElements(t *testing.T) {
	h := `<div><h1>Title</h1><p>Para</p><br/><ul><li>Item</li></ul></div>`
	text := htmlToText(h)
	if !strings.Contains(text, "Title") || !strings.Contains(text, "Para") || !strings.Contains(text, "Item") {
		t.Errorf("htmlToText = %q", text)
	}
	if !strings.Contains(text, "\n") {
		t.Error("expected newlines for block elements")
	}
}

func TestHTMLToText_Noscript(t *testing.T) {
	h := `<body><noscript>hidden</noscript><p>visible</p></body>`
	text := htmlToText(h)
	if strings.Contains(text, "hidden") {
		t.Errorf("should strip noscript: %q", text)
	}
	if !strings.Contains(text, "visible") {
		t.Error("should contain visible")
	}
}

func TestGetAttr(t *testing.T) {
	node := &html.Node{
		Type: html.ElementNode,
		Data: "a",
		Attr: []html.Attribute{
			{Key: "href", Val: "https://example.com"},
			{Key: "class", Val: "link"},
		},
	}
	if got := getAttr(node, "href"); got != "https://example.com" {
		t.Errorf("getAttr(href) = %q", got)
	}
	if got := getAttr(node, "class"); got != "link" {
		t.Errorf("getAttr(class) = %q", got)
	}
	if got := getAttr(node, "missing"); got != "" {
		t.Errorf("getAttr(missing) = %q", got)
	}
}

func TestExtractText(t *testing.T) {
	// Simple text node
	textNode := &html.Node{Type: html.TextNode, Data: "hello"}
	if got := extractText(textNode); got != "hello" {
		t.Errorf("extractText(text) = %q", got)
	}

	// Nested nodes
	parent := &html.Node{Type: html.ElementNode, Data: "p"}
	child := &html.Node{Type: html.TextNode, Data: "inner text"}
	parent.AppendChild(child)
	if got := extractText(parent); got != "inner text" {
		t.Errorf("extractText(nested) = %q", got)
	}
}

func TestExtractDDGURL(t *testing.T) {
	url := extractDDGURL("//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com&rut=abc")
	if url != "https://example.com" {
		t.Errorf("extractDDGURL = %q", url)
	}

	url = extractDDGURL("https://example.com")
	if url != "https://example.com" {
		t.Errorf("direct URL = %q", url)
	}

	url = extractDDGURL("")
	if url != "" {
		t.Errorf("empty = %q", url)
	}
}

func TestParseDDGHTML(t *testing.T) {
	body := `<html><body>
		<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com">Example</a>
		<a class="result__snippet">A description</a>
		<a class="result__a" href="https://golang.org">Go Lang</a>
	</body></html>`
	results, err := parseDDGHTML(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 1 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Title != "Example" {
		t.Errorf("title = %q", results[0].Title)
	}
	if results[0].URL != "https://example.com" {
		t.Errorf("url = %q", results[0].URL)
	}
}

func TestParseDDGHTML_Empty(t *testing.T) {
	results, err := parseDDGHTML(strings.NewReader("<html><body></body></html>"))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseDDGHTML_MaxResults(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 20; i++ {
		sb.WriteString(`<a class="result__a" href="https://example.com">Title</a>`)
	}
	sb.WriteString("</body></html>")
	results, err := parseDDGHTML(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) > maxSearchItems {
		t.Errorf("expected max %d, got %d", maxSearchItems, len(results))
	}
}

func TestWebFetch_HTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><p>Hello Test</p></body></html>"))
	}))
	defer srv.Close()

	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	text, err := WebFetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Hello Test") {
		t.Errorf("WebFetch = %q", text)
	}
}

func TestWebFetch_PlainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("raw content"))
	}))
	defer srv.Close()

	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	text, err := WebFetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if text != "raw content" {
		t.Errorf("WebFetch plain = %q", text)
	}
}

func TestWebFetch_EmptyHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>   </body></html>"))
	}))
	defer srv.Close()

	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	text, err := WebFetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "無可讀文字") {
		t.Errorf("empty html = %q", text)
	}
}

func TestWebFetch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	_, err := WebFetch(context.Background(), srv.URL)
	if err == nil {
		t.Error("expected error for 500")
	}
}

func TestWebFetch_Canceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	_, err := WebFetch(ctx, srv.URL)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestWebFetch_AddHTTPS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	// Can't easily test https:// prefix addition with httptest, but test bad URL fails
	_, err := WebFetch(context.Background(), "://invalid")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestWebSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>
			<a class="result__a" href="https://golang.org">Go Programming</a>
		</body></html>`))
	}))
	defer srv.Close()

	old := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = old }()

	// WebSearch builds URL from duckduckgo, but we can't redirect to our test server easily.
	// Instead, test parseDDGHTML directly (already covered above).
	// For integration, test that canceled context returns error.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := WebSearch(ctx, "test")
	if err == nil {
		t.Error("expected error for canceled context")
	}
}
