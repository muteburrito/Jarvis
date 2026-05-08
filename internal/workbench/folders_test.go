package workbench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveWatchedFolderDedupesCaseInsensitivePaths(t *testing.T) {
	dataDir := t.TempDir()
	folder := filepath.Join(t.TempDir(), "Docs")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := SaveWatchedFolder(dataDir, folder); err != nil {
		t.Fatal(err)
	}
	if err := SaveWatchedFolder(dataDir, filepath.Clean(folder)); err != nil {
		t.Fatal(err)
	}

	folders, err := LoadWatchedFolders(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 {
		t.Fatalf("expected one watched folder, got %d", len(folders))
	}
	if folders[0].LastIndexedAt.IsZero() {
		t.Fatal("expected watched folder timestamp to be set")
	}
}
