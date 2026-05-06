package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"go-chatbot/internal/app"
	"go-chatbot/web"
)

// Version is set at build time via ldflags: -X main.Version=v1.2.3
var Version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("jarvis desktop starting", "version", Version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runtime, err := app.NewRuntime(ctx, Version)
	if err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}

	apiURL, shutdownAPI, err := startLoopbackAPI(ctx, runtime.Server)
	if err != nil {
		slog.Error("api startup failed", "error", err)
		os.Exit(1)
	}
	defer shutdownAPI()

	err = wails.Run(&options.App{
		Title:     runtime.Config.AppName,
		Width:     1280,
		Height:    820,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Handler: desktopAssets{
				assets:  web.Content,
				apiBase: apiURL,
			},
		},
		OnShutdown: func(context.Context) {
			shutdownAPI()
			cancel()
		},
		BackgroundColour: options.NewRGB(15, 23, 42),
	})
	if err != nil {
		slog.Error("desktop error", "error", err)
		os.Exit(1)
	}
}

func startLoopbackAPI(ctx context.Context, handler http.Handler) (string, func(), error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("desktop api server failed", "error", err)
		}
	}()
	url := "http://" + listener.Addr().String()
	slog.Info("desktop api server starting", "url", url)
	return url, func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}, nil
}
