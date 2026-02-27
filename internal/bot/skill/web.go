package skill

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	webTimeout     = 30 * time.Second
	maxFetchBytes  = 100 * 1024 // 100 KB
	maxSearchItems = 8
)

var httpClient = &http.Client{Timeout: webTimeout}

// WebSearchResult is one result from DuckDuckGo HTML search.
type WebSearchResult struct {
	Title string
	URL   string
	Desc  string
}

// WebSearch queries DuckDuckGo HTML (no API key needed) and returns results.
func WebSearch(ctx context.Context, query string) ([]WebSearchResult, error) {
	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("建立搜尋請求失敗: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Axle/1.0)")

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, context.Canceled
		}
		return nil, fmt.Errorf("搜尋請求失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("搜尋回應異常: %d", resp.StatusCode)
	}

	return parseDDGHTML(resp.Body)
}

// parseDDGHTML extracts search results from DuckDuckGo HTML page.
func parseDDGHTML(r io.Reader) ([]WebSearchResult, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("解析搜尋結果失敗: %w", err)
	}

	var results []WebSearchResult
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(results) >= maxSearchItems {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			cls := getAttr(n, "class")
			if strings.Contains(cls, "result__a") {
				href := getAttr(n, "href")
				title := extractText(n)
				if href != "" && title != "" {
					// DuckDuckGo wraps URLs in redirect; extract actual URL
					actualURL := extractDDGURL(href)
					desc := ""
					// Walk siblings to find snippet
					for sib := n.Parent; sib != nil; sib = sib.NextSibling {
						if sib.Type == html.ElementNode {
							sibCls := getAttr(sib, "class")
							if strings.Contains(sibCls, "result__snippet") {
								desc = extractText(sib)
								break
							}
						}
					}
					results = append(results, WebSearchResult{
						Title: title,
						URL:   actualURL,
						Desc:  desc,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return results, nil
}

// WebFetch downloads a URL and extracts readable text content.
func WebFetch(ctx context.Context, rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("建立請求失敗: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Axle/1.0)")

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return "", context.Canceled
		}
		return "", fmt.Errorf("請求失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("回應異常: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes))
	if err != nil {
		return "", fmt.Errorf("讀取回應失敗: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		text := htmlToText(string(body))
		if text == "" {
			return "（頁面無可讀文字內容）", nil
		}
		return text, nil
	}
	// Non-HTML: return raw text (capped)
	return string(body), nil
}

// htmlToText does a simple extraction of visible text from HTML.
func htmlToText(raw string) string {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return raw
	}
	var sb strings.Builder
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		// Skip script/style tags
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "noscript") {
			return
		}
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		}
		// Add linebreaks for block elements
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "br", "div", "h1", "h2", "h3", "h4", "h5", "h6", "li", "tr":
				sb.WriteString("\n")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)
	return strings.TrimSpace(sb.String())
}

// ── HTML helpers ──────────────────────────────────────────────────────────────

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return strings.TrimSpace(sb.String())
}

func extractDDGURL(href string) string {
	if u, err := url.Parse(href); err == nil {
		if actual := u.Query().Get("uddg"); actual != "" {
			return actual
		}
	}
	return href
}
