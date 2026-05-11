package rag

import (
	"context"
	"fmt"
	"strings"

	ollamaapi "github.com/ollama/ollama/api"

	"go-chatbot/internal/config"
	"go-chatbot/internal/gemma"
	"go-chatbot/internal/ollama"
	"go-chatbot/internal/vectorstore"
	"go-chatbot/internal/workbench"
)

type Chain struct {
	ollama *ollama.Client
	store  *vectorstore.Store
	cfg    *config.Config
}

func NewChain(ollamaClient *ollama.Client, store *vectorstore.Store, cfg *config.Config) *Chain {
	return &Chain{
		ollama: ollamaClient,
		store:  store,
		cfg:    cfg,
	}
}

type QueryOptions struct {
	ResearchMode     bool
	Locale           string
	Timezone         string
	ChatModel        string
	ToolContext      string
	FocusDocumentIDs []string
	FocusFiles       []string
	ReplyTo          *ReplyContext
}

type ReplyContext struct {
	Role    string
	Content string
}

type QueryResult struct {
	Sources []map[string]string
}

func (c *Chain) Query(ctx context.Context, question string, history []ollamaapi.Message, onToken func(string) error, opts *QueryOptions) (*QueryResult, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}

	var sources []map[string]string
	var systemContent string

	searchStore := c.searchStore()
	if searchStore.EntryCount() > 0 {
		embeddings, err := c.ollama.Embed(ctx, []string{question})
		if err != nil {
			return nil, fmt.Errorf("embed question: %w", err)
		}
		if len(embeddings) == 0 {
			return nil, fmt.Errorf("no embedding returned for question")
		}

		results := c.searchRelevantEntries(searchStore, embeddings[0], question, opts)
		results = filterUserVisibleResults(results)
		ragContext := buildContext(results)
		sources = buildSourceList(results)

		if opts.ResearchMode {
			systemContent = buildSystemPrompt(opts.ChatModel, gemma.ProfileResearch, fmt.Sprintf(researchPromptTemplate, ragContext))
		} else {
			systemContent = buildSystemPrompt(opts.ChatModel, gemma.ProfileChat, fmt.Sprintf(systemPromptTemplate, ragContext))
		}
	} else {
		systemContent = buildSystemPrompt(opts.ChatModel, gemma.ProfileChat, directChatPrompt)
	}

	if opts.Locale != "" || opts.Timezone != "" {
		localeCtx := buildLocaleContext(opts.Locale, opts.Timezone)
		systemContent += "\n\n" + localeCtx
	}
	if focusCtx := buildFocusContext(opts.FocusFiles); focusCtx != "" {
		systemContent += "\n\n" + focusCtx
	}
	if replyCtx := buildReplyContext(opts.ReplyTo); replyCtx != "" {
		systemContent += "\n\n" + replyCtx
	}
	if repoCtx := c.repoContext(question); repoCtx != "" {
		systemContent += "\n\n" + repoCtx
	}
	if toolCtx := buildToolContext(opts.ToolContext); toolCtx != "" {
		systemContent += "\n\n" + toolCtx
	}

	messages := make([]ollamaapi.Message, 0, len(history)+2)
	messages = append(messages, ollamaapi.Message{
		Role:    "system",
		Content: systemContent,
	})
	messages = append(messages, history...)
	messages = append(messages, ollamaapi.Message{
		Role:    "user",
		Content: question,
	})

	tokenHandler := onToken
	if gemma.IsGemma4(opts.ChatModel) {
		filter := gemma.NewThinkingFilter(onToken)
		tokenHandler = filter.Write
		defer func() {
			_ = filter.Flush()
		}()
	}

	if err := c.ollama.ChatStreamWithModel(ctx, opts.ChatModel, messages, tokenHandler); err != nil {
		return nil, fmt.Errorf("chat stream: %w", err)
	}

	return &QueryResult{Sources: sources}, nil
}

func filterUserVisibleResults(results []vectorstore.SearchResult) []vectorstore.SearchResult {
	filtered := make([]vectorstore.SearchResult, 0, len(results))
	for _, result := range results {
		if isInternalStateSource(result.Metadata["source"]) {
			continue
		}
		filtered = append(filtered, result)
	}
	return filtered
}

func isInternalStateSource(source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		return false
	}
	names := []string{
		"chat_sessions.json",
		"projects.json",
		"repo_map.json",
		"task_state.json",
		"watched_folders.json",
	}
	for _, name := range names {
		if strings.HasSuffix(source, "/"+name) || strings.HasSuffix(source, "\\"+name) || source == name {
			return true
		}
	}
	return false
}

func (c *Chain) searchRelevantEntries(store *vectorstore.Store, query []float32, question string, opts *QueryOptions) []vectorstore.SearchResult {
	if len(opts.FocusDocumentIDs) == 0 {
		return store.HybridSearch(query, question, c.cfg.TopK)
	}

	results := store.HybridSearchByDocumentIDs(query, question, c.cfg.TopK, opts.FocusDocumentIDs)
	if len(results) > 0 {
		return results
	}
	return store.HybridSearch(query, question, c.cfg.TopK)
}

func (c *Chain) searchStore() *vectorstore.Store {
	store, ok := c.activeProjectStore()
	if ok && store.EntryCount() > 0 {
		return store
	}
	return c.store
}

func (c *Chain) activeProjectStore() (*vectorstore.Store, bool) {
	state, err := workbench.LoadProjectState(c.cfg.DataDir)
	if err != nil {
		return nil, false
	}
	project, ok := workbench.ActiveProject(state)
	if !ok {
		return nil, false
	}
	store, err := vectorstore.LoadFromDisk(workbench.ResolveProjectVectorStoreDir(c.cfg.DataDir, project))
	if err != nil || store == nil {
		return nil, false
	}
	return store, true
}

func buildFocusContext(files []string) string {
	files = compactStrings(files, 12)
	if len(files) == 0 {
		return ""
	}
	return "The user explicitly focused these indexed files for this turn: " +
		strings.Join(files, ", ") +
		". Prioritize retrieved evidence from those files when it is relevant."
}

func buildReplyContext(reply *ReplyContext) string {
	if reply == nil {
		return ""
	}
	content := strings.TrimSpace(reply.Content)
	if content == "" {
		return ""
	}
	role := strings.TrimSpace(reply.Role)
	if role == "" {
		role = "message"
	}
	if len(content) > 1200 {
		content = content[:1200] + "..."
	}
	return fmt.Sprintf(
		"The user is replying to this previous %s message:\n\n%s\n\nUse it as local conversation context for the next answer.",
		role,
		content,
	)
}

func compactStrings(values []string, limit int) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func (c *Chain) ListModels(ctx context.Context) ([]string, error) {
	return c.ollama.ListModels(ctx)
}

func (c *Chain) HasModel(ctx context.Context, model string) bool {
	return c.ollama.HasModel(ctx, model)
}

func (c *Chain) DefaultChatModel() string {
	return c.cfg.ChatModel
}

func (c *Chain) repoContext(question string) string {
	repoMap, err := workbench.LoadRepoMap(c.cfg.DataDir)
	if err != nil {
		return ""
	}
	return buildRepoContext(repoMap, question)
}
