package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"go-chatbot/internal/app"
)

// Version is set at build time via ldflags: -X main.Version=v1.2.3
var Version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("jarvis server starting", "version", Version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	runtime, err := app.NewRuntime(ctx, Version)
	if err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
	if err := runtime.Server.Start(ctx); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
