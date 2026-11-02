package ollama

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"

	"go-chatbot/internal/config"
	"go-chatbot/internal/gemma"
	"go-chatbot/internal/llamacpp"
)

type Client struct {
	api       *api.Client
	llama     *llamacpp.Engine
	backend   string
	cfg       *config.Config
	dimOnce   sync.Once
	dimension int
	statusMu  sync.RWMutex
	status    llamacpp.Status
}

func New(cfg *config.Config) (*Client, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.LLMBackend))
	if mode == "" {
		mode = "auto"
	}

	c := &Client{
		cfg:   cfg,
		llama: llamacpp.NewEngine(cfg),
	}

	if mode == "llamacpp" || mode == "direct" || mode == "gguf" {
		c.backend = "llamacpp"
		return c, nil
	}

	u, err := url.Parse(cfg.OllamaURL)
	if err != nil {
		if mode == "ollama" {
			return nil, fmt.Errorf("invalid ollama URL: %w", err)
		}
		c.backend = "llamacpp"
		return c, nil
	}

	httpClient := &http.Client{Timeout: 2 * time.Second}
	probeClient := api.NewClient(u, httpClient)
	if err := probeClient.Heartbeat(context.Background()); err != nil {
		if mode == "ollama" {
			return nil, fmt.Errorf("cannot reach ollama at %s: %w", cfg.OllamaURL, err)
		}
		slog.Info("Ollama not running, switching to built-in on-device GGUF engine (llama.cpp)")
		c.backend = "llamacpp"
		return c, nil
	}

	c.api = api.NewClient(u, http.DefaultClient)
	c.backend = "ollama"
	c.status = llamacpp.Status{
		Backend:        "ollama",
		State:          "ready",
		ActiveTask:     "Connected to local Ollama runtime",
		ProgressPct:    100,
		Ready:          true,
		ChatModel:      cfg.ChatModel,
		EmbeddingModel: cfg.EmbeddingModel,
	}
	return c, nil
}

func (c *Client) Backend() string {
	return c.backend
}

func (c *Client) EngineStatus() llamacpp.Status {
	if c.backend == "llamacpp" {
		return c.llama.GetStatus()
	}
	c.statusMu.RLock()
	defer c.statusMu.RUnlock()
	return c.status
}

func (c *Client) setOllamaProgress(state, task string, completed, total int64, ready bool) {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	pct := 0
	if total > 0 && completed >= 0 {
		pct = int(completed * 100 / total)
		if pct > 100 {
			pct = 100
		}
	} else if ready {
		pct = 100
	}
	c.status = llamacpp.Status{
		Backend:        "ollama",
		State:          state,
		ActiveTask:     task,
		ProgressPct:    pct,
		CompletedBytes: completed,
		TotalBytes:     total,
		Ready:          ready,
		ChatModel:      c.cfg.ChatModel,
		EmbeddingModel: c.cfg.EmbeddingModel,
	}
}

func (c *Client) WarmupAsync(ctx context.Context) {
	go func() {
		if c.backend == "llamacpp" {
			c.llama.Warmup(ctx)
			return
		}
		for _, model := range []string{c.cfg.EmbeddingModel, c.cfg.ChatModel} {
			if c.HasModel(ctx, model) {
				continue
			}
			c.setOllamaProgress("downloading", "Downloading "+model, 0, 0, false)
			_ = c.PullModel(ctx, model, func(status string, completed, total int64) {
				c.setOllamaProgress("downloading", "Downloading "+model+": "+status, completed, total, false)
			})
		}
		c.setOllamaProgress("ready", "Connected to local Ollama runtime", 0, 0, true)
	}()
}

func (c *Client) Close() {
	if c.llama != nil {
		c.llama.Shutdown()
	}
}

func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	if c.backend == "llamacpp" {
		return c.llama.ListModels(), nil
	}
	resp, err := c.api.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	var names []string
	for _, m := range resp.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

func (c *Client) HasModel(ctx context.Context, model string) bool {
	if c.backend == "llamacpp" {
		return c.llama.HasModel(model)
	}
	models, err := c.ListModels(ctx)
	if err != nil {
		return false
	}
	for _, m := range models {
		if m == model || strings.HasPrefix(m, model+":") || strings.TrimSuffix(m, ":latest") == model {
			return true
		}
	}
	return false
}

