package llamacpp

import (
	"os"
	"path/filepath"
	"strings"
)

type ModelSpec struct {
	Name           string `json:"name"`
	Filename       string `json:"filename"`
	URL            string `json:"url"`
	MMProjFilename string `json:"mmproj_filename,omitempty"`
	MMProjURL      string `json:"mmproj_url,omitempty"`
	SizeMB         int64  `json:"size_mb"`
	IsEmbedding    bool   `json:"is_embedding"`
	ContextSize    int    `json:"context_size"`
}

func DefaultCatalog() []ModelSpec {
	return []ModelSpec{
		{
			Name:        "nomic-embed-text",
			Filename:    "nomic-embed-text-v1.5.Q8_0.gguf",
			URL:         envOr("GGUF_URL_NOMIC_EMBED", "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q8_0.gguf"),
			SizeMB:      146,
			IsEmbedding: true,
			ContextSize: 2048,
		},
		{
			Name:           "gemma4:e2b",
			Filename:       "gemma-4-E2B-it-Q4_K_M.gguf",
			URL:            envOr("GGUF_URL_GEMMA4_E2B", "https://huggingface.co/unsloth/gemma-4-E2B-it-GGUF/resolve/main/gemma-4-E2B-it-Q4_K_M.gguf"),
			MMProjFilename: "mmproj-gemma-4-E2B-it-F16.gguf",
			MMProjURL:      envOr("GGUF_URL_GEMMA4_E2B_MMPROJ", "https://huggingface.co/unsloth/gemma-4-E2B-it-GGUF/resolve/main/mmproj-F16.gguf"),
			SizeMB:         1580,
			ContextSize:    8192,
		},
		{
			Name:           "gemma4:e4b",
			Filename:       "gemma-4-E4B-it-Q4_K_M.gguf",
			URL:            envOr("GGUF_URL_GEMMA4_E4B", "https://huggingface.co/unsloth/gemma-4-E4B-it-GGUF/resolve/main/gemma-4-E4B-it-Q4_K_M.gguf"),
			MMProjFilename: "mmproj-gemma-4-E4B-it-F16.gguf",
			MMProjURL:      envOr("GGUF_URL_GEMMA4_E4B_MMPROJ", "https://huggingface.co/unsloth/gemma-4-E4B-it-GGUF/resolve/main/mmproj-F16.gguf"),
			SizeMB:         2650,
			ContextSize:    8192,
		},
		{
			Name:        "qwen2.5:3b",
			Filename:    "Qwen2.5-3B-Instruct-Q4_K_M.gguf",
			URL:         envOr("GGUF_URL_QWEN25_3B", "https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF/resolve/main/qwen2.5-3b-instruct-q4_k_m.gguf"),
			SizeMB:      2010,
			ContextSize: 8192,
		},
		{
			Name:        "qwen2.5:7b",
			Filename:    "Qwen2.5-7B-Instruct-Q4_K_M.gguf",
			URL:         envOr("GGUF_URL_QWEN25_7B", "https://huggingface.co/Qwen/Qwen2.5-7B-Instruct-GGUF/resolve/main/qwen2.5-7b-instruct-q4_k_m.gguf"),
			SizeMB:      4680,
			ContextSize: 8192,
		},
		{
			Name:        "llama3.2:3b",
			Filename:    "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
			URL:         envOr("GGUF_URL_LLAMA32_3B", "https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF/resolve/main/Llama-3.2-3B-Instruct-Q4_K_M.gguf"),
			SizeMB:      2020,
			ContextSize: 8192,
		},
		{
			Name:        "gemma4:26b",
			Filename:    "gemma-4-26B-A4B-it-Q4_K_M.gguf",
			URL:         envOr("GGUF_URL_GEMMA4_26B", "https://huggingface.co/unsloth/gemma-4-26B-A4B-it-GGUF/resolve/main/gemma-4-26B-A4B-it-Q4_K_M.gguf"),
			SizeMB:      15800,
			ContextSize: 8192,
		},
	}
}

func ResolveModelSpec(name string) (ModelSpec, bool) {
	normalized := NormalizeModelName(name)
	for _, spec := range DefaultCatalog() {
		if NormalizeModelName(spec.Name) == normalized || strings.EqualFold(spec.Filename, normalized) {
			return spec, true
		}
	}
	return ModelSpec{}, false
}

func NormalizeModelName(name string) string {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	trimmed = strings.TrimSuffix(trimmed, ":latest")
	switch trimmed {
	case "gemma4", "gemma-4", "gemma-4-e4b":
		return "gemma4:e4b"
	case "gemma-4-e2b":
		return "gemma4:e2b"
	case "gemma-4-26b":
		return "gemma4:26b"
	case "qwen2.5-3b", "qwen2.5-3b-instruct":
		return "qwen2.5:3b"
	case "qwen2.5-7b", "qwen2.5-7b-instruct", "qwen2.5":
		return "qwen2.5:7b"
	case "llama-3.2-3b", "llama3.2":
		return "llama3.2:3b"
	case "nomic-embed-text-v1.5":
		return "nomic-embed-text"
	default:
		return trimmed
	}
}

func IsModelFilePresent(modelsDir string, spec ModelSpec) bool {
	if modelsDir == "" || spec.Filename == "" {
		return false
	}
	path := filepath.Join(modelsDir, spec.Filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Size() > 1024*1024
}

func InstalledModels(modelsDir string) []string {
	var installed []string
	seen := make(map[string]bool)

	for _, spec := range DefaultCatalog() {
		if IsModelFilePresent(modelsDir, spec) {
			installed = append(installed, spec.Name)
			seen[strings.ToLower(spec.Filename)] = true
		}
	}

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return installed
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".gguf") || strings.HasPrefix(lower, "mmproj") || seen[lower] {
			continue
		}
		installed = append(installed, strings.TrimSuffix(name, filepath.Ext(name)))
	}
	return installed
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
