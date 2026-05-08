package document

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"go-chatbot/internal/config"
	"go-chatbot/internal/ollama"
	"go-chatbot/internal/vectorstore"
)

var ignoredDirs = map[string]bool{
	"bin":          true,
	"obj":          true,
	"node_modules": true,
	".git":         true,
	".vs":          true,
	".vscode":      true,
	".idea":        true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
	"__pycache__":  true,
	".next":        true,
	".nuget":       true,
	"packages":     true,
	"TestResults":  true,
	"Debug":        true,
	"Release":      true,
}

var ignoredStateFiles = map[string]bool{
	"chat_sessions.json":   true,
	"projects.json":        true,
	"repo_map.json":        true,
	"task_state.json":      true,
	"watched_folders.json": true,
}

type Processor struct {
	registry    *Registry
	chunker     *Chunker
	ollama      *ollama.Client
	store       *vectorstore.Store
	cfg         *config.Config
	visionReady bool
}

func NewProcessor(registry *Registry, chunker *Chunker, ollamaClient *ollama.Client, store *vectorstore.Store, cfg *config.Config, visionReady bool) *Processor {
	return &Processor{
		registry:    registry,
		chunker:     chunker,
		ollama:      ollamaClient,
		store:       store,
		cfg:         cfg,
		visionReady: visionReady,
	}
}

func (p *Processor) ProcessFile(ctx context.Context, filePath string) (string, int, error) {
	docs, err := p.registry.Load(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("load: %w", err)
	}

	docs = p.describeImages(ctx, docs)

	info, _ := os.Stat(filePath)
	var fileSize int64
	var modifiedAt time.Time
	if info != nil {
		fileSize = info.Size()
		modifiedAt = info.ModTime()
	}

	return p.processDocs(ctx, docs, fileBaseName(filePath), filePath, fileSize, modifiedAt)
}

func (p *Processor) ProcessURL(ctx context.Context, url string) (string, string, int, error) {
	docs, err := FetchURL(ctx, url)
	if err != nil {
		return "", "", 0, fmt.Errorf("fetch: %w", err)
	}

	title := url
	if len(docs) > 0 {
		if t, ok := docs[0].Metadata["title"]; ok && t != "" {
			title = t
		}
	}

	docID, chunks, err := p.processDocs(ctx, docs, title, url, 0, time.Time{})
	return docID, title, chunks, err
}

func (p *Processor) processDocs(
	ctx context.Context,
	docs []Document,
	filename string,
	sourcePath string,
	fileSize int64,
	modifiedAt time.Time,
) (string, int, error) {
	var validDocs []Document
	for _, doc := range docs {
		if strings.TrimSpace(doc.Content) != "" {
			validDocs = append(validDocs, doc)
		}
	}
	if len(validDocs) == 0 {
		return "", 0, fmt.Errorf("no content produced from %s", sourcePath)
	}

	docID := "doc_" + uuid.New().String()[:8]
	chunks := p.chunker.SplitDocuments(validDocs, docID)
	if len(chunks) == 0 {
		return "", 0, fmt.Errorf("no chunks produced from %s", sourcePath)
	}

	slog.Info("processing content", "source", sourcePath, "chunks", len(chunks))

	const batchSize = 32
	var entries []vectorstore.Entry

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[i:end]

		texts := make([]string, len(batch))
		for j, c := range batch {
			texts[j] = c.Content
		}

		embeddings, err := p.ollama.Embed(ctx, texts)
		if err != nil {
			return "", 0, fmt.Errorf("embed batch %d: %w", i/batchSize, err)
		}

		for j, c := range batch {
			entries = append(entries, vectorstore.Entry{
				ID:         fmt.Sprintf("%s_chunk_%d", docID, c.Index),
				DocumentID: docID,
				Content:    c.Content,
				Metadata:   c.Metadata,
				Embedding:  embeddings[j],
			})
		}
	}

	p.store.Add(entries)

	p.store.AddDocument(vectorstore.DocumentInfo{
		ID:         docID,
		Filename:   filename,
		FilePath:   sourcePath,
		Size:       fileSize,
		ChunkCount: len(chunks),
		UploadedAt: time.Now(),
		ModifiedAt: modifiedAt,
	})

	if err := p.store.Save(p.cfg.VectorStoreDir); err != nil {
		slog.Warn("failed to persist vectorstore", "error", err)
	}

	return docID, len(chunks), nil
}

