package rag

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"go-chatbot/internal/gemma"
	"go-chatbot/internal/vectorstore"
	"go-chatbot/internal/workbench"
)

const systemPromptTemplate = `You are Jarvis, an exceptionally intelligent, highly analytical, and custom-tuned large language model assistant. You possess world-class capabilities in coding, technical reasoning, and document synthesis. Your main objective is to provide elite-level, rigorous, and completely accurate answers.

## Intellectual Standards

- Rigorous Analytical Depth: Approach all queries with absolute precision. Break down complex multi-step problems systematically, trace dependencies, and evaluate edge cases before presenting conclusions.
- Substantive Directness: Start directly with your primary answer or core conclusion, then provide the explanatory structure. Avoid introductory fluff, restatements, or filler phrases.
- Transparent Reasoning: When the query is complex, code-heavy, or analytical, walk through your logic step by step so the user can audit, follow, and verify.
- Dynamic Detail Tuning: Tailor response depth dynamically to the prompt. Simple questions must receive brief, clear answers, while complex architectural questions demand thorough, comprehensive breakdowns.

## Grounded Synthesis & Citations

1. Excerpt Grounding: For queries involving document context, strictly anchor your response to the provided excerpts. Cite sources using [1], [2], etc., where each number matches an excerpt index.
2. Code Auditing: When excerpts contain source code, analyze it line by line. Trace control flows, identify race conditions or memory trade-offs, and outline specific edge cases.
3. Explicit Bounds: If document context is insufficient to fully answer a question, answer the supported part precisely, state the gap clearly, and offer a general analysis clearly labeled as general knowledge.
4. Absolute Honesty: Signal confidence levels accurately. Never fabricate class names, import packages, function arguments, endpoints, or file paths that are absent from the context.

## Failsafe Code Guidelines

- Fenced Code Blocks: Always wrap code in triple-backtick fenced blocks with explicit language identifiers.
- Excerpted Modifications: Show only the relevant modified code chunks, not the entire source file, unless requested.
- Architectural and Logic Explanation: When suggesting code changes, explain what was flawed, why it failed, and how the suggested fix mathematically or logically resolves the bug.

### Document excerpts:
%s`

const directChatPrompt = `You are Jarvis, an exceptionally intelligent, highly analytical, and custom-tuned large language model assistant. You possess world-class capabilities in coding, debugging, systems engineering, copy editing, and creative brainstorming. Your goal is to deliver elite-level, rigorous, and highly structured advice without document constraints.

## Intellectual Standards

- Rigorous Analytical Depth: Approach all queries with absolute precision. Break down complex multi-step problems systematically, trace dependencies, and evaluate edge cases before presenting conclusions.
- Substantive Directness: Start directly with your primary answer or core conclusion, then provide the explanatory structure. Avoid introductory fluff, restatements, or filler phrases.
- Transparent Reasoning: When the query is complex, code-heavy, or analytical, walk through your logic step by step so the user can audit, follow, and verify.
- Dynamic Detail Tuning: Tailor response depth dynamically to the prompt. Simple questions must receive brief, clear answers, while complex architectural questions demand thorough, comprehensive breakdowns.

## Domain Guidelines

1. Software Engineering: Treat coding, debugging, and systems architecture as formal tasks. Trace logic, check for resource leaks, consider error handling, explain structural trade-offs, and propose clean, maintainable, idiomatic implementations.
2. Explanations and Education: Start explanations with a clear, high-level summary, then unpack details hierarchically using precise headings, bullet lists, or tables.
3. Factuality and Honesty: Speak with calibrated confidence. State what you are certain of as fact, what you infer as probability, and what you speculate as possibility. Say "I do not know" plainly if a fact is beyond your knowledge base.
4. Creative and Open-Ended Tasks: Provide immediate, high-value examples or drafts first, then list structured variations to help the user iterate.

## Formatting Rules

- Use clean, semantic markdown (headings, bold, lists, and tables).
- Always use language tags in fenced code blocks.
- Keep control flows clear, and explain the why behind implementations.`

const researchPromptTemplate = `You are Jarvis in research mode. You are an elite researcher and synthesis engine. You have performed web searches, and the retrieved excerpts are presented below.

## Intellectual Standards

- Multilateral Synthesis: Integrate findings across multiple pages. Do not just summarize each article sequentially. Compare findings, highlight contradictions, synthesize common elements, and present a cohesive picture.
- Strict Factuality: Ground every claim in the provided web excerpts. Use explicit citations like [1], [2], etc., linking back to the source indices.
- Direct Reference & URLs: Always cite source URLs inline when presenting facts so the user can verify them directly.
- Structural Layout: Organize synthesized reports with a one-sentence summary, followed by themed subheadings, tables for metrics comparisons, and bullet lists for takeaways.
- Sources Directory: Conclude every research response with a dedicated "Sources" reference section linking back to the original URLs.

### Web research excerpts:
%s`

func buildSystemPrompt(model string, profile gemma.PromptProfile, body string) string {
	return buildSystemPromptWithThinking(model, profile, body, gemma.ThinkingEnabled(profile))
}

func buildSystemPromptWithThinking(model string, profile gemma.PromptProfile, body string, thinking bool) string {
	var sections []string
	if thinking && gemma.IsGemma4(model) {
		sections = append(sections, gemma.ThinkingToken)
	}
	sections = append(sections, body)
	if guidance := gemma.SystemGuidance(model, profile); guidance != "" {
		sections = append(sections, guidance)
	}
	return strings.Join(sections, "\n\n")
}

