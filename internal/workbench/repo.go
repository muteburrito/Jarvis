package workbench

import (
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type RepoMap struct {
	Root        string        `json:"root"`
	Files       []RepoFile    `json:"files"`
	Symbols     []SymbolInfo  `json:"symbols"`
	Packages    []PackageInfo `json:"packages,omitempty"`
	Groups      []FileGroup   `json:"groups,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
	FileCount   int           `json:"file_count"`
	SymbolCount int           `json:"symbol_count"`
	TestCount   int           `json:"test_count"`
}

type RepoFile struct {
	Path      string     `json:"path"`
	Directory string     `json:"directory,omitempty"`
	Extension string     `json:"extension,omitempty"`
	Kind      string     `json:"kind"`
	Language  string     `json:"language"`
	Package   string     `json:"package,omitempty"`
	Imports   []string   `json:"imports,omitempty"`
	Media     *MediaInfo `json:"media,omitempty"`
	SizeBytes int64      `json:"size_bytes"`
	IsTest    bool       `json:"is_test,omitempty"`
}

type SymbolInfo struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	FilePath string `json:"file_path"`
	Line     int    `json:"line"`
	Language string `json:"language"`
}

type PackageInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Language    string `json:"language"`
	FileCount   int    `json:"file_count"`
	TestCount   int    `json:"test_count"`
	SymbolCount int    `json:"symbol_count"`
}

type FileGroup struct {
	Path      string         `json:"path"`
	FileCount int            `json:"file_count"`
	TestCount int            `json:"test_count"`
	SizeBytes int64          `json:"size_bytes"`
	Kinds     map[string]int `json:"kinds,omitempty"`
}

type MediaInfo struct {
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Format string `json:"format,omitempty"`
}

var ignoredRepoDirs = map[string]bool{
	".git":         true,
	".idea":        true,
	".next":        true,
	".nuget":       true,
	".vscode":      true,
	".vs":          true,
	"bin":          true,
	"build":        true,
	"Debug":        true,
	"dist":         true,
	"node_modules": true,
	"obj":          true,
	"packages":     true,
	"Release":      true,
	"target":       true,
	"vendor":       true,
	"data":         true,
	"vectorstore":  true,
}

var ignoredRepoFiles = map[string]bool{
	"chat_sessions.json":   true,
	"projects.json":        true,
	"repo_map.json":        true,
	"task_state.json":      true,
	"watched_folders.json": true,
}

func ScanRepository(root string) (*RepoMap, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	repo := &RepoMap{
		Root:      absRoot,
		UpdatedAt: time.Now(),
	}

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if ignoredRepoDirs[name] || ignoredRepoDirs[strings.ToLower(name)] || strings.HasPrefix(name, ".") && path != absRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if ignoredRepoFiles[strings.ToLower(d.Name())] {
			return nil
		}

		kind, language := classifyWorkspaceFile(path)
		if kind == "" {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)
		directory := filepath.ToSlash(filepath.Dir(relPath))
		if directory == "." {
			directory = ""
		}

		file := RepoFile{
			Path:      relPath,
			Directory: directory,
			Extension: strings.TrimPrefix(strings.ToLower(filepath.Ext(relPath)), "."),
			Kind:      kind,
			Language:  language,
			SizeBytes: info.Size(),
			IsTest:    isTestPath(relPath, language),
		}
		if kind == "image" {
			file.Media = readImageMetadata(path)
		}

		if isTextReadable(kind, language) {
			content, err := os.ReadFile(path)
			if err == nil {
				text := string(content)
				file.Package = extractPackageName(text, language)
				file.Imports = extractImports(text, language)
				repo.Symbols = append(repo.Symbols, extractSymbols(text, relPath, language)...)
			}
		}

		repo.Files = append(repo.Files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}

	repo.FileCount = len(repo.Files)
	repo.SymbolCount = len(repo.Symbols)
	repo.Packages, repo.Groups, repo.TestCount = buildWorkspaceMetadata(repo.Files, repo.Symbols)
	return repo, nil
}

func SaveRepoMap(dir string, repo *RepoMap) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(repo, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "repo_map.json"), data, 0o644)
}

func LoadRepoMap(dir string) (*RepoMap, error) {
	data, err := os.ReadFile(filepath.Join(dir, "repo_map.json"))
	if err != nil {
		return nil, err
	}
	var repo RepoMap
	if err := json.Unmarshal(data, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

func classifyWorkspaceFile(path string) (string, string) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "code", "go"
	case ".js", ".jsx":
		return "code", "javascript"
	case ".ts", ".tsx":
		return "code", "typescript"
	case ".py":
		return "code", "python"
	case ".cs":
		return "code", "csharp"
	case ".java":
		return "code", "java"
	case ".c", ".cpp", ".h":
		return "code", "cpp"
	case ".rs":
		return "code", "rust"
	case ".rb":
		return "code", "ruby"
	case ".sh":
		return "code", "shell"
	case ".css":
		return "code", "css"
	case ".html":
		return "code", "html"
	case ".sql":
		return "code", "sql"
	case ".kt":
		return "code", "kotlin"
	case ".swift":
		return "code", "swift"
	case ".php":
		return "code", "php"
	case ".json":
		return "data", "json"
	case ".yaml", ".yml":
		return "data", "yaml"
	case ".xml":
		return "data", "xml"
	case ".csv":
		return "spreadsheet", "csv"
	case ".xlsx":
		return "spreadsheet", "xlsx"
	case ".xls":
		return "spreadsheet", "xls"
	case ".docx":
		return "document", "docx"
	case ".doc":
		return "document", "doc"
	case ".pptx":
		return "presentation", "pptx"
	case ".ppt":
		return "presentation", "ppt"
	case ".pdf":
		return "pdf", "pdf"
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".tiff", ".tif":
		return "image", strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	case ".md":
		return "document", "markdown"
	case ".txt", ".log":
		return "text", strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	case ".toml", ".ini", ".cfg", ".conf", ".tf", ".hcl":
		return "config", strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	default:
		return "", ""
	}
}

func isTextReadable(kind, language string) bool {
	switch kind {
	case "code", "data", "text", "config":
		return true
	case "document":
		return language == "markdown"
	default:
		return false
	}
}

func readImageMetadata(path string) *MediaInfo {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	cfg, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil
	}
	return &MediaInfo{
		Width:  cfg.Width,
		Height: cfg.Height,
		Format: format,
	}
}

func buildWorkspaceMetadata(files []RepoFile, symbols []SymbolInfo) ([]PackageInfo, []FileGroup, int) {
	symbolCounts := make(map[string]int, len(symbols))
	for _, symbol := range symbols {
		symbolCounts[symbol.FilePath]++
	}

	type packageAccumulator struct {
		PackageInfo
		files map[string]bool
	}
	packageMap := map[string]*packageAccumulator{}
	groupMap := map[string]*FileGroup{}
	testCount := 0

	for _, file := range files {
		if file.IsTest {
			testCount++
		}

		groupPath := topLevelGroup(file.Path)
		group := groupMap[groupPath]
		if group == nil {
			group = &FileGroup{Path: groupPath, Kinds: map[string]int{}}
			groupMap[groupPath] = group
		}
		group.FileCount++
		group.SizeBytes += file.SizeBytes
		if file.IsTest {
			group.TestCount++
		}
		if file.Kind != "" {
			group.Kinds[file.Kind]++
		}

		if file.Kind != "code" {
			continue
		}
		packageName := file.Package
		if packageName == "" {
			packageName = packageNameFromPath(file.Directory)
		}
		packagePath := file.Directory
		key := file.Language + "|" + packagePath + "|" + packageName
		pkg := packageMap[key]
		if pkg == nil {
			pkg = &packageAccumulator{
				PackageInfo: PackageInfo{
					Name:     packageName,
					Path:     packagePath,
					Language: file.Language,
				},
				files: map[string]bool{},
			}
			packageMap[key] = pkg
		}
		if !pkg.files[file.Path] {
			pkg.files[file.Path] = true
			pkg.FileCount++
			pkg.SymbolCount += symbolCounts[file.Path]
			if file.IsTest {
				pkg.TestCount++
			}
		}
	}

	packages := make([]PackageInfo, 0, len(packageMap))
	for _, pkg := range packageMap {
		packages = append(packages, pkg.PackageInfo)
	}
	sort.Slice(packages, func(i, j int) bool {
		if packages[i].Path != packages[j].Path {
			return packages[i].Path < packages[j].Path
		}
		if packages[i].Language != packages[j].Language {
			return packages[i].Language < packages[j].Language
		}
		return packages[i].Name < packages[j].Name
	})

	groups := make([]FileGroup, 0, len(groupMap))
	for _, group := range groupMap {
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].FileCount != groups[j].FileCount {
			return groups[i].FileCount > groups[j].FileCount
		}
		return groups[i].Path < groups[j].Path
	})

	return packages, groups, testCount
}

func topLevelGroup(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." || path == "" {
		return "root"
	}
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return "root"
	}
	return parts[0]
}

func packageNameFromPath(path string) string {
	path = strings.Trim(filepath.ToSlash(path), "/")
	if path == "" || path == "." {
		return "root"
	}
	return filepath.Base(path)
}

func isTestPath(path string, language string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(lower)
	if strings.Contains(lower, "/test/") || strings.Contains(lower, "/tests/") {
		return true
	}
	switch language {
	case "go":
		return strings.HasSuffix(base, "_test.go")
	case "javascript", "typescript":
		return strings.Contains(base, ".test.") || strings.Contains(base, ".spec.")
	case "python":
		return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")
	case "csharp":
		return strings.Contains(base, "test")
	default:
		return strings.Contains(base, "test") || strings.Contains(base, "spec")
	}
}

func extractPackageName(text, language string) string {
	switch language {
	case "go":
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "package ") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					return fields[1]
				}
			}
		}
	case "python":
		return ""
	case "javascript", "typescript":
		return ""
	}
	return ""
}

func extractSymbols(text, filePath, language string) []SymbolInfo {
	patterns := symbolPatterns(language)
	if len(patterns) == 0 {
		return nil
	}

	lines := strings.Split(text, "\n")
	var symbols []SymbolInfo
	for lineIndex, line := range lines {
		for kind, pattern := range patterns {
			match := pattern.FindStringSubmatch(line)
			if len(match) < 2 {
				continue
			}
			symbols = append(symbols, SymbolInfo{
				Name:     match[1],
				Kind:     kind,
				FilePath: filePath,
				Line:     lineIndex + 1,
				Language: language,
			})
			break
		}
	}
	return symbols
}

func symbolPatterns(language string) map[string]*regexp.Regexp {
	switch language {
	case "go":
		return map[string]*regexp.Regexp{
			"function": regexp.MustCompile(`^\s*func\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
			"type":     regexp.MustCompile(`^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)\s+`),
			"constant": regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\b`),
			"variable": regexp.MustCompile(`^\s*var\s+([A-Za-z_][A-Za-z0-9_]*)\b`),
		}
	case "javascript", "typescript":
		return map[string]*regexp.Regexp{
			"class":    regexp.MustCompile(`^\s*(?:export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`),
			"function": regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`),
			"constant": regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=`),
		}
	case "python":
		return map[string]*regexp.Regexp{
			"class":    regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)\b`),
			"function": regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
		}
	case "csharp":
		return map[string]*regexp.Regexp{
			"class":     regexp.MustCompile(`^\s*(?:public|private|internal|protected)?\s*(?:sealed\s+|static\s+|abstract\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)\b`),
			"interface": regexp.MustCompile(`^\s*(?:public|private|internal|protected)?\s*interface\s+([A-Za-z_][A-Za-z0-9_]*)\b`),
			"method":    regexp.MustCompile(`^\s*(?:public|private|internal|protected)?\s*(?:static\s+|async\s+)?[A-Za-z0-9_<>,\[\]?]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
		}
	default:
		return nil
	}
}

func extractImports(text, language string) []string {
	var imports []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch language {
		case "go":
			if strings.HasPrefix(line, `"`) && strings.HasSuffix(line, `"`) {
				imports = append(imports, strings.Trim(line, `"`))
			}
		case "javascript", "typescript":
			if strings.HasPrefix(line, "import ") {
				imports = append(imports, line)
			}
		case "python":
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ") {
				imports = append(imports, line)
			}
		case "csharp":
			if strings.HasPrefix(line, "using ") {
				imports = append(imports, strings.TrimSuffix(strings.TrimPrefix(line, "using "), ";"))
			}
		}
	}
	return imports
}
