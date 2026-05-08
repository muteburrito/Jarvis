package websearch

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	ollamaapi "github.com/ollama/ollama/api"

	"go-chatbot/internal/config"
	"go-chatbot/internal/document"
	"go-chatbot/internal/ollama"
	"go-chatbot/internal/vectorstore"
)

const maxSearchQueries = 3
const maxResultsPerQuery = 8
const maxSuccessfulFetches = 5
const maxQuietSearchQueries = 2
const maxQuietResultsPerQuery = 5
const maxQuietFetches = 2

const queryGenPrompt = `Today's date is %s. Generate %d focused web search queries for answering this question. Use the current year in queries when the question is about recent or current events. Return only the queries, one per line, no numbering or bullet points.

Question: %s`

type Researcher struct {
	ollama    *ollama.Client
	processor *document.Processor
	store     *vectorstore.Store
	cfg       *config.Config
}

type ProgressEvent struct {
	Step   string `json:"step"`
	Detail string `json:"detail"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
}

func NewResearcher(ollamaClient *ollama.Client, processor *document.Processor, store *vectorstore.Store, cfg *config.Config) *Researcher {
	return &Researcher{
		ollama:    ollamaClient,
		processor: processor,
		store:     store,
		cfg:       cfg,
	}
}

func (r *Researcher) Research(ctx context.Context, question, chatModel string, onProgress func(ProgressEvent) error) error {
	sendProgress := func(step, detail, status, url string) {
		if onProgress == nil {
			return
		}
		_ = onProgress(ProgressEvent{
			Step:   step,
			Detail: detail,
			Status: status,
			URL:    url,
		})
	}

	sendProgress("queries", "Generating search queries", "running", "")

	queries, err := r.generateQueries(ctx, question, chatModel)
	if err != nil {
		slog.Warn("failed to generate search queries, using original question", "error", err)
		queries = []string{question}
	}
	sendProgress("queries", fmt.Sprintf("Prepared %d search queries", len(queries)), "done", "")

	var allResults []Result
	seen := make(map[string]bool)

	for _, q := range queries {
		sendProgress("search", q, "running", "")
		results, err := Search(ctx, q, maxResultsPerQuery)
		if err != nil {
			slog.Warn("search failed", "query", q, "error", err)
			sendProgress("search", q, "failed", "")
			continue
		}
		sendProgress("search", fmt.Sprintf("%s (%d results)", q, len(results)), "done", "")
		for _, res := range results {
			if !seen[res.URL] {
				seen[res.URL] = true
				allResults = append(allResults, res)
			}
		}
	}

	if len(allResults) == 0 {
		sendProgress("search", "No search results found", "failed", "")
		return nil
	}

	sendProgress("fetch", fmt.Sprintf("Found %d unique results. Fetching top articles", len(allResults)), "running", "")

	indexed := make(map[string]bool)
	for _, d := range r.store.ListDocuments() {
		indexed[d.FilePath] = true
	}

	fetched := 0
	attempted := 0
	for _, res := range allResults {
		if fetched >= maxSuccessfulFetches {
			break
		}
		if indexed[res.URL] {
			slog.Debug("skipping already indexed URL", "url", res.URL)
			fetched++
			continue
		}

		attempted++
		title := res.Title
		if title == "" {
			title = res.URL
		}
		sendProgress("fetch", fmt.Sprintf("[%d/%d] %s", attempted, len(allResults), title), "running", res.URL)

		_, fetchedTitle, chunks, err := r.processor.ProcessURL(ctx, res.URL)
		if err != nil {
			slog.Warn("failed to fetch research URL", "url", res.URL, "error", err)
			sendProgress("fetch", title, "failed", res.URL)
			continue
		}

		slog.Info("research: indexed article", "url", res.URL, "title", fetchedTitle, "chunks", chunks)
		sendProgress("index", fmt.Sprintf("%s (%d chunks)", fetchedTitle, chunks), "done", res.URL)
		fetched++
	}

	if fetched > 0 {
		sendProgress("analyze", fmt.Sprintf("Indexed %d articles. Analyzing content", fetched), "done", "")
	} else {
		sendProgress("analyze", "All results were already indexed. Analyzing content", "done", "")
	}

	return nil
}

func (r *Researcher) GatherLiveContext(ctx context.Context, question, chatModel string) (int, error) {
	queries, err := r.generateQueries(ctx, question, chatModel)
	if err != nil {
		slog.Warn("quiet live context query generation failed, using original question", "error", err)
		queries = []string{question}
	}
	if len(queries) > maxQuietSearchQueries {
		queries = queries[:maxQuietSearchQueries]
	}

	var allResults []Result
	seen := make(map[string]bool)
	for _, q := range queries {
		results, err := Search(ctx, q, maxQuietResultsPerQuery)
		if err != nil {
			slog.Warn("quiet live context search failed", "query", q, "error", err)
			continue
		}
		for _, res := range results {
			if res.URL == "" || seen[res.URL] {
				continue
			}
			seen[res.URL] = true
			allResults = append(allResults, res)
		}
	}

	if len(allResults) == 0 {
		return 0, nil
	}

	indexed := make(map[string]bool)
	for _, d := range r.store.ListDocuments() {
		indexed[d.FilePath] = true
	}

	fetched := 0
	for _, res := range allResults {
		if fetched >= maxQuietFetches {
			break
		}
		if indexed[res.URL] {
			fetched++
			continue
		}
		_, title, chunks, err := r.processor.ProcessURL(ctx, res.URL)
		if err != nil {
			slog.Warn("quiet live context fetch failed", "url", res.URL, "error", err)
			continue
		}
		slog.Info("quiet live context indexed page", "url", res.URL, "title", title, "chunks", chunks)
		fetched++
	}
	return fetched, nil
}

func (r *Researcher) generateQueries(ctx context.Context, question, chatModel string) ([]string, error) {
	today := time.Now().Format("January 2, 2006")
	prompt := fmt.Sprintf(queryGenPrompt, today, maxSearchQueries, question)
	messages := []ollamaapi.Message{
		{Role: "user", Content: prompt},
	}

	resp, err := r.ollama.ChatOnceWithModel(ctx, chatModel, messages)
	if err != nil {
		return nil, err
	}

	var queries []string
	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "0123456789.-) ")
		line = strings.TrimSpace(line)
		if line != "" {
			queries = append(queries, line)
		}
	}

	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries generated")
	}
	if len(queries) > maxSearchQueries {
		queries = queries[:maxSearchQueries]
	}

	return queries, nil
}
