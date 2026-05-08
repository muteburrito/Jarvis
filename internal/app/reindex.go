package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go-chatbot/internal/config"
	"go-chatbot/internal/document"
	"go-chatbot/internal/workbench"
)

const folderReindexInterval = 10 * time.Minute

func startFolderReindexer(
	ctx context.Context,
	processor *document.Processor,
	cfg *config.Config,
	tasks *workbench.TaskStore,
) {
	if cfg == nil {
		return
	}
	if _, err := workbench.LoadWatchedFolders(cfg.DataDir); err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("could not load watched folders", "error", err)
		}
		return
	}

	go func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				reindexWatchedFolders(ctx, processor, cfg, tasks)
				timer.Reset(folderReindexInterval)
			}
		}
	}()
}

func reindexWatchedFolders(
	ctx context.Context,
	processor *document.Processor,
	cfg *config.Config,
	tasks *workbench.TaskStore,
) {
	folders, err := workbench.LoadWatchedFolders(cfg.DataDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("could not load watched folders", "error", err)
		}
		return
	}

	changed := false
	for i, folder := range folders {
		select {
		case <-ctx.Done():
			return
		default:
		}

		info, err := os.Stat(folder.Path)
		if err != nil || !info.IsDir() {
			slog.Warn("watched folder unavailable", "path", folder.Path, "error", err)
			continue
		}

		processed, chunks, err := processor.ProcessDirectory(ctx, folder.Path)
		if err != nil {
			slog.Warn("failed to re-index watched folder", "path", folder.Path, "error", err)
			continue
		}

		repoMap, err := workbench.ScanRepository(folder.Path)
		if err != nil {
			slog.Warn("failed to refresh workspace map", "path", folder.Path, "error", err)
		} else if err := workbench.SaveRepoMap(cfg.DataDir, repoMap); err != nil {
			slog.Warn("failed to save workspace map", "path", folder.Path, "error", err)
		}

		folders[i].LastIndexedAt = time.Now()
		changed = true
		if processed > 0 {
			recordFolderReindexTrace(tasks, folder.Path, processed, chunks)
		}
	}

	if changed {
		if err := workbench.SaveWatchedFolders(cfg.DataDir, folders); err != nil {
			slog.Warn("failed to save watched folders", "error", err)
		}
	}
}

func recordFolderReindexTrace(tasks *workbench.TaskStore, path string, processed int, chunks int) {
	if tasks == nil {
		return
	}
	tasks.AddTrace(workbench.TraceEvent{
		Type:    "folder_reindex",
		Summary: "Re-indexed watched folder",
		Metadata: map[string]string{
			"path":   path,
			"files":  fmt.Sprintf("%d", processed),
			"chunks": fmt.Sprintf("%d", chunks),
		},
	})
}
