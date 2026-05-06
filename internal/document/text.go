package document

import (
	"fmt"
	"os"
	"strings"
)

type TextLoader struct{}

func (l *TextLoader) Load(filePath string) ([]Document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("empty file: %s", filePath)
	}

	return []Document{{
		Content:  content,
		Metadata: map[string]string{"source": filePath},
	}}, nil
}

func (l *TextLoader) Extensions() []string {
	return []string{
		".txt", ".md", ".go", ".py", ".js", ".ts", ".java",
		".c", ".cpp", ".h", ".rs", ".rb", ".sh", ".yaml", ".yml",
		".json", ".xml", ".html", ".css", ".sql", ".csv", ".log",
		".cs", ".csproj", ".sln", ".xaml", ".razor",
		".kt", ".swift", ".scala", ".r", ".lua", ".php",
		".tf", ".hcl", ".toml", ".ini", ".cfg", ".conf",
		".proto", ".graphql", ".dockerfile",
	}
}
