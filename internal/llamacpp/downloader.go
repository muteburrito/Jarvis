package llamacpp

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ProgressFunc func(status string, completed, total int64)

func DownloadFile(ctx context.Context, url, destPath, label string, onProgress ProgressFunc) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		if onProgress != nil {
			onProgress(label+" ready", info.Size(), info.Size())
		}
		return nil
	}

	partPath := destPath + ".part"
	var offset int64
	if info, err := os.Stat(partPath); err == nil {
		offset = info.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Jarvis-Local-AI/2.1")
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", label, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		_ = os.Remove(partPath)
		offset = 0
		return DownloadFile(ctx, url, destPath, label, onProgress)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("download %s returned HTTP %d", label, resp.StatusCode)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == http.StatusPartialContent && offset > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
		offset = 0
	}

	out, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		return fmt.Errorf("open temp file: %w", err)
	}

	total := resp.ContentLength
	if total > 0 {
		total += offset
	}

	buf := make([]byte, 64*1024)
	completed := offset
	lastReport := time.Time{}

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				_ = out.Close()
				return fmt.Errorf("write model data: %w", writeErr)
			}
			completed += int64(n)
			if onProgress != nil && time.Since(lastReport) >= 250*time.Millisecond {
				lastReport = time.Now()
				onProgress("Downloading "+label, completed, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = out.Close()
			return fmt.Errorf("read model stream: %w", readErr)
		}
	}

	if err := out.Sync(); err != nil {
		_ = out.Close()
		return fmt.Errorf("sync model file: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close model file: %w", err)
	}

	if err := os.Rename(partPath, destPath); err != nil {
		return fmt.Errorf("finalize model file: %w", err)
	}

	if onProgress != nil {
		onProgress(label+" ready", completed, completed)
	}
	return nil
}

func ServerBinaryName() string {
	if runtime.GOOS == "windows" {
		return "llama-server.exe"
	}
	return "llama-server"
}

func FindServerBinary(engineDir, explicitPath string) string {
	if explicitPath != "" {
		if info, err := os.Stat(explicitPath); err == nil && !info.IsDir() {
			return explicitPath
		}
	}
	binName := ServerBinaryName()
	if engineDir != "" {
		candidate := filepath.Join(engineDir, binName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	if path, err := exec.LookPath(binName); err == nil {
		return path
	}
	return ""
}

func EnsureServerBinary(ctx context.Context, engineDir, explicitPath string, onProgress ProgressFunc) (string, error) {
	if found := FindServerBinary(engineDir, explicitPath); found != "" {
		return found, nil
	}
	if engineDir == "" {
		return "", fmt.Errorf("engine directory is not configured")
	}
	if err := os.MkdirAll(engineDir, 0o755); err != nil {
		return "", fmt.Errorf("create engine dir: %w", err)
	}

	assetURL, err := resolveLlamaCppAssetURL(ctx)
	if err != nil {
		return "", err
	}

	zipPath := filepath.Join(engineDir, "llama-server-bundle.zip")
	if err := DownloadFile(ctx, assetURL, zipPath, "on-device AI runtime (llama.cpp)", onProgress); err != nil {
		return "", err
	}
	defer os.Remove(zipPath)

	if onProgress != nil {
		onProgress("Extracting on-device AI runtime", 1, 1)
	}
	if err := extractServerBundle(zipPath, engineDir); err != nil {
		return "", err
	}

	found := FindServerBinary(engineDir, explicitPath)
	if found == "" {
		return "", fmt.Errorf("llama-server binary not found after extracting archive")
	}
	return found, nil
}

type githubRelease struct {
	Assets []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func resolveLlamaCppAssetURL(ctx context.Context) (string, error) {
	if customURL := strings.TrimSpace(os.Getenv("LLAMA_SERVER_URL")); customURL != "" {
		return customURL, nil
	}

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://api.github.com/repos/ggml-org/llama.cpp/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Jarvis-Local-AI/2.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("query llama.cpp release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("query llama.cpp release returned HTTP %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("parse llama.cpp release: %w", err)
	}

	url := pickReleaseAsset(rel, runtime.GOOS, runtime.GOARCH)
	if url == "" {
		return "", fmt.Errorf("no matching llama.cpp release asset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return url, nil
}

func pickReleaseAsset(rel githubRelease, goos, goarch string) string {
	var patterns []string
	switch goos {
	case "windows":
		if goarch == "arm64" {
			patterns = []string{"bin-win-cpu-arm64.zip", "win-arm64.zip"}
		} else {
			patterns = []string{"bin-win-vulkan-x64.zip", "bin-win-cpu-x64.zip", "bin-win-avx2-x64.zip"}
		}
	case "darwin":
		if goarch == "arm64" {
			patterns = []string{"bin-macos-arm64.zip", "macos-arm64.zip"}
		} else {
			patterns = []string{"bin-macos-x64.zip", "macos-x64.zip"}
		}
	default:
		patterns = []string{"bin-ubuntu-vulkan-x64.zip", "bin-ubuntu-x64.zip"}
	}

	for _, pat := range patterns {
		for _, asset := range rel.Assets {
			lower := strings.ToLower(asset.Name)
			if strings.HasSuffix(lower, pat) || strings.Contains(lower, pat) {
				return asset.BrowserDownloadURL
			}
		}
	}
	return ""
}

func extractServerBundle(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open runtime zip: %w", err)
	}
	defer r.Close()

	binName := strings.ToLower(ServerBinaryName())
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		lower := strings.ToLower(base)
		isRelevant := lower == binName ||
			strings.HasSuffix(lower, ".dll") ||
			strings.HasSuffix(lower, ".so") ||
			strings.HasSuffix(lower, ".dylib") ||
			strings.HasSuffix(lower, ".metal")
		if !isRelevant {
			continue
		}

		targetPath := filepath.Join(destDir, base)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if lower == binName || strings.HasSuffix(lower, ".so") || strings.HasSuffix(lower, ".dylib") {
			mode = 0o755
		}
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}
