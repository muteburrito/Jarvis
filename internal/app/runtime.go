package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go-chatbot/internal/config"
	"go-chatbot/internal/document"
	"go-chatbot/internal/ollama"
	"go-chatbot/internal/rag"
	"go-chatbot/internal/server"
	"go-chatbot/internal/system"
	"go-chatbot/internal/updater"
	"go-chatbot/internal/vectorstore"
	"go-chatbot/internal/websearch"
	"go-chatbot/internal/workbench"
	"go-chatbot/web"
)

type Runtime struct {
	Config *config.Config
	Server *server.Server
}

func NewRuntime(ctx context.Context, version string) (*Runtime, error) {
	cfg := config.Load()
	applyAdaptiveChatModel(cfg)

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	ollamaClient, err := ollama.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to Ollama at %s: %w", cfg.OllamaURL, err)
	}
	slog.Info("connected to Ollama", "url", cfg.OllamaURL)

	if err := ensureRequiredModels(ctx, ollamaClient, cfg); err != nil {
		return nil, err
	}

	dim, err := ollamaClient.EmbeddingDimension(ctx)
	if err != nil {
		return nil, fmt.Errorf("embedding model %q returned invalid response: %w", cfg.EmbeddingModel, err)
	}
	slog.Info("embedding model ready", "model", cfg.EmbeddingModel, "dimension", dim)

	store, err := vectorstore.LoadFromDisk(cfg.VectorStoreDir)
	if err != nil {
		slog.Warn("could not load vectorstore from disk, starting fresh", "error", err)
	}
	if store == nil {
		store = vectorstore.New(dim)
		slog.Info("initialized empty vectorstore")
	} else {
		slog.Info("loaded vectorstore from disk",
			"entries", store.EntryCount(),
			"documents", store.DocumentCount())
	}

	visionReady := ollamaClient.HasModel(ctx, cfg.VisionModel)
	if visionReady {
		slog.Info("vision model available, image support enabled", "model", cfg.VisionModel)
	} else {
		slog.Warn("vision model is not installed, image support disabled", "model", cfg.VisionModel)
	}

	processor := newProcessor(ollamaClient, store, cfg, visionReady)
	chain := rag.NewChain(ollamaClient, store, cfg)
	autoIndexDataDir(ctx, processor, cfg.DataDir)

	researcher := websearch.NewResearcher(ollamaClient, processor, store, cfg)
	slog.Info("web research mode available")

	taskStore, err := workbench.NewTaskStore(cfg.DataDir)
	if err != nil {
		slog.Warn("could not initialize task store", "error", err)
	}
	startFolderReindexer(ctx, processor, cfg, taskStore)

	chatStore, err := workbench.NewChatStore(cfg.DataDir)
	if err != nil {
		slog.Warn("could not initialize chat store", "error", err)
	}

	upd := updater.New(version, cfg.GitHubRepo, cfg.GitHubToken)
	upd.Start(ctx)

	srv := server.New(server.Dependencies{
		Config:    cfg,
		Processor: processor,
		Chain:     chain,
		Store:     store,
		Research:  researcher,
		Tasks:     taskStore,
		Chats:     chatStore,
		Updater:   upd,
		WebFS:     web.Content,
	})

	return &Runtime{Config: cfg, Server: srv}, nil
}

func ensureRequiredModels(ctx context.Context, client *ollama.Client, cfg *config.Config) error {
	var missing []string
	if !client.HasModel(ctx, cfg.ChatModel) {
		missing = append(missing, cfg.ChatModel)
	}
	if !client.HasModel(ctx, cfg.EmbeddingModel) {
		missing = append(missing, cfg.EmbeddingModel)
	}
	for _, model := range missing {
		if err := pullModel(ctx, client, model); err != nil {
			return fmt.Errorf("download required model %q: %w", model, err)
		}
	}
	return nil
}

func newProcessor(
	client *ollama.Client,
	store *vectorstore.Store,
	cfg *config.Config,
	visionReady bool,
) *document.Processor {
	registry := document.NewRegistry()
	registry.Register(&document.PDFLoader{})
	registry.Register(&document.DOCXLoader{})
	registry.Register(&document.XLSXLoader{})
	registry.Register(&document.PPTXLoader{})
	registry.Register(&document.ImageLoader{})
	registry.Register(&document.TextLoader{})

	chunker := document.NewChunker(cfg.ChunkSize, cfg.ChunkOverlap)
	return document.NewProcessor(registry, chunker, client, store, cfg, visionReady)
}

func autoIndexDataDir(ctx context.Context, processor *document.Processor, dataDir string) {
	info, err := os.Stat(dataDir)
	if err != nil || !info.IsDir() {
		return
	}
	files, _ := os.ReadDir(dataDir)
	if len(files) == 0 {
		return
	}
	slog.Info("scanning data directory for files to index", "path", dataDir)
	processed, chunks, err := processor.ProcessDirectory(ctx, dataDir)
	if err != nil {
		slog.Warn("error scanning data directory", "error", err)
		return
	}
	if processed > 0 {
		slog.Info("auto-indexed files from data directory", "files", processed, "chunks", chunks)
	}
}

func applyAdaptiveChatModel(cfg *config.Config) {
	if os.Getenv("CHAT_MODEL") != "" {
		slog.Info("using chat model from CHAT_MODEL", "model", cfg.ChatModel)
		return
	}

	hw := system.DetectHardware()
	cfg.ChatModel = selectChatModel(hw)
	slog.Info("selected adaptive chat model",
		"model", cfg.ChatModel,
		"total_ram_mb", hw.TotalRAMMB,
		"gpu_total_mb", hw.GPUTotalMB)
}

func selectChatModel(hw system.HardwareInfo) string {
	const tinyModel = "gemma4:e2b"
	const lightModel = "gemma4:e4b"
	if hw.GPUDetected && hw.GPUTotalMB < 8000 {
		return tinyModel
	}
	if !hw.GPUDetected && hw.TotalRAMMB > 0 && hw.TotalRAMMB < 12000 {
		return tinyModel
	}
	return lightModel
}

func pullModel(ctx context.Context, client *ollama.Client, model string) error {
	slog.Info("downloading required Ollama model", "model", model)
	if err := client.PullModel(ctx, model, func(string, int64, int64) {}); err != nil {
		return err
	}
	if !client.HasModel(ctx, model) {
		return fmt.Errorf("model was not available after pull")
	}
	return nil
}