func (p *Processor) ProcessDirectory(ctx context.Context, dir string) (int, int, error) {
	knownFiles := make(map[string]vectorstore.DocumentInfo)
	for _, d := range p.store.ListDocuments() {
		key, ok := canonicalDocumentPath(d.FilePath)
		if ok {
			knownFiles[key] = d
		}
	}

	var processed, totalChunks int

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if ignoredDirs[d.Name()] || ignoredDirs[name] || strings.HasPrefix(d.Name(), ".") && d.Name() != "." || p.isAppStoragePath(path) {
				slog.Debug("skipping ignored directory", "dir", path)
				return filepath.SkipDir
			}
			return nil
		}
		if ignoredStateFiles[strings.ToLower(d.Name())] || p.isAppStoragePath(path) {
			return nil
		}
		if !p.registry.CanLoad(path) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			slog.Warn("failed to stat file", "file", path, "error", err)
			return nil
		}

		key, ok := canonicalDocumentPath(path)
		if ok {
			if known, exists := knownFiles[key]; exists && sameSourceFile(info, known) {
				slog.Debug("skipping unchanged indexed file", "file", path)
				return nil
			}
			if known, exists := knownFiles[key]; exists {
				slog.Info("re-indexing changed file", "file", path)
				if err := p.RemoveDocument(ctx, known.ID); err != nil {
					slog.Warn("failed to remove changed indexed file", "file", path, "error", err)
					return nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, chunks, err := p.ProcessFile(ctx, path)
		if err != nil {
			slog.Warn("failed to process file", "file", path, "error", err)
			return nil
		}
		processed++
		totalChunks += chunks
		return nil
	})
	if err != nil {
		return processed, totalChunks, fmt.Errorf("walk dir: %w", err)
	}

	return processed, totalChunks, nil
}

func canonicalDocumentPath(path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "", false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	return strings.ToLower(filepath.Clean(absPath)), true
}

func (p *Processor) isAppStoragePath(path string) bool {
	return p.sameOrInsideConfiguredDir(path, p.cfg.DataDir) || p.sameOrInsideConfiguredDir(path, p.cfg.VectorStoreDir)
}

func (p *Processor) sameOrInsideConfiguredDir(path string, dir string) bool {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(dir) == "" {
		return false
	}
	pathAbs, pathErr := filepath.Abs(path)
	dirAbs, dirErr := filepath.Abs(dir)
	if pathErr != nil || dirErr != nil {
		return false
	}
	pathAbs = filepath.Clean(pathAbs)
	dirAbs = filepath.Clean(dirAbs)
	if strings.EqualFold(pathAbs, dirAbs) {
		return true
	}
	rel, err := filepath.Rel(dirAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func sameSourceFile(info os.FileInfo, doc vectorstore.DocumentInfo) bool {
	if info == nil {
		return false
	}
	if info.Size() != doc.Size {
		return false
	}
	return !doc.ModifiedAt.IsZero() && info.ModTime().Equal(doc.ModifiedAt)
}

func (p *Processor) RemoveDocument(ctx context.Context, docID string) error {
	for _, doc := range p.store.ListDocuments() {
		if doc.ID == docID {
			p.removeOwnedDataFile(doc.FilePath)
			break
		}
	}
	p.store.RemoveByDocumentID(docID)
	p.store.RemoveDocument(docID)
	return p.store.Save(p.cfg.VectorStoreDir)
}

func (p *Processor) ClearAll(ctx context.Context) error {
	for _, doc := range p.store.ListDocuments() {
		p.removeOwnedDataFile(doc.FilePath)
	}
	p.store.Clear()
	return p.store.Save(p.cfg.VectorStoreDir)
}

func (p *Processor) removeOwnedDataFile(path string) {
	if !p.isOwnedDataFile(path) {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to remove indexed data file", "path", path, "error", err)
	}
}

func (p *Processor) isOwnedDataFile(path string) bool {
	if strings.TrimSpace(path) == "" || p.cfg == nil || strings.TrimSpace(p.cfg.DataDir) == "" {
		return false
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDataDir, err := filepath.Abs(p.cfg.DataDir)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absDataDir, absPath)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}

	info, err := os.Stat(absPath)
	return err == nil && !info.IsDir()
}

func (p *Processor) describeImages(ctx context.Context, docs []Document) []Document {
	if !p.visionReady {
		var filtered []Document
		for _, doc := range docs {
			if len(doc.ImageData) == 0 {
				filtered = append(filtered, doc)
			}
		}
		if len(filtered) < len(docs) {
			slog.Warn("skipping images, vision model not available")
		}
		return filtered
	}

	for i := range docs {
		if len(docs[i].ImageData) == 0 {
			continue
		}
		source := docs[i].Metadata["source"]
		slog.Info("describing image with vision model", "source", source)

		description, err := p.ollama.DescribeImage(ctx, docs[i].ImageData)
		if err != nil {
			slog.Warn("failed to describe image", "source", source, "error", err)
			continue
		}
		docs[i].Content = description
		docs[i].ImageData = nil
	}
	return docs
}

func (p *Processor) CanLoad(filePath string) bool {
	return p.registry.CanLoad(filePath)
}

func fileBaseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
