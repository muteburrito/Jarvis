package document

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go-chatbot/internal/config"
	"go-chatbot/internal/vectorstore"
)

func TestClearAllRemovesOnlyOwnedDataFiles(t *testing.T) {
	dataDir := t.TempDir()
	vectorDir := t.TempDir()
	externalDir := t.TempDir()

	ownedPath := filepath.Join(dataDir, "uploaded.txt")
	statePath := filepath.Join(dataDir, "chat_sessions.json")
	externalPath := filepath.Join(externalDir, "source.txt")

	writeTestFile(t, ownedPath, "uploaded")
	writeTestFile(t, statePath, "state")
	writeTestFile(t, externalPath, "external")

	store := vectorstore.New(2)
	store.AddDocument(vectorstore.DocumentInfo{ID: "owned", FilePath: ownedPath, Filename: "uploaded.txt"})
	store.AddDocument(vectorstore.DocumentInfo{ID: "external", FilePath: externalPath, Filename: "source.txt"})
	store.AddDocument(vectorstore.DocumentInfo{ID: "url", FilePath: "https://example.com/doc", Filename: "doc"})

	processor := NewProcessor(nil, nil, nil, store, &config.Config{
		DataDir:        dataDir,
		VectorStoreDir: vectorDir,
	}, false)

	if err := processor.ClearAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("expected owned upload to be removed, got err %v", err)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected state file to remain, got %v", err)
	}
	if _, err := os.Stat(externalPath); err != nil {
		t.Fatalf("expected external source file to remain, got %v", err)
	}
	if store.DocumentCount() != 0 {
		t.Fatalf("expected store documents to be cleared, got %d", store.DocumentCount())
	}
}

func TestRemoveDocumentPreservesExternalSourceFiles(t *testing.T) {
	dataDir := t.TempDir()
	vectorDir := t.TempDir()
	externalDir := t.TempDir()
	externalPath := filepath.Join(externalDir, "source.txt")
	writeTestFile(t, externalPath, "external")

	store := vectorstore.New(2)
	store.AddDocument(vectorstore.DocumentInfo{ID: "external", FilePath: externalPath, Filename: "source.txt"})

	processor := NewProcessor(nil, nil, nil, store, &config.Config{
		DataDir:        dataDir,
		VectorStoreDir: vectorDir,
	}, false)

	if err := processor.RemoveDocument(context.Background(), "external"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(externalPath); err != nil {
		t.Fatalf("expected external source file to remain, got %v", err)
	}
	if store.DocumentCount() != 0 {
		t.Fatalf("expected document to be removed from store, got %d", store.DocumentCount())
	}
}

func TestSameSourceFileRequiresSizeAndModifiedTimeMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.txt")
	writeTestFile(t, path, "same")

	modifiedAt := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, modifiedAt, modifiedAt); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	doc := vectorstore.DocumentInfo{
		FilePath:   path,
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
	}
	if !sameSourceFile(info, doc) {
		t.Fatal("expected matching source file to be unchanged")
	}

	doc.Size++
	if sameSourceFile(info, doc) {
		t.Fatal("expected size change to require re-index")
	}

	doc.Size = info.Size()
	doc.ModifiedAt = time.Time{}
	if sameSourceFile(info, doc) {
		t.Fatal("expected missing modified time to require re-index")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
