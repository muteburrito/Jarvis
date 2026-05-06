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
	"bin":            true,
	"obj":            true,
	"node_modules":   true,
	".git":           true,
	".vs":            true,
	".vscode":        true,
	".idea":          true,
	"vendor":         true,
	"dist":           true,
	"build":          true,
	"target":         true,
	"__pycache__":    true,
	".next":          true,
	".nuget":         true,
	"packages":       true,
	"TestResults":    true,
	"Debug":          true,
	"Release":        true,
}

type Processor struct {
	registry     *Registry
	chunker      *Chunker
	ollama       *ollama.Client
	store        *vectorstore.Store
	cfg          *config.Config
	visionReady  bool
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
	if info != nil {
		fileSize = info.Size()
	}

	return p.processDocs(ctx, docs, fileBaseName(filePath), filePath, fileSize)
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

	docID, chunks, err := p.processDocs(ctx, docs, title, url, 0)
	return docID, title, chunks, err
}

func (p *Processor) processDocs(ctx context.Context, docs []Document, filename, sourcePath string, fileSize int64) (string, int, error) {
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
	})

	if err := p.store.Save(p.cfg.VectorStoreDir); err != nil {
		slog.Warn("failed to persist vectorstore", "error", err)
	}

	return docID, len(chunks), nil
}

func (p *Processor) ProcessDirectory(ctx context.Context, dir string) (int, int, error) {
	knownFiles := make(map[string]bool)
	for _, d := range p.store.ListDocuments() {
		knownFiles[d.FilePath] = true
	}

	var processed, totalChunks int

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if ignoredDirs[d.Name()] || ignoredDirs[name] || strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				slog.Debug("skipping ignored directory", "dir", path)
				return filepath.SkipDir
			}
			return nil
		}
		if !p.registry.CanLoad(path) {
			return nil
		}
		absPath, _ := filepath.Abs(path)
		if knownFiles[absPath] || knownFiles[path] {
			slog.Debug("skipping already indexed file", "file", path)
			return nil
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

func (p *Processor) RemoveDocument(ctx context.Context, docID string) error {
	p.store.RemoveByDocumentID(docID)
	p.store.RemoveDocument(docID)
	return p.store.Save(p.cfg.VectorStoreDir)
}

func (p *Processor) ClearAll(ctx context.Context) error {
	p.store.Clear()
	return p.store.Save(p.cfg.VectorStoreDir)
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
