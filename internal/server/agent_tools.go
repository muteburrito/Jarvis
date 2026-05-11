package server

import (
	"fmt"
	"strings"

	"go-chatbot/internal/workbench"
)

const maxAgentToolFiles = 3

func (s *Server) gatherReadOnlyToolContext(question string) string {
	root, repoMap, ok := s.activeProjectWorkspace(nil)
	if !ok || repoMap == nil {
		return ""
	}

	query := toolSearchQuery(question)
	results := workbench.SearchProjectFiles(repoMap, workbench.FileSearchOptions{
		Query: query,
		Limit: 8,
	})
	if len(results) == 0 && query != "" {
		results = workbench.SearchProjectFiles(repoMap, workbench.FileSearchOptions{Limit: 8})
	}
	if len(results) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Read-only project tool results\n")
	b.WriteString("These results were gathered from safe local project tools before answering. Use them as additional context, but do not claim files were modified.\n\n")

	count := 0
	for _, result := range results {
		if count >= maxAgentToolFiles {
			break
		}
		summary, err := workbench.SummarizeProjectFile(root, repoMap, result.Path)
		if err != nil {
			continue
		}
		count++
		fmt.Fprintf(&b, "### %s\n", summary.Path)
		fmt.Fprintf(&b, "- kind: %s\n", summary.Kind)
		if summary.Language != "" {
			fmt.Fprintf(&b, "- language: %s\n", summary.Language)
		}
		if summary.Description != "" {
			fmt.Fprintf(&b, "- description: %s\n", summary.Description)
		}
		if len(summary.Imports) > 0 {
			fmt.Fprintf(&b, "- imports: %s\n", strings.Join(compactList(summary.Imports, 8), ", "))
		}
		if len(summary.Symbols) > 0 {
			var symbols []string
			for _, symbol := range summary.Symbols {
				symbols = append(symbols, fmt.Sprintf("%s %s:%d", symbol.Kind, symbol.Name, symbol.Line))
				if len(symbols) >= 8 {
					break
				}
			}
			fmt.Fprintf(&b, "- symbols: %s\n", strings.Join(symbols, ", "))
		}
		if strings.TrimSpace(summary.Excerpt) != "" {
			fmt.Fprintf(&b, "\nExcerpt:\n%s\n", trimToolExcerpt(summary.Excerpt, 2000))
		}
		b.WriteString("\n")
	}

	if count == 0 {
		return ""
	}
	s.recordTaskTrace("tool_read", "Prepared read-only project tool context", map[string]string{
		"query": query,
		"files": fmt.Sprintf("%d", count),
	})
	return b.String()
}

func toolSearchQuery(question string) string {
	question = strings.ToLower(strings.TrimSpace(question))
	terms := strings.FieldsFunc(question, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-' || r == '/')
	})
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "this": true, "that": true,
		"what": true, "where": true, "when": true, "how": true, "why": true, "can": true,
		"you": true, "please": true, "explain": true, "inspect": true, "check": true,
		"file": true, "files": true, "code": true, "project": true,
	}
	for _, term := range terms {
		term = strings.Trim(term, ".,;:!?()[]{}<>\"'")
		if len(term) < 3 || stop[term] {
			continue
		}
		if strings.Contains(term, "/") || strings.Contains(term, ".") || strings.Contains(term, "_") || strings.Contains(term, "-") {
			return term
		}
	}
	for _, term := range terms {
		term = strings.Trim(term, ".,;:!?()[]{}<>\"'")
		if len(term) >= 4 && !stop[term] {
			return term
		}
	}
	return ""
}

func compactList(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func trimToolExcerpt(excerpt string, maxLen int) string {
	excerpt = strings.TrimSpace(excerpt)
	if len(excerpt) <= maxLen {
		return excerpt
	}
	return excerpt[:maxLen] + "\n... excerpt truncated ..."
}
