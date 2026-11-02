package llamacpp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"

	"go-chatbot/internal/config"
)

func TestResolveModelSpec(t *testing.T) {
	spec, ok := ResolveModelSpec("gemma4:e4b")
	if !ok {
		t.Fatal("expected gemma4:e4b to resolve")
	}
	if !strings.HasSuffix(spec.Filename, ".gguf") {
		t.Fatalf("expected .gguf filename, got %s", spec.Filename)
	}

	embedSpec, ok := ResolveModelSpec("nomic-embed-text")
	if !ok || !embedSpec.IsEmbedding {
		t.Fatal("expected nomic-embed-text embedding spec")
	}
}

func TestLocalHashEmbed(t *testing.T) {
	vecs := LocalHashEmbed([]string{"hello local ai world", "document rag search"}, DefaultEmbeddingDimension)
	if len(vecs) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vecs))
	}
	if len(vecs[0]) != DefaultEmbeddingDimension {
		t.Fatalf("expected dim %d, got %d", DefaultEmbeddingDimension, len(vecs[0]))
	}
}

func TestDownloadFileWithProgress(t *testing.T) {
	payload := strings.Repeat("GGUF-TEST-BYTES-", 256)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "test.gguf")
	var lastCompleted int64
	err := DownloadFile(context.Background(), srv.URL+"/test.gguf", dest, "test-model", func(_ string, completed, _ int64) {
		lastCompleted = completed
	})
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}
	if lastCompleted != int64(len(payload)) {
		t.Fatalf("expected completed %d, got %d", len(payload), lastCompleted)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != payload {
		t.Fatalf("downloaded file mismatch")
	}
}

func TestEngineChatAndEmbedWithMockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/v1/embeddings":
			w.Header().Set("Content-Type", "application/json")
			vec := make([]float32, DefaultEmbeddingDimension)
			vec[0] = 1.0
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"embedding": vec},
				},
			})
		case "/v1/chat/completions":
			var req struct {
				Stream bool `json:"stream"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Stream {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"On-device "}}]}`)
				_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Gemma 4!"}}]}`)
				_, _ = fmt.Fprintln(w, `data: [DONE]`)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{"message": map[string]string{"content": "Single response"}},
				},
			})
		}
	}))
	defer srv.Close()

	cfg := &config.Config{
		ChatModel:      "gemma4:e2b",
		EmbeddingModel: "nomic-embed-text",
		ModelsDir:      t.TempDir(),
		EngineDir:      t.TempDir(),
	}
	eng := NewEngine(cfg)
	eng.chatServer = &managedServer{
		baseURL:   srv.URL,
		modelName: "gemma4:e2b",
	}
	eng.embedServer = &managedServer{
		baseURL:   srv.URL,
		modelName: "nomic-embed-text",
	}

	var streamed strings.Builder
	err := eng.ChatStream(context.Background(), "gemma4:e2b", []api.Message{{Role: "user", Content: "Hi"}}, func(tok string) error {
		streamed.WriteString(tok)
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}
	if streamed.String() != "On-device Gemma 4!" {
		t.Fatalf("got %q, want %q", streamed.String(), "On-device Gemma 4!")
	}

	once, err := eng.ChatOnce(context.Background(), "gemma4:e2b", []api.Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatalf("ChatOnce failed: %v", err)
	}
	if once != "Single response" {
		t.Fatalf("got %q, want %q", once, "Single response")
	}
}
