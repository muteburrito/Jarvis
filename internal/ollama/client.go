package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/ollama/ollama/api"

	"go-chatbot/internal/config"
)

type Client struct {
	api       *api.Client
	cfg       *config.Config
	dimOnce   sync.Once
	dimension int
}

func New(cfg *config.Config) (*Client, error) {
	u, err := url.Parse(cfg.OllamaURL)
	if err != nil {
		return nil, fmt.Errorf("invalid ollama URL: %w", err)
	}
	apiClient := api.NewClient(u, http.DefaultClient)

	c := &Client{api: apiClient, cfg: cfg}

	if err := apiClient.Heartbeat(context.Background()); err != nil {
		return nil, fmt.Errorf("cannot reach ollama at %s: %w", cfg.OllamaURL, err)
	}

	return c, nil
}

func (c *Client) ListModels(ctx context.Context) ([]string, error) {
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
	stream := true
	return c.api.Pull(ctx, &api.PullRequest{
		Model:  model,
		Stream: &stream,
	}, func(resp api.ProgressResponse) error {
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
	resp, err := c.api.Embed(ctx, &api.EmbedRequest{
		Model:     c.cfg.EmbeddingModel,
		Input:     texts,
		KeepAlive: c.keepAlive(),
	})
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
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
	stream := true
	req := &api.ChatRequest{
		Model:     model,
		Messages:  messages,
		Stream:    &stream,
		KeepAlive: c.keepAlive(),
		Options: map[string]interface{}{
			"temperature": 0.0,
		},
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
	stream := false
	req := &api.ChatRequest{
		Model:     model,
		Messages:  messages,
		Stream:    &stream,
		KeepAlive: c.keepAlive(),
		Options: map[string]interface{}{
			"temperature": 0.0,
		},
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
	return c.dimension, dimErr
}