func (c *Client) PullModel(ctx context.Context, model string, onProgress func(status string, completed, total int64)) error {
	if c.backend == "llamacpp" {
		return c.llama.PullModel(ctx, model, onProgress)
	}
	stream := true
	return c.api.Pull(ctx, &api.PullRequest{
		Model:  model,
		Stream: &stream,
	}, func(resp api.ProgressResponse) error {
		c.setOllamaProgress("downloading", "Downloading "+model+": "+resp.Status, resp.Completed, resp.Total, false)
		if onProgress != nil {
			onProgress(resp.Status, resp.Completed, resp.Total)
		}
		return nil
	})
}

func (c *Client) keepAlive() *api.Duration {
	return &api.Duration{Duration: c.cfg.OllamaKeepAlive}
}

func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if c.backend == "llamacpp" {
		return c.llama.Embed(ctx, texts)
	}
	resp, err := c.api.Embed(ctx, &api.EmbedRequest{
		Model:     c.cfg.EmbeddingModel,
		Input:     texts,
		KeepAlive: c.keepAlive(),
	})
	if err != nil {
		return llamacpp.LocalHashEmbed(texts, llamacpp.DefaultEmbeddingDimension), nil
	}

	results := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		f32 := make([]float32, len(emb))
		for j, v := range emb {
			f32[j] = float32(v)
		}
		results[i] = f32
	}
	return results, nil
}

func (c *Client) ChatStream(ctx context.Context, messages []api.Message, onToken func(string) error) error {
	return c.ChatStreamWithModel(ctx, c.cfg.ChatModel, messages, onToken)
}

func (c *Client) ChatStreamWithModel(ctx context.Context, model string, messages []api.Message, onToken func(string) error) error {
	if model == "" {
		model = c.cfg.ChatModel
	}
	if c.backend == "llamacpp" {
		return c.llama.ChatStream(ctx, model, messages, onToken)
	}
	stream := true
	req := &api.ChatRequest{
		Model:     model,
		Messages:  messages,
		Stream:    &stream,
		KeepAlive: c.keepAlive(),
		Options:   gemma.Options(model, false),
	}

	return c.api.Chat(ctx, req, func(resp api.ChatResponse) error {
		if resp.Message.Content != "" {
			if err := onToken(resp.Message.Content); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *Client) ChatOnce(ctx context.Context, messages []api.Message) (string, error) {
	return c.ChatOnceWithModel(ctx, c.cfg.ChatModel, messages)
}

func (c *Client) ChatOnceWithModel(ctx context.Context, model string, messages []api.Message) (string, error) {
	if model == "" {
		model = c.cfg.ChatModel
	}
	if c.backend == "llamacpp" {
		return c.llama.ChatOnce(ctx, model, messages)
	}
	stream := false
	req := &api.ChatRequest{
		Model:     model,
		Messages:  messages,
		Stream:    &stream,
		KeepAlive: c.keepAlive(),
		Options:   gemma.Options(model, true),
	}

	var result string
	err := c.api.Chat(ctx, req, func(resp api.ChatResponse) error {
		result = resp.Message.Content
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("chat: %w", err)
	}
	return result, nil
}

func (c *Client) DescribeImage(ctx context.Context, imageData []byte) (string, error) {
	if c.backend == "llamacpp" {
		return c.llama.DescribeImage(ctx, imageData)
	}
	var result strings.Builder
	stream := true
	req := &api.ChatRequest{
		Model: c.cfg.VisionModel,
		Messages: []api.Message{
			{
				Role:    "user",
				Content: "Describe this image in detail. Include any text, diagrams, charts, tables, code, or visual elements you see. Be thorough and specific so the description can be used for search and question answering later.",
				Images:  []api.ImageData{imageData},
			},
		},
		Stream:    &stream,
		KeepAlive: c.keepAlive(),
	}

	err := c.api.Chat(ctx, req, func(resp api.ChatResponse) error {
		result.WriteString(resp.Message.Content)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("describe image: %w", err)
	}

	return result.String(), nil
}

func (c *Client) EmbeddingDimension(ctx context.Context) (int, error) {
	if c.backend == "llamacpp" {
		return llamacpp.DefaultEmbeddingDimension, nil
	}
	var dimErr error
	c.dimOnce.Do(func() {
		embeddings, err := c.Embed(ctx, []string{"dimension test"})
		if err != nil {
			dimErr = err
			return
		}
		if len(embeddings) == 0 || len(embeddings[0]) == 0 {
			dimErr = fmt.Errorf("empty embedding returned")
			return
		}
		c.dimension = len(embeddings[0])
	})
	if dimErr != nil || c.dimension == 0 {
		return llamacpp.DefaultEmbeddingDimension, nil
	}
	return c.dimension, nil
}
