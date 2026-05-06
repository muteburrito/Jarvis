package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            int
	OllamaURL       string
	OllamaKeepAlive time.Duration
	ChatModel       string
	EmbeddingModel  string
	VisionModel     string
	DataDir         string
	VectorStoreDir  string
	ChunkSize       int
	ChunkOverlap    int
	TopK            int
	MaxUploadBytes  int64
	GitHubRepo      string
	GitHubToken     string
	AppName         string
	SupportEmail    string
	SupportSubject  string
	SupportURL      string
}

func Load() *Config {
	return &Config{
		Port:            envInt("PORT", 8080),
		OllamaURL:       envStr("OLLAMA_URL", "http://localhost:11434"),
		OllamaKeepAlive: envDuration("OLLAMA_KEEP_ALIVE", 30*time.Second),
		ChatModel:       envStr("CHAT_MODEL", "gemma4:e4b"),
		EmbeddingModel:  envStr("EMBEDDING_MODEL", "nomic-embed-text"),
		VisionModel:     envStr("VISION_MODEL", "llava"),
		DataDir:         envStr("DATA_DIR", "./data"),
		VectorStoreDir:  envStr("VECTORSTORE_DIR", "./vectorstore"),
		ChunkSize:       envInt("CHUNK_SIZE", 1000),
		ChunkOverlap:    envInt("CHUNK_OVERLAP", 200),
		TopK:            envInt("TOP_K", 5),
		MaxUploadBytes:  int64(envInt("MAX_UPLOAD_MB", 50)) * 1024 * 1024,
		GitHubRepo:      envStr("GITHUB_REPO", envStr("GITHUB_REPOSITORY", "")),
		GitHubToken:     envStr("GITHUB_TOKEN", ""),
		AppName:         envStr("APP_NAME", "Jarvis"),
		SupportEmail:    envStr("SUPPORT_EMAIL", ""),
		SupportSubject:  envStr("SUPPORT_SUBJECT", "Jarvis Support"),
		SupportURL:      envStr("SUPPORT_URL", ""),
	}
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
