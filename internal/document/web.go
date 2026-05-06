package document

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var skipTags = map[string]bool{
	"script":   true,
	"style":    true,
	"noscript": true,
	"nav":      true,
	"footer":   true,
	"svg":      true,
	"iframe":   true,
	"head":     true,
}

func FetchURL(ctx context.Context, url string) ([]Document, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("URL must start with http:// or https://")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	contentType := resp.Header.Get("Content-Type")

	if strings.Contains(contentType, "text/plain") {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		content := strings.TrimSpace(string(body))
		if content == "" {
			return nil, fmt.Errorf("empty page at %s", url)
		}
		return []Document{{
			Content:  content,
			Metadata: map[string]string{"source": url, "type": "web", "title": url},
		}}, nil
	}

	body := io.LimitReader(resp.Body, 10*1024*1024)
	title, content, err := extractHTMLText(body)
	if err != nil {
		return nil, fmt.Errorf("extract text: %w", err)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("no readable text extracted from %s", url)
	}

	if title == "" {
		title = url
	}

	return []Document{{
		Content: content,
		Metadata: map[string]string{
			"source": url,
			"type":   "web",
			"title":  title,
		},
	}}, nil
}

func extractHTMLText(r io.Reader) (title, text string, err error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", "", fmt.Errorf("parse HTML: %w", err)
	}

	var sb strings.Builder
	var titleBuilder strings.Builder
	var inTitle bool

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if skipTags[n.Data] {
				return
			}
			if n.Data == "title" {
				inTitle = true
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				if inTitle {
					titleBuilder.WriteString(text)
				}
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "br", "div", "h1", "h2", "h3", "h4", "h5", "h6",
				"li", "tr", "blockquote", "pre", "article", "section":
				sb.WriteString("\n")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}

		if n.Type == html.ElementNode {
			if n.Data == "title" {
				inTitle = false
			}
			switch n.Data {
			case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6",
				"article", "section", "pre", "blockquote":
				sb.WriteString("\n")
			}
		}
	}

	walk(doc)

	raw := sb.String()
	lines := strings.Split(raw, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.TrimSpace(titleBuilder.String()), strings.Join(cleaned, "\n"), nil
}
