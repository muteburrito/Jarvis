package workbench

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxDiffPatchBytes = 120000

type DiffSummary struct {
	Root      string     `json:"root"`
	Files     []DiffFile `json:"files"`
	FileCount int        `json:"file_count"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type DiffFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Binary    bool   `json:"binary"`
	Patch     string `json:"patch,omitempty"`
	Truncated bool   `json:"truncated"`
}

func LoadDiffSummary(root string) (DiffSummary, error) {
	repoRoot, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return DiffSummary{}, fmt.Errorf("active project is not a git repository: %w", err)
	}
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))

	statusText, err := gitOutput(repoRoot, "status", "--porcelain=v1", "-uall")
	if err != nil {
		return DiffSummary{}, err
	}

	summary := DiffSummary{
		Root:      repoRoot,
		UpdatedAt: time.Now(),
	}
	for _, line := range strings.Split(statusText, "\n") {
		status, path, ok := parseGitStatusLine(line)
		if !ok {
			continue
		}
		file := DiffFile{
			Path:   path,
			Status: statusLabel(status),
		}
		if status == "??" {
			file.Additions, file.Binary = countNewFileLines(repoRoot, path)
			file.Patch, file.Truncated = untrackedPatch(repoRoot, path, file.Binary)
		} else {
			file.Additions, file.Deletions, file.Binary = diffCounts(repoRoot, path)
			file.Patch, file.Truncated = trackedPatch(repoRoot, path)
		}
		summary.Additions += file.Additions
		summary.Deletions += file.Deletions
		summary.Files = append(summary.Files, file)
	}
	summary.FileCount = len(summary.Files)
	return summary, nil
}

func gitOutput(root string, args ...string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("project path is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return string(out), nil
}

func parseGitStatusLine(line string) (string, string, bool) {
	if len(line) < 4 {
		return "", "", false
	}
	status := strings.TrimSpace(line[:2])
	path := strings.TrimSpace(line[3:])
	if status == "" || path == "" {
		return "", "", false
	}
	if strings.Contains(path, " -> ") {
		parts := strings.Split(path, " -> ")
		path = parts[len(parts)-1]
	}
	path = strings.Trim(path, `"`)
	return status, filepath.Clean(path), true
}

func statusLabel(status string) string {
	switch {
	case status == "??":
		return "untracked"
	case strings.Contains(status, "A"):
		return "added"
	case strings.Contains(status, "D"):
		return "deleted"
	case strings.Contains(status, "R"):
		return "renamed"
	case strings.Contains(status, "C"):
		return "copied"
	default:
		return "modified"
	}
}

func diffCounts(root string, path string) (int, int, bool) {
	additions, deletions, binary := parseNumstat(mustGitOutput(root, "diff", "--numstat", "--", path))
	stagedAdditions, stagedDeletions, stagedBinary := parseNumstat(mustGitOutput(root, "diff", "--cached", "--numstat", "--", path))
	return additions + stagedAdditions, deletions + stagedDeletions, binary || stagedBinary
}

func parseNumstat(text string) (int, int, bool) {
	additions := 0
	deletions := 0
	binary := false
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		if parts[0] == "-" || parts[1] == "-" {
			binary = true
			continue
		}
		if value, err := strconv.Atoi(parts[0]); err == nil {
			additions += value
		}
		if value, err := strconv.Atoi(parts[1]); err == nil {
			deletions += value
		}
	}
	return additions, deletions, binary
}

func trackedPatch(root string, path string) (string, bool) {
	staged := mustGitOutput(root, "diff", "--cached", "--", path)
	unstaged := mustGitOutput(root, "diff", "--", path)
	patch := strings.TrimSpace(strings.TrimSpace(staged) + "\n" + strings.TrimSpace(unstaged))
	return limitPatch(patch)
}

func untrackedPatch(root string, path string, binary bool) (string, bool) {
	if binary {
		return "Binary file is not shown.", false
	}

	fullPath := filepath.Join(root, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return "", false
	}
	defer file.Close()

	var builder strings.Builder
	builder.WriteString("diff --git a/")
	builder.WriteString(filepath.ToSlash(path))
	builder.WriteString(" b/")
	builder.WriteString(filepath.ToSlash(path))
	builder.WriteString("\nnew file mode 100644\n--- /dev/null\n+++ b/")
	builder.WriteString(filepath.ToSlash(path))
	builder.WriteString("\n")

	lineCount := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		lineCount++
	}
	if _, err := file.Seek(0, 0); err != nil {
		return "", false
	}
	builder.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", lineCount))

	scanner = bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		builder.WriteString("+")
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
	}
	return limitPatch(builder.String())
}

func countNewFileLines(root string, path string) (int, bool) {
	fullPath := filepath.Join(root, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return 0, false
	}
	defer file.Close()

	prefix := make([]byte, 8192)
	n, err := file.Read(prefix)
	if err != nil && n == 0 {
		return 0, false
	}
	if bytes.Contains(prefix[:n], []byte{0}) {
		return 0, true
	}
	if _, err := file.Seek(0, 0); err != nil {
		return 0, false
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, false
}

func mustGitOutput(root string, args ...string) string {
	out, err := gitOutput(root, args...)
	if err != nil {
		return ""
	}
	return out
}

func limitPatch(patch string) (string, bool) {
	if len(patch) <= maxDiffPatchBytes {
		return patch, false
	}
	return patch[:maxDiffPatchBytes] + "\n... diff truncated ...", true
}
