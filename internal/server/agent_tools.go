package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go-chatbot/internal/workbench"
)

const (
	maxAgentToolFiles    = 3
	maxAgentToolCalls    = 4
	toolPlanningTimeout  = 8 * time.Second
	toolReadPreviewBytes = 12000
)

type plannedToolCall struct {
	Tool  string `json:"tool"`
	Query string `json:"query,omitempty"`
	Path  string `json:"path,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type toolPlanEnvelope struct {
	ToolCalls []plannedToolCall `json:"tool_calls"`
}

func (s *Server) gatherReadOnlyToolContext(ctx context.Context, question string, chatModel string) string {
	root, repoMap, ok := s.activeProjectWorkspace(nil)
	if !ok || repoMap == nil {
		return ""
	}

	planned := s.planReadOnlyToolCalls(ctx, repoMap, question, chatModel)
	if len(planned) == 0 {
		return s.gatherDeterministicToolContext(root, repoMap, question)
	}

	contextText := s.executeReadOnlyToolCalls(root, repoMap, planned)
	if contextText != "" {
		return contextText
	}
	return s.gatherDeterministicToolContext(root, repoMap, question)
}

func (s *Server) planReadOnlyToolCalls(ctx context.Context, repoMap *workbench.RepoMap, question string, chatModel string) []plannedToolCall {
	planCtx, cancel := context.WithTimeout(ctx, toolPlanningTimeout)
	defer cancel()

	result, err := s.chain.PlanToolCalls(
		planCtx,
		chatModel,
		readOnlyToolPlannerSystemPrompt(),
		readOnlyToolPlannerUserPrompt(repoMap, question),
	)
	if err != nil {
		slog.Debug("read-only tool planner failed", "error", err)
		return nil
	}

	calls := parseToolCalls(result)
	if len(calls) == 0 {
		slog.Debug("read-only tool planner returned no usable calls", "response", result)
		s.recordTaskTrace("tool_plan", "No read-only project tools selected", nil)
		return nil
	}
	s.recordTaskTrace("tool_plan", "Planned read-only project tools", map[string]string{
		"calls": fmt.Sprintf("%d", len(calls)),
		"plan":  summarizePlannedToolCalls(calls),
	})
	return calls
}

func (s *Server) executeReadOnlyToolCalls(root string, repoMap *workbench.RepoMap, calls []plannedToolCall) string {
	var b strings.Builder
	b.WriteString("## Read-only project tool results\n")
	b.WriteString("These results were gathered from safe local project tools before answering. Use them as additional context, but do not claim files were modified.\n\n")

	executed := 0
	for _, call := range calls {
		if executed >= maxAgentToolCalls {
			break
		}
		added := false
		switch call.Tool {
		case "search_files":
			added = appendSearchFilesResult(&b, repoMap, call)
		case "summarize_file":
			added = appendFileSummaryResult(&b, root, repoMap, call.Path)
		case "read_file":
			added = appendFileReadResult(&b, root, repoMap, call.Path)
		}
		if !added {
			continue
		}
		executed++
		s.recordTaskTrace("tool_call", "Ran read-only project tool", map[string]string{
			"tool":  call.Tool,
			"query": call.Query,
			"path":  call.Path,
		})
	}

	if executed == 0 {
		return ""
	}
	return b.String()
}

func (s *Server) gatherDeterministicToolContext(root string, repoMap *workbench.RepoMap, question string) string {
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
		if appendFileSummaryResult(&b, root, repoMap, result.Path) {
			count++
		}
	}

	if count == 0 {
		return ""
	}
	s.recordTaskTrace("tool_read", "Prepared deterministic project tool context", map[string]string{
		"query": query,
		"files": fmt.Sprintf("%d", count),
	})
	return b.String()
}

func summarizePlannedToolCalls(calls []plannedToolCall) string {
	var parts []string
	for _, call := range calls {
		switch call.Tool {
		case "search_files":
			parts = append(parts, fmt.Sprintf("search_files:%s", call.Query))
		case "summarize_file", "read_file":
			parts = append(parts, fmt.Sprintf("%s:%s", call.Tool, call.Path))
		}
	}
	return strings.Join(parts, " | ")
}

func appendSearchFilesResult(b *strings.Builder, repoMap *workbench.RepoMap, call plannedToolCall) bool {
	query := strings.TrimSpace(call.Query)
	if query == "" {
		return false
	}
	limit := call.Limit
	if limit <= 0 || limit > 12 {
		limit = 8
	}
	results := workbench.SearchProjectFiles(repoMap, workbench.FileSearchOptions{
		Query: query,
		Limit: limit,
	})
	if len(results) == 0 {
		return false
	}

	fmt.Fprintf(b, "### Tool: search_files(%q)\n", query)
	for _, result := range results {
		fmt.Fprintf(b, "- %s", result.Path)
		if result.Kind != "" {
			fmt.Fprintf(b, " [%s", result.Kind)
			if result.Language != "" {
				fmt.Fprintf(b, ", %s", result.Language)
			}
			b.WriteString("]")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return true
}

func appendFileSummaryResult(b *strings.Builder, root string, repoMap *workbench.RepoMap, path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	summary, err := workbench.SummarizeProjectFile(root, repoMap, path)
	if err != nil {
		return false
	}

	fmt.Fprintf(b, "### Tool: summarize_file(%s)\n", summary.Path)
	fmt.Fprintf(b, "- kind: %s\n", summary.Kind)
	if summary.Language != "" {
		fmt.Fprintf(b, "- language: %s\n", summary.Language)
	}
	if summary.Description != "" {
		fmt.Fprintf(b, "- description: %s\n", summary.Description)
	}
	if len(summary.Imports) > 0 {
		fmt.Fprintf(b, "- imports: %s\n", strings.Join(compactList(summary.Imports, 8), ", "))
	}
	if len(summary.Symbols) > 0 {
		var symbols []string
		for _, symbol := range summary.Symbols {
			symbols = append(symbols, fmt.Sprintf("%s %s:%d", symbol.Kind, symbol.Name, symbol.Line))
			if len(symbols) >= 8 {
				break
			}
		}
		fmt.Fprintf(b, "- symbols: %s\n", strings.Join(symbols, ", "))
	}
	if strings.TrimSpace(summary.Excerpt) != "" {
		fmt.Fprintf(b, "\nExcerpt:\n%s\n", trimToolExcerpt(summary.Excerpt, 2000))
	}
	b.WriteString("\n")
	return true
}

func appendFileReadResult(b *strings.Builder, root string, repoMap *workbench.RepoMap, path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	read, err := workbench.ReadProjectFile(root, repoMap, workbench.FileReadRequest{
		Path:     path,
		MaxBytes: toolReadPreviewBytes,
	})
	if err != nil || read.Binary || strings.TrimSpace(read.Content) == "" {
		return false
	}

	fmt.Fprintf(b, "### Tool: read_file(%s)\n", read.Path)
	fmt.Fprintf(b, "- kind: %s\n", read.Kind)
	if read.Language != "" {
		fmt.Fprintf(b, "- language: %s\n", read.Language)
	}
	if read.Truncated {
		b.WriteString("- note: preview truncated\n")
	}
	fmt.Fprintf(b, "\nContent preview:\n%s\n\n", trimToolExcerpt(read.Content, 3000))
	return true
}

func readOnlyToolPlannerSystemPrompt() string {
	return `You select read-only project tools for a local coding assistant.
Return only compact JSON with this exact shape:
{"tool_calls":[{"tool":"search_files","query":"text","limit":8},{"tool":"summarize_file","path":"relative/path"},{"tool":"read_file","path":"relative/path"}]}

Rules:
- Use at most 4 tool calls.
- Allowed tools are search_files, summarize_file, and read_file.
- Use only relative paths shown in the workspace map.
- Prefer summarize_file before read_file unless exact file contents are clearly needed.
- If no project tool is useful, return {"tool_calls":[]}.
- Do not include markdown, comments, or extra keys.`
}

func readOnlyToolPlannerUserPrompt(repoMap *workbench.RepoMap, question string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "User question:\n%s\n\n", strings.TrimSpace(question))
	if len(repoMap.Groups) > 0 {
		b.WriteString("Workspace folder groups:\n")
		for i, group := range repoMap.Groups {
			if i >= 12 {
				fmt.Fprintf(&b, "... %d more groups omitted\n", len(repoMap.Groups)-i)
				break
			}
			fmt.Fprintf(&b, "- %s: %d files, %d tests\n", group.Path, group.FileCount, group.TestCount)
		}
		b.WriteString("\n")
	}
	if len(repoMap.Packages) > 0 {
		b.WriteString("Code packages:\n")
		for i, pkg := range repoMap.Packages {
			if i >= 20 {
				fmt.Fprintf(&b, "... %d more packages omitted\n", len(repoMap.Packages)-i)
				break
			}
			fmt.Fprintf(&b, "- %s", pkg.Name)
			if pkg.Path != "" {
				fmt.Fprintf(&b, " in %s", pkg.Path)
			}
			fmt.Fprintf(&b, " (%s, %d files, %d tests)\n", pkg.Language, pkg.FileCount, pkg.TestCount)
		}
		b.WriteString("\n")
	}
	b.WriteString("Workspace files available to read:\n")
	for i, file := range repoMap.Files {
		if i >= 80 {
			fmt.Fprintf(&b, "... %d more files omitted\n", len(repoMap.Files)-i)
			break
		}
		fmt.Fprintf(&b, "- %s", file.Path)
		if file.Kind != "" {
			fmt.Fprintf(&b, " [%s", file.Kind)
			if file.Language != "" {
				fmt.Fprintf(&b, ", %s", file.Language)
			}
			b.WriteString("]")
		}
		b.WriteString("\n")
	}
	if len(repoMap.Symbols) > 0 {
		b.WriteString("\nUseful symbols:\n")
		for i, symbol := range repoMap.Symbols {
			if i >= 50 {
				fmt.Fprintf(&b, "... %d more symbols omitted\n", len(repoMap.Symbols)-i)
				break
			}
			fmt.Fprintf(&b, "- %s %s in %s:%d\n", symbol.Kind, symbol.Name, symbol.FilePath, symbol.Line)
		}
	}
	return b.String()
}

func parseToolCalls(raw string) []plannedToolCall {
	raw = extractJSONObject(raw)
	if raw == "" {
		return nil
	}

	var envelope toolPlanEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return nil
	}

	var calls []plannedToolCall
	for _, call := range envelope.ToolCalls {
		call.Tool = strings.ToLower(strings.TrimSpace(call.Tool))
		call.Query = strings.TrimSpace(call.Query)
		call.Path = strings.TrimSpace(call.Path)
		switch call.Tool {
		case "search_files":
			if call.Query == "" {
				continue
			}
		case "summarize_file", "read_file":
			if call.Path == "" {
				continue
			}
		default:
			continue
		}
		calls = append(calls, call)
		if len(calls) >= maxAgentToolCalls {
			break
		}
	}
	return calls
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return ""
	}
	return raw[start : end+1]
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
