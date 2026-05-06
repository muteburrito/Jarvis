package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

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

// Version is set at build time via ldflags: -X main.Version=v1.2.3
var Version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("jarvis starting", "version", Version)

	cfg := config.Load()
	applyAdaptiveChatModel(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ollamaClient, err := ollama.New(cfg)
	if err != nil {
		slog.Error("cannot reach Ollama", "url", cfg.OllamaURL, "error", err)
		fmt.Println("\n[Jarvis] Ollama is not running or not installed.")
		fmt.Println("")
		fmt.Println("  To install Ollama, visit: https://ollama.com/download")
		fmt.Println("")
		fmt.Println("  After installing, start Ollama and run these commands:")
		fmt.Printf("    ollama pull %s\n", cfg.ChatModel)
		fmt.Printf("    ollama pull %s\n", cfg.EmbeddingModel)
		fmt.Println("")
		os.Exit(1)
	}
	slog.Info("connected to Ollama", "url", cfg.OllamaURL)

	var missing []string
	if !ollamaClient.HasModel(ctx, cfg.ChatModel) {
		missing = append(missing, cfg.ChatModel)
	}
	if !ollamaClient.HasModel(ctx, cfg.EmbeddingModel) {
		missing = append(missing, cfg.EmbeddingModel)
	}
	if len(missing) > 0 {
		fmt.Println("\n[Jarvis] Ollama is running, but some required models are not installed.")
		fmt.Println("  Jarvis will download them now. This can take a while on the first run.")
		fmt.Println("")
		for _, model := range missing {
			if err := pullModel(ctx, ollamaClient, model); err != nil {
				slog.Error("failed to pull required model", "model", model, "error", err)
				fmt.Printf("\n[Jarvis] Failed to download model '%s'.\n", model)
				fmt.Println("  You can retry manually with:")
				fmt.Printf("    ollama pull %s\n", model)
				fmt.Println("")
				os.Exit(1)
			}
		}
		fmt.Println("\n[Jarvis] Required models are ready.")
	}

	dim, err := ollamaClient.EmbeddingDimension(ctx)
	if err != nil {
		slog.Error("failed to determine embedding dimension", "error", err)
		fmt.Printf("\n[Jarvis] The embedding model '%s' did not return a valid response.\n", cfg.EmbeddingModel)
		fmt.Println("  Try pulling it again: ollama pull", cfg.EmbeddingModel)
		fmt.Println("")
		os.Exit(1)
	}
	slog.Info("embedding model ready", "model", cfg.EmbeddingModel, "dimension", dim)

	store, err := vectorstore.LoadFromDisk(cfg.VectorStoreDir)
	if err != nil {
		slog.Warn("could not load vectorstore from disk, starting fresh", "error", err)
		store = nil
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
		fmt.Printf("\n[Jarvis] Vision model '%s' is not installed. Image support is disabled.\n", cfg.VisionModel)
		fmt.Printf("  To enable image support: ollama pull %s\n\n", cfg.VisionModel)
	}

	registry := document.NewRegistry()
	registry.Register(&document.PDFLoader{})
	registry.Register(&document.DOCXLoader{})
	registry.Register(&document.XLSXLoader{})
	registry.Register(&document.PPTXLoader{})
	registry.Register(&document.ImageLoader{})
	registry.Register(&document.TextLoader{})

	chunker := document.NewChunker(cfg.ChunkSize, cfg.ChunkOverlap)
	processor := document.NewProcessor(registry, chunker, ollamaClient, store, cfg, visionReady)
	chain := rag.NewChain(ollamaClient, store, cfg)

	if info, err := os.Stat(cfg.DataDir); err == nil && info.IsDir() {
		files, _ := os.ReadDir(cfg.DataDir)
		if len(files) > 0 {
			slog.Info("scanning data directory for files to index", "path", cfg.DataDir)
			processed, chunks, err := processor.ProcessDirectory(ctx, cfg.DataDir)
			if err != nil {
				slog.Warn("error scanning data directory", "error", err)
			} else if processed > 0 {
				slog.Info("auto-indexed files from data directory", "files", processed, "chunks", chunks)
			}
		}
	}

	researcher := websearch.NewResearcher(ollamaClient, processor, store, cfg)
	slog.Info("web research mode available")

	taskStore, err := workbench.NewTaskStore(cfg.DataDir)
	if err != nil {
		slog.Warn("could not initialize task store", "error", err)
		taskStore = nil
	}
	chatStore, err := workbench.NewChatStore(cfg.DataDir)
	if err != nil {
		slog.Warn("could not initialize chat store", "error", err)
		chatStore = nil
	}

	upd := updater.New(Version, cfg.GitHubRepo, cfg.GitHubToken)
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
	if err := srv.Start(ctx); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
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
	fmt.Printf("  Downloading %s\n", model)
	lastPercent := int64(-1)
	lastStatus := ""
	err := client.PullModel(ctx, model, func(status string, completed, total int64) {
		if total > 0 {
			percent := completed * 100 / total
			if percent != lastPercent || status != lastStatus {
				fmt.Printf("    %s %d%%\n", status, percent)
				lastPercent = percent
				lastStatus = status
			}
			return
		}
		if status != "" && status != lastStatus {
			fmt.Printf("    %s\n", status)
			lastStatus = status
		}
	})
	if err != nil {
		return err
	}
	if !client.HasModel(ctx, model) {
		return fmt.Errorf("model was not available after pull")
	}
	return nil
}
