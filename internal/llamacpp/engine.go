package llamacpp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"

	"go-chatbot/internal/config"
	"go-chatbot/internal/gemma"
)

const DefaultEmbeddingDimension = 768

type Status struct {
	Backend        string `json:"backend"`
	State          string `json:"state"`
	ActiveTask     string `json:"active_task"`
	ProgressPct    int    `json:"progress_pct"`
	CompletedBytes int64  `json:"completed_bytes,omitempty"`
	TotalBytes     int64  `json:"total_bytes,omitempty"`
	Ready          bool   `json:"ready"`
	VisionReady    bool   `json:"vision_ready"`
	ChatModel      string `json:"chat_model"`
	EmbeddingModel string `json:"embedding_model"`
	Error          string `json:"error,omitempty"`
}

type managedServer struct {
	cmd       *exec.Cmd
	baseURL   string
	modelName string
	modelPath string
}

type Engine struct {
	cfg         *config.Config
	mu          sync.RWMutex
	procMu      sync.Mutex
	status      Status
	chatServer  *managedServer
	embedServer *managedServer
	httpClient  *http.Client
}

func NewEngine(cfg *config.Config) *Engine {
	_ = os.MkdirAll(cfg.ModelsDir, 0o755)
	_ = os.MkdirAll(cfg.EngineDir, 0o755)

	e := &Engine{
		cfg: cfg,
		status: Status{
			Backend:        "llamacpp",
			State:          "idle",
			ActiveTask:     "Initializing on-device AI engine",
			ChatModel:      cfg.ChatModel,
			EmbeddingModel: cfg.EmbeddingModel,
			VisionReady:    true,
		},
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
	e.refreshReadyStateLocked()
	return e
}

func (e *Engine) GetStatus() Status {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

func (e *Engine) setProgress(state, task string, completed, total int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.State = state
	e.status.ActiveTask = task
	e.status.CompletedBytes = completed
	e.status.TotalBytes = total
	e.status.Error = ""
	if total > 0 && completed >= 0 {
		pct := int(completed * 100 / total)
		if pct > 100 {
			pct = 100
		}
		e.status.ProgressPct = pct
	} else {
		e.status.ProgressPct = 0
	}
	e.refreshReadyStateLocked()
}

func (e *Engine) setError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.State = "error"
	if err != nil {
		e.status.Error = err.Error()
		e.status.ActiveTask = err.Error()
	}
	e.refreshReadyStateLocked()
}

func (e *Engine) setReady(task string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.State = "ready"
	if task == "" {
		task = "On-device AI ready"
	}
	e.status.ActiveTask = task
	e.status.ProgressPct = 100
	e.status.CompletedBytes = 0
	e.status.TotalBytes = 0
	e.status.Error = ""
	e.status.Ready = true
}

func (e *Engine) refreshReadyStateLocked() {
	spec, ok := ResolveModelSpec(e.cfg.ChatModel)
	bin := FindServerBinary(e.cfg.EngineDir, e.cfg.LlamaServerPath)
	e.status.Ready = ok && IsModelFilePresent(e.cfg.ModelsDir, spec) && bin != ""
}

func (e *Engine) ListModels() []string {
	installed := InstalledModels(e.cfg.ModelsDir)
	seen := make(map[string]bool)
	var out []string
	for _, m := range installed {
		norm := NormalizeModelName(m)
		if !seen[norm] {
			seen[norm] = true
			out = append(out, m)
		}
	}
	return out
}

func (e *Engine) HasModel(model string) bool {
	spec, ok := ResolveModelSpec(model)
	if ok {
		return IsModelFilePresent(e.cfg.ModelsDir, spec)
	}
	path := filepath.Join(e.cfg.ModelsDir, model)
	if !strings.HasSuffix(strings.ToLower(path), ".gguf") {
		path += ".gguf"
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 1024*1024
}

func (e *Engine) PullModel(ctx context.Context, model string, onProgress ProgressFunc) error {
	report := func(status string, completed, total int64) {
		e.setProgress("downloading", status, completed, total)
		if onProgress != nil {
			onProgress(status, completed, total)
		}
	}

	if _, err := EnsureServerBinary(ctx, e.cfg.EngineDir, e.cfg.LlamaServerPath, report); err != nil {
		e.setError(err)
		return err
	}

	spec, ok := ResolveModelSpec(model)
	if !ok {
		err := fmt.Errorf("unknown GGUF model %q", model)
		e.setError(err)
		return err
	}

	destPath := filepath.Join(e.cfg.ModelsDir, spec.Filename)
	if err := DownloadFile(ctx, spec.URL, destPath, spec.Name, report); err != nil {
		e.setError(err)
		return err
	}

	if spec.MMProjURL != "" && spec.MMProjFilename != "" {
		mmDest := filepath.Join(e.cfg.ModelsDir, spec.MMProjFilename)
		if err := DownloadFile(ctx, spec.MMProjURL, mmDest, spec.Name+" vision adapter", report); err != nil {
			slog.Warn("optional vision projector download failed, continuing with text model", "model", spec.Name, "error", err)
		}
	}

	e.setReady("On-device AI ready (" + spec.Name + ")")
	return nil
}

func (e *Engine) Warmup(ctx context.Context) {
	if err := e.PullModel(ctx, e.cfg.EmbeddingModel, nil); err != nil {
		slog.Warn("background embedding model warmup skipped", "error", err)
	}
	if err := e.PullModel(ctx, e.cfg.ChatModel, nil); err != nil {
		slog.Warn("background chat model warmup skipped", "error", err)
		return
	}
	e.setReady("On-device AI ready (" + e.cfg.ChatModel + ")")
}

func (e *Engine) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	srv, err := e.ensureEmbedProcess(ctx)
	if err != nil {
		slog.Debug("using fast local feature embeddings while GGUF embed server is unavailable", "reason", err)
		return LocalHashEmbed(texts, DefaultEmbeddingDimension), nil
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"input": texts,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.baseURL+"/v1/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return LocalHashEmbed(texts, DefaultEmbeddingDimension), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LocalHashEmbed(texts, DefaultEmbeddingDimension), nil
	}

	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || len(parsed.Data) == 0 {
		return LocalHashEmbed(texts, DefaultEmbeddingDimension), nil
	}

	out := make([][]float32, len(parsed.Data))
	for i, item := range parsed.Data {
		out[i] = normalizeOrPadVector(item.Embedding, DefaultEmbeddingDimension)
	}
	return out, nil
}

func (e *Engine) ChatStream(ctx context.Context, model string, messages []api.Message, onToken func(string) error) error {
	if model == "" {
		model = e.cfg.ChatModel
	}
	srv, err := e.ensureChatProcess(ctx, model)
	if err != nil {
		return err
	}

	payload := buildOpenAIChatRequest(model, messages, true, false)
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("on-device chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errText, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("on-device chat HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(errText)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			if data == "[DONE]" {
				break
			}
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" && onToken != nil {
				if err := onToken(choice.Delta.Content); err != nil {
					return err
				}
			}
		}
	}
	return scanner.Err()
}

