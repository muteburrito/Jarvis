package workbench

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	defaultToolLimit = 50
	maxToolLimit     = 200
	maxReadBytes     = 240000
)

type FileSearchOptions struct {
	Query string
	Kind  string
	Limit int
}

type FileReadRequest struct {
	Path     string `json:"path"`
	MaxBytes int64  `json:"max_bytes"`
}

type FileSearchResult struct {
	Path      string     `json:"path"`
	Kind      string     `json:"kind"`
	Language  string     `json:"language"`
	SizeBytes int64      `json:"size_bytes"`
	Imports   []string   `json:"imports,omitempty"`
	Media     *MediaInfo `json:"media,omitempty"`
	Score     int        `json:"score"`
}

type FileReadResult struct {
	Path      string     `json:"path"`
	Kind      string     `json:"kind"`
	Language  string     `json:"language"`
	SizeBytes int64      `json:"size_bytes"`
	Media     *MediaInfo `json:"media,omitempty"`
	Content   string     `json:"content,omitempty"`
	Binary    bool       `json:"binary"`
	Truncated bool       `json:"truncated"`
}

type FileSummary struct {
	Path        string       `json:"path"`
	Kind        string       `json:"kind"`
	Language    string       `json:"language"`
	SizeBytes   int64        `json:"size_bytes"`
	Media       *MediaInfo   `json:"media,omitempty"`
	LineCount   int          `json:"line_count"`
	Imports     []string     `json:"imports,omitempty"`
	Symbols     []SymbolInfo `json:"symbols,omitempty"`
	Excerpt     string       `json:"excerpt,omitempty"`
	Binary      bool         `json:"binary"`
	Readable    bool         `json:"readable"`
	Truncated   bool         `json:"truncated"`
	Description string       `json:"description"`
}