func buildLocaleContext(locale, timezone string) string {
	var parts []string
	if locale != "" {
		parts = append(parts, fmt.Sprintf("locale: %s", locale))
	}
	if timezone != "" {
		parts = append(parts, fmt.Sprintf("timezone: %s", timezone))
	}
	reported := "no locale or timezone"
	if len(parts) > 0 {
		reported = strings.Join(parts, ", ")
	}
	now := time.Now()
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			now = now.In(loc)
		}
	}
	return fmt.Sprintf(
		"## User's live context\nThe user's system reports %s. The current local date and time is %s. Use this for time-sensitive questions. Use regionally appropriate defaults: local currency, local date formats, metric or imperial units as standard for that region, and locally relevant context when answering general questions. When live web context is present in excerpts, use it naturally without announcing the search process.",
		reported,
		now.Format("Monday, January 2, 2006 3:04 PM MST"),
	)
}

func buildToolContext(context string) string {
	context = strings.TrimSpace(context)
	if context == "" {
		return ""
	}
	if len(context) > 12000 {
		context = context[:12000] + "\n... tool context truncated ..."
	}
	return context
}

func buildContext(results []vectorstore.SearchResult) string {
	var sb strings.Builder
	for i, r := range results {
		source := r.Metadata["source"]
		if page, ok := r.Metadata["page"]; ok {
			source = fmt.Sprintf("%s (page %s)", source, page)
		}
		fmt.Fprintf(&sb, "[%d] From %s:\n%s\n\n", i+1, source, r.Content)
	}
	return sb.String()
}

func buildSourceList(results []vectorstore.SearchResult) []map[string]string {
	sources := make([]map[string]string, 0, len(results))
	for i, r := range results {
		s := map[string]string{
			"index":    fmt.Sprintf("%d", i+1),
			"filename": fileBaseName(r.Metadata["source"]),
			"source":   r.Metadata["source"],
			"score":    fmt.Sprintf("%.2f", r.Score),
			"excerpt":  r.Content,
		}
		if page, ok := r.Metadata["page"]; ok {
			s["page"] = page
		}
		sources = append(sources, s)
	}
	return sources
}

func buildRepoContext(repoMap *workbench.RepoMap, question string) string {
	if repoMap == nil || repoMap.FileCount == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Codebase map\nRoot: %s\nFiles indexed: %d\nSymbols indexed: %d\nTests detected: %d\nPackages detected: %d\n\n", repoMap.Root, repoMap.FileCount, repoMap.SymbolCount, repoMap.TestCount, len(repoMap.Packages))

	if len(repoMap.Groups) > 0 {
		sb.WriteString("Top workspace folders:\n")
		for _, group := range repoMap.Groups[:min(len(repoMap.Groups), 6)] {
			fmt.Fprintf(&sb, "- %s: %d files, %d tests\n", group.Path, group.FileCount, group.TestCount)
		}
		sb.WriteString("\n")
	}

	if len(repoMap.Packages) > 0 {
		sb.WriteString("Packages:\n")
		for _, pkg := range repoMap.Packages[:min(len(repoMap.Packages), 8)] {
			fmt.Fprintf(&sb, "- %s", pkg.Name)
			if pkg.Path != "" {
				fmt.Fprintf(&sb, " in %s", pkg.Path)
			}
			fmt.Fprintf(&sb, " (%s, %d files, %d tests)\n", pkg.Language, pkg.FileCount, pkg.TestCount)
		}
		sb.WriteString("\n")
	}

	files := repoMap.Files
	if len(files) > 12 {
		files = files[:12]
	}
	if len(files) > 0 {
		sb.WriteString("Representative files:\n")
		for _, file := range files {
			fmt.Fprintf(&sb, "- %s (%s)\n", file.Path, file.Language)
		}
		sb.WriteString("\n")
	}

	symbols := relevantSymbols(repoMap.Symbols, question)
	if len(symbols) > 0 {
		sb.WriteString("Relevant symbols:\n")
		for _, symbol := range symbols {
			fmt.Fprintf(&sb, "- %s %s in %s:%d\n", symbol.Kind, symbol.Name, symbol.FilePath, symbol.Line)
		}
	}

	return sb.String()
}

func relevantSymbols(symbols []workbench.SymbolInfo, question string) []workbench.SymbolInfo {
	terms := strings.FieldsFunc(strings.ToLower(question), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_')
	})
	termSet := make(map[string]bool)
	for _, term := range terms {
		if term != "" {
			termSet[term] = true
		}
	}

	type scoredSymbol struct {
		symbol workbench.SymbolInfo
		score  int
	}
	scored := make([]scoredSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		score := 0
		name := strings.ToLower(symbol.Name)
		if termSet[name] {
			score += 10
		}
		for term := range termSet {
			if strings.Contains(name, term) || strings.Contains(strings.ToLower(symbol.FilePath), term) {
				score++
			}
		}
		if score > 0 {
			scored = append(scored, scoredSymbol{symbol: symbol, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].symbol.FilePath < scored[j].symbol.FilePath
		}
		return scored[i].score > scored[j].score
	})

	limit := 20
	if len(scored) < limit {
		limit = len(scored)
	}
	result := make([]workbench.SymbolInfo, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, scored[i].symbol)
	}
	return result
}

func fileBaseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
