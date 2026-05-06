package rag

import (
	"context"
	"fmt"

	ollamaapi "github.com/ollama/ollama/api"

	"go-chatbot/internal/config"
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
	ResearchMode bool
	Locale       string
	Timezone     string
	ChatModel    string
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

	if c.store.EntryCount() > 0 {
		embeddings, err := c.ollama.Embed(ctx, []string{question})
		if err != nil {
			return nil, fmt.Errorf("embed question: %w", err)
		}
		if len(embeddings) == 0 {
			return nil, fmt.Errorf("no embedding returned for question")
		}

		results := c.store.HybridSearch(embeddings[0], question, c.cfg.TopK)
		ragContext := buildContext(results)
		sources = buildSourceList(results)

		if opts.ResearchMode {
			systemContent = fmt.Sprintf(researchPromptTemplate, ragContext)
		} else {
			systemContent = fmt.Sprintf(systemPromptTemplate, ragContext)
		}
	} else {
		systemContent = directChatPrompt
	}

	if opts.Locale != "" || opts.Timezone != "" {
		localeCtx := buildLocaleContext(opts.Locale, opts.Timezone)
		systemContent += "\n\n" + localeCtx
	}
	if repoCtx := c.repoContext(question); repoCtx != "" {
		systemContent += "\n\n" + repoCtx
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

	if err := c.ollama.ChatStreamWithModel(ctx, opts.ChatModel, messages, onToken); err != nil {
		return nil, fmt.Errorf("chat stream: %w", err)
	}

	return &QueryResult{Sources: sources}, nil
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