func (e *Engine) ChatOnce(ctx context.Context, model string, messages []api.Message) (string, error) {
	if model == "" {
		model = e.cfg.ChatModel
	}
	srv, err := e.ensureChatProcess(ctx, model)
	if err != nil {
		return "", err
	}

	payload := buildOpenAIChatRequest(model, messages, false, true)
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("on-device chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errText, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("on-device chat HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(errText)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", nil
	}
	return parsed.Choices[0].Message.Content, nil
}

func (e *Engine) DescribeImage(ctx context.Context, imageData []byte) (string, error) {
	msg := api.Message{
		Role:    "user",
		Content: "Describe this image in detail. Include any text, diagrams, charts, tables, code, or visual elements you see. Be thorough and specific so the description can be used for search and question answering later.",
		Images:  []api.ImageData{imageData},
	}
	var out strings.Builder
	err := e.ChatStream(ctx, e.cfg.ChatModel, []api.Message{msg}, func(tok string) error {
		out.WriteString(tok)
		return nil
	})
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func (e *Engine) Shutdown() {
	e.procMu.Lock()
	defer e.procMu.Unlock()
	if e.chatServer != nil && e.chatServer.cmd != nil && e.chatServer.cmd.Process != nil {
		_ = e.chatServer.cmd.Process.Kill()
		e.chatServer = nil
	}
	if e.embedServer != nil && e.embedServer.cmd != nil && e.embedServer.cmd.Process != nil {
		_ = e.embedServer.cmd.Process.Kill()
		e.embedServer = nil
	}
}

func (e *Engine) ensureChatProcess(ctx context.Context, model string) (*managedServer, error) {
	norm := NormalizeModelName(model)

	e.procMu.Lock()
	if e.chatServer != nil && e.chatServer.modelName == norm && isServerHealthy(e.chatServer.baseURL) {
		srv := e.chatServer
		e.procMu.Unlock()
		return srv, nil
	}
	e.procMu.Unlock()

	if !e.HasModel(norm) {
		if err := e.PullModel(ctx, norm, nil); err != nil {
			return nil, err
		}
	}

	binPath, err := EnsureServerBinary(ctx, e.cfg.EngineDir, e.cfg.LlamaServerPath, nil)
	if err != nil {
		return nil, err
	}

	spec, ok := ResolveModelSpec(norm)
	var modelPath string
	var mmprojPath string
	ctxSize := 8192
	if ok {
		modelPath = filepath.Join(e.cfg.ModelsDir, spec.Filename)
		if spec.ContextSize > 0 {
			ctxSize = spec.ContextSize
		}
		if spec.MMProjFilename != "" {
			candidate := filepath.Join(e.cfg.ModelsDir, spec.MMProjFilename)
			if info, statErr := os.Stat(candidate); statErr == nil && info.Size() > 1024 {
				mmprojPath = candidate
			}
		}
	} else {
		modelPath = filepath.Join(e.cfg.ModelsDir, model)
		if !strings.HasSuffix(strings.ToLower(modelPath), ".gguf") {
			modelPath += ".gguf"
		}
	}

	e.procMu.Lock()
	defer e.procMu.Unlock()

	if e.chatServer != nil && e.chatServer.modelName == norm && isServerHealthy(e.chatServer.baseURL) {
		return e.chatServer, nil
	}
	if e.chatServer != nil && e.chatServer.cmd != nil && e.chatServer.cmd.Process != nil {
		_ = e.chatServer.cmd.Process.Kill()
		e.chatServer = nil
	}

	e.setProgress("starting", "Starting local inference engine ("+norm+")", 0, 0)
	port, err := freeLoopbackPort()
	if err != nil {
		return nil, err
	}

	args := []string{
		"-m", modelPath,
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", port),
		"-c", fmt.Sprintf("%d", ctxSize),
		"-ngl", "99",
	}
	if mmprojPath != "" {
		args = append(args, "--mmproj", mmprojPath)
	}

	srv, err := startLlamaServerProcess(ctx, binPath, args, norm, modelPath, port)
	if err != nil {
		e.setError(err)
		return nil, err
	}
	e.chatServer = srv
	e.setReady("On-device AI ready (" + norm + ")")
	return srv, nil
}

func (e *Engine) ensureEmbedProcess(ctx context.Context) (*managedServer, error) {
	norm := NormalizeModelName(e.cfg.EmbeddingModel)
	spec, ok := ResolveModelSpec(norm)
	if !ok || !IsModelFilePresent(e.cfg.ModelsDir, spec) {
		return nil, fmt.Errorf("embedding GGUF model not downloaded yet")
	}
	binPath := FindServerBinary(e.cfg.EngineDir, e.cfg.LlamaServerPath)
	if binPath == "" {
		return nil, fmt.Errorf("llama-server binary not downloaded yet")
	}

	e.procMu.Lock()
	defer e.procMu.Unlock()

	if e.embedServer != nil && isServerHealthy(e.embedServer.baseURL) {
		return e.embedServer, nil
	}
	if e.embedServer != nil && e.embedServer.cmd != nil && e.embedServer.cmd.Process != nil {
		_ = e.embedServer.cmd.Process.Kill()
		e.embedServer = nil
	}

	port, err := freeLoopbackPort()
	if err != nil {
		return nil, err
	}
	modelPath := filepath.Join(e.cfg.ModelsDir, spec.Filename)
	args := []string{
		"-m", modelPath,
		"--host", "127.0.0.1",
		"--port", fmt.Sprintf("%d", port),
		"--embedding",
		"--pooling", "mean",
		"-c", "2048",
		"-ngl", "99",
	}

	srv, err := startLlamaServerProcess(ctx, binPath, args, norm, modelPath, port)
	if err != nil {
		return nil, err
	}
	e.embedServer = srv
	return srv, nil
}

func startLlamaServerProcess(ctx context.Context, binPath string, args []string, modelName, modelPath string, port int) (*managedServer, error) {
	cmd := exec.Command(binPath, args...)
	cmd.Dir = filepath.Dir(binPath)
	configureChildProcess(cmd)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start llama-server: %w", err)
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			_ = cmd.Process.Kill()
			return nil, ctx.Err()
		}
		if isServerHealthy(baseURL) {
			return &managedServer{
				cmd:       cmd,
				baseURL:   baseURL,
				modelName: modelName,
				modelPath: modelPath,
			}, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	return nil, fmt.Errorf("timed out waiting for llama-server on %s", baseURL)
}

func isServerHealthy(baseURL string) bool {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func freeLoopbackPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func buildOpenAIChatRequest(model string, messages []api.Message, stream, deterministic bool) map[string]interface{} {
	opts := gemma.Options(model, deterministic)
	outMsgs := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		if len(m.Images) == 0 {
			outMsgs = append(outMsgs, map[string]interface{}{
				"role":    m.Role,
				"content": m.Content,
			})
			continue
		}
		parts := []map[string]interface{}{
			{
				"type": "text",
				"text": m.Content,
			},
		}
		for _, img := range m.Images {
			b64 := base64.StdEncoding.EncodeToString(img)
			parts = append(parts, map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]string{
					"url": "data:image/jpeg;base64," + b64,
				},
			})
		}
		outMsgs = append(outMsgs, map[string]interface{}{
			"role":    m.Role,
			"content": parts,
		})
	}

	req := map[string]interface{}{
		"model":    model,
		"messages": outMsgs,
		"stream":   stream,
	}
	if temp, ok := opts["temperature"]; ok {
		req["temperature"] = temp
	}
	if topP, ok := opts["top_p"]; ok {
		req["top_p"] = topP
	}
	if topK, ok := opts["top_k"]; ok {
		req["top_k"] = topK
	}
	return req
}

func LocalHashEmbed(texts []string, dim int) [][]float32 {
	if dim <= 0 {
		dim = DefaultEmbeddingDimension
	}
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec := make([]float32, dim)
		tokens := strings.Fields(strings.ToLower(text))
		for _, tok := range tokens {
			h := fnv.New64a()
			_, _ = h.Write([]byte(tok))
			sum := h.Sum64()
			idx := int(sum % uint64(dim))
			sign := float32(1.0)
			if (sum>>32)&1 == 1 {
				sign = -1.0
			}
			vec[idx] += sign
		}
		results[i] = normalizeOrPadVector(vec, dim)
	}
	return results
}

func normalizeOrPadVector(vec []float32, targetDim int) []float32 {
	out := make([]float32, targetDim)
	copy(out, vec)
	var sumSq float64
	for _, v := range out {
		sumSq += float64(v) * float64(v)
	}
	if sumSq > 0 {
		norm := float32(math.Sqrt(sumSq))
		for i := range out {
			out[i] /= norm
		}
	}
	return out
}
