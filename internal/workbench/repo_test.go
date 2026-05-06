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
