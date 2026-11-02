package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"go-chatbot/internal/app"
	"go-chatbot/web"
)

// Version is set at build time via ldflags: -X main.Version=v1.2.3
var Version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("jarvis desktop starting (wails v3)", "version", Version)

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

	desktopApp := application.New(application.Options{
		Name:        runtime.Config.AppName,
		Description: "Private on-device AI assistant",
		Assets: application.AssetOptions{
			Handler: desktopAssets{
				assets:  web.Content,
				apiBase: apiURL,
			},
		},
		OnShutdown: func() {
			shutdownAPI()
			cancel()
		},
	})

	mainWindow := desktopApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            runtime.Config.AppName,
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        640,
		BackgroundColour: application.NewRGB(15, 23, 42),
	})

	quickWindow := desktopApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "quick-ask",
		Title:            runtime.Config.AppName + " Quick Ask",
		Width:            560,
		Height:           640,
		MinWidth:         420,
		MinHeight:        480,
		Hidden:           true,
		AlwaysOnTop:      true,
		HideOnFocusLost:  true,
		HideOnEscape:     true,
		BackgroundColour: application.NewRGB(15, 23, 42),
	})

	setupSystemTray(desktopApp, runtime.Config.AppName, mainWindow, quickWindow)

	if err := desktopApp.Run(); err != nil {
		slog.Error("desktop error", "error", err)
		os.Exit(1)
	}
}

func setupSystemTray(desktopApp *application.App, appName string, mainWindow, quickWindow application.Window) {
	tray := desktopApp.SystemTray.New()
	tray.SetTooltip(appName + " - On-Device AI")

	menu := application.NewMenu()
	menu.Add("Open " + appName).OnClick(func(*application.Context) {
		mainWindow.Show()
		mainWindow.Focus()
	})
	menu.Add("Quick Ask (Always on Top)").OnClick(func(*application.Context) {
		quickWindow.Show()
		quickWindow.Focus()
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(*application.Context) {
		desktopApp.Quit()
	})

	tray.SetMenu(menu)
	tray.OnClick(func() {
		mainWindow.Show()
		mainWindow.Focus()
	})
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
		_ = server.Shutdown(shutdownCtx)
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
		_ = server.Shutdown(shutdownCtx)
	}, nil
}
