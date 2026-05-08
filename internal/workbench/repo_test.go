package workbench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRepositoryExtractsGoSymbols(t *testing.T) {
	root := t.TempDir()
	content := `package demo

import "fmt"

type Runner struct{}

func Start() {
	fmt.Println("go")
}
`
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	repoMap, err := ScanRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if repoMap.FileCount != 1 {
		t.Fatalf("expected 1 file, got %d", repoMap.FileCount)
	}
	if repoMap.Files[0].Kind != "code" {
		t.Fatalf("expected code kind, got %q", repoMap.Files[0].Kind)
	}
	if repoMap.SymbolCount != 2 {
		t.Fatalf("expected 2 symbols, got %d", repoMap.SymbolCount)
	}

	names := map[string]bool{}
	for _, symbol := range repoMap.Symbols {
		names[symbol.Name] = true
	}
	if !names["Runner"] || !names["Start"] {
		t.Fatalf("missing expected symbols: %#v", names)
	}
}

func TestScanRepositoryMapsRegularWorkspaceFiles(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"report.pdf":        []byte("%PDF-1.7"),
		"notes.docx":        []byte("fake zip bytes"),
		"deck.pptx":         []byte("fake zip bytes"),
		"budget.xlsx":       []byte("fake zip bytes"),
		"diagram.png":       []byte{0x89, 0x50, 0x4e, 0x47},
		"notes.md":          []byte("# Notes\nregular workspace file"),
		"ignored/video.mp4": []byte("video"),
	}
	if err := os.Mkdir(filepath.Join(root, "ignored"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	repoMap, err := ScanRepository(root)
	if err != nil {
		t.Fatal(err)
	}

	kinds := map[string]bool{}
	for _, file := range repoMap.Files {
		kinds[file.Kind] = true
	}

	for _, kind := range []string{"pdf", "document", "presentation", "spreadsheet", "image"} {
		if !kinds[kind] {
			t.Fatalf("missing mapped kind %q in %#v", kind, kinds)
		}
	}
	if repoMap.FileCount != 6 {
		t.Fatalf("expected 6 mappable files, got %d", repoMap.FileCount)
	}
}