func SearchProjectFiles(repo *RepoMap, options FileSearchOptions) []FileSearchResult {
	if repo == nil {
		return nil
	}
	limit := options.Limit
	if limit <= 0 {
		limit = defaultToolLimit
	}
	if limit > maxToolLimit {
		limit = maxToolLimit
	}
	query := strings.ToLower(strings.TrimSpace(options.Query))
	kind := strings.ToLower(strings.TrimSpace(options.Kind))

	var results []FileSearchResult
	for _, file := range repo.Files {
		if kind != "" && strings.ToLower(file.Kind) != kind {
			continue
		}
		score := fileMatchScore(file, query)
		if query != "" && score == 0 {
			continue
		}
		results = append(results, FileSearchResult{
			Path:      file.Path,
			Kind:      file.Kind,
			Language:  file.Language,
			SizeBytes: file.SizeBytes,
			Imports:   file.Imports,
			Media:     file.Media,
			Score:     score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Path < results[j].Path
	})
	if len(results) > limit {
		return results[:limit]
	}
	return results
}

func ReadProjectFile(root string, repo *RepoMap, req FileReadRequest) (FileReadResult, error) {
	relPath, fullPath, err := safeProjectPath(root, req.Path)
	if err != nil {
		return FileReadResult{}, err
	}
	file, ok := repoFileByPath(repo, relPath)
	if !ok {
		return FileReadResult{}, fmt.Errorf("file is not in the workspace map: %s", relPath)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return FileReadResult{}, err
	}
	if info.IsDir() {
		return FileReadResult{}, errors.New("path is a directory")
	}

	limit := req.MaxBytes
	if limit <= 0 || limit > maxReadBytes {
		limit = maxReadBytes
	}
	data, truncated, err := readLimited(fullPath, limit)
	if err != nil {
		return FileReadResult{}, err
	}
	binary := isBinaryData(data)
	result := FileReadResult{
		Path:      relPath,
		Kind:      file.Kind,
		Language:  file.Language,
		SizeBytes: info.Size(),
		Media:     file.Media,
		Binary:    binary,
		Truncated: truncated,
	}
	if !binary && utf8.Valid(data) {
		result.Content = string(data)
	}
	return result, nil
}

func SummarizeProjectFile(root string, repo *RepoMap, path string) (FileSummary, error) {
	read, err := ReadProjectFile(root, repo, FileReadRequest{Path: path, MaxBytes: 64000})
	if err != nil {
		return FileSummary{}, err
	}
	file, _ := repoFileByPath(repo, read.Path)
	summary := FileSummary{
		Path:        read.Path,
		Kind:        read.Kind,
		Language:    read.Language,
		SizeBytes:   read.SizeBytes,
		Media:       file.Media,
		Imports:     file.Imports,
		Binary:      read.Binary,
		Readable:    read.Content != "",
		Truncated:   read.Truncated,
		Description: describeProjectFile(read),
	}
	if read.Content != "" {
		lines := strings.Split(read.Content, "\n")
		summary.LineCount = len(lines)
		summary.Excerpt = strings.Join(lines[:min(len(lines), 20)], "\n")
	}
	for _, symbol := range repo.Symbols {
		if symbol.FilePath == read.Path {
			summary.Symbols = append(summary.Symbols, symbol)
			if len(summary.Symbols) >= 30 {
				break
			}
		}
	}
	return summary, nil
}

func fileMatchScore(file RepoFile, query string) int {
	if query == "" {
		return 1
	}
	path := strings.ToLower(file.Path)
	base := strings.ToLower(filepath.Base(file.Path))
	kind := strings.ToLower(file.Kind)
	language := strings.ToLower(file.Language)

	score := 0
	if base == query {
		score += 100
	}
	if strings.Contains(base, query) {
		score += 50
	}
	if strings.Contains(path, query) {
		score += 25
	}
	if strings.Contains(kind, query) || strings.Contains(language, query) {
		score += 10
	}
	for _, item := range file.Imports {
		if strings.Contains(strings.ToLower(item), query) {
			score += 5
		}
	}
	return score
}

func safeProjectPath(root string, path string) (string, string, error) {
	if strings.TrimSpace(root) == "" {
		return "", "", errors.New("project root is empty")
	}
	if strings.TrimSpace(path) == "" {
		return "", "", errors.New("path is required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	cleanRel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if filepath.IsAbs(cleanRel) || cleanRel == "." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || cleanRel == ".." {
		return "", "", errors.New("path must stay inside the active project")
	}
	fullPath := filepath.Join(rootAbs, cleanRel)
	rel, err := filepath.Rel(rootAbs, fullPath)
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", "", errors.New("path must stay inside the active project")
	}
	return filepath.ToSlash(rel), fullPath, nil
}

func repoFileByPath(repo *RepoMap, path string) (RepoFile, bool) {
	if repo == nil {
		return RepoFile{}, false
	}
	normalized := filepath.ToSlash(filepath.Clean(path))
	for _, file := range repo.Files {
		if filepath.ToSlash(filepath.Clean(file.Path)) == normalized {
			return file, true
		}
	}
	return RepoFile{}, false
}

func readLimited(path string, limit int64) ([]byte, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	var buffer bytes.Buffer
	written, err := buffer.ReadFrom(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, false, err
	}
	data := buffer.Bytes()
	if written > limit {
		return data[:limit], true, nil
	}
	return data, false, nil
}

func isBinaryData(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	prefix := data[:min(len(data), 8192)]
	return bytes.Contains(prefix, []byte{0}) || !utf8.Valid(prefix)
}

func describeProjectFile(file FileReadResult) string {
	if file.Media != nil && file.Media.Width > 0 && file.Media.Height > 0 {
		format := file.Media.Format
		if format == "" {
			format = file.Language
		}
		return fmt.Sprintf("%s image, %dx%d pixels", format, file.Media.Width, file.Media.Height)
	}
	if file.Binary {
		return fmt.Sprintf("%s file, binary content is not shown", file.Kind)
	}
	if file.Content == "" {
		return fmt.Sprintf("%s file, no readable preview available", file.Kind)
	}
	return fmt.Sprintf("%s file with readable %s content", file.Kind, file.Language)
}
