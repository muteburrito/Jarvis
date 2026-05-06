package websearch

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

type Result struct {
	Title   string
	URL     string
	Snippet string
}

func Search(ctx context.Context, query string, maxResults int) ([]Result, error) {
	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned HTTP %d", resp.StatusCode)
	}

	body := io.LimitReader(resp.Body, 2*1024*1024)
	return parseResults(body, maxResults)
}

func parseResults(r io.Reader, maxResults int) ([]Result, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var results []Result
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(results) >= maxResults {
			return
		}

		if isResultDiv(n) {
			if r := extractResult(n); r.URL != "" {
				results = append(results, r)
			}
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return results, nil
}

func isResultDiv(n *html.Node) bool {
	if n.Type != html.ElementNode || n.Data != "div" {
		return false
	}
	for _, a := range n.Attr {
		if a.Key == "class" && strings.Contains(a.Val, "result__body") {
			return true
		}
	}
	return false
}

func extractResult(n *html.Node) Result {
	var r Result
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			cls := attrVal(node, "class")
			if strings.Contains(cls, "result__a") {
				r.Title = textContent(node)
				href := attrVal(node, "href")
				r.URL = cleanDDGLink(href)
			}
			if strings.Contains(cls, "result__snippet") {
				r.Snippet = textContent(node)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return r
}

func cleanDDGLink(href string) string {
	if strings.Contains(href, "uddg=") {
		if u, err := url.Parse(href); err == nil {
			if actual := u.Query().Get("uddg"); actual != "" {
				return actual
			}
		}
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	return ""
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}
