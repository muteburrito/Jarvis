package workbench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectToolsSearchReadAndSummarize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc hello() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# Notes\n\nRemember the tests.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo, err := ScanRepository(dir)
	if err != nil {
		t.Fatal(err)
	}

	results := SearchProjectFiles(repo, FileSearchOptions{Query: "main", Limit: 10})
	if len(results) != 1 || results[0].Path != "main.go" {
		t.Fatalf("unexpected search results: %#v", results)
	}

	read, err := ReadProjectFile(dir, repo, FileReadRequest{Path: "main.go"})
	if err != nil {
		t.Fatal(err)
	}
	if read.Binary || read.Content == "" || read.Kind != "code" {
		t.Fatalf("unexpected read result: %#v", read)
	}

	summary, err := SummarizeProjectFile(dir, repo, "main.go")
	if err != nil {
		t.Fatal(err)
	}
	if summary.LineCount == 0 || len(summary.Symbols) != 1 || summary.Symbols[0].Name != "hello" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestReadProjectFileRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	repo := &RepoMap{Root: dir}
	if _, err := ReadProjectFile(dir, repo, FileReadRequest{Path: "../outside.txt"}); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestProjectToolsIncludeImageMetadata(t *testing.T) {
	dir := t.TempDir()
	writeTestPNG(t, filepath.Join(dir, "diagram.png"), 16, 9)

	repo, err := ScanRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := SearchProjectFiles(repo, FileSearchOptions{Query: "diagram", Limit: 10})
	if len(results) != 1 || results[0].Media == nil {
		t.Fatalf("expected image metadata in search result: %#v", results)
	}
	summary, err := SummarizeProjectFile(dir, repo, "diagram.png")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Media == nil || summary.Media.Width != 16 || summary.Media.Height != 9 {
		t.Fatalf("unexpected image summary: %#v", summary)
	}
}
