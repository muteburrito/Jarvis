package workbench

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadDiffSummary(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "jarvis@example.test")
	runGit(t, dir, "config", "user.name", "Jarvis Test")

	tracked := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "tracked.txt")
	runGit(t, dir, "commit", "-m", "initial")

	if err := os.WriteFile(tracked, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("fresh\nfile\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary, err := LoadDiffSummary(dir)
	if err != nil {
		t.Fatal(err)
	}
	if summary.FileCount != 2 {
		t.Fatalf("expected 2 changed files, got %d", summary.FileCount)
	}
	if summary.Additions < 3 {
		t.Fatalf("expected at least 3 additions, got %d", summary.Additions)
	}
	if !hasDiffFile(summary, "tracked.txt", "modified") {
		t.Fatalf("missing tracked modified file: %#v", summary.Files)
	}
	if !hasDiffFile(summary, "new.txt", "untracked") {
		t.Fatalf("missing untracked file: %#v", summary.Files)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func hasDiffFile(summary DiffSummary, path string, status string) bool {
	for _, file := range summary.Files {
		if file.Path == path && file.Status == status && file.Patch != "" {
			return true
		}
	}
	return false
}
