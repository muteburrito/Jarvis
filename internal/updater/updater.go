package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var builtinRepo string

type Status struct {
	Current         string    `json:"current"`
	Latest          string    `json:"latest,omitempty"`
	LatestNotes     string    `json:"latest_notes,omitempty"`
	UpdateAvailable bool      `json:"update_available"`
	InstallerURL    string    `json:"installer_url,omitempty"`
	CheckedAt       time.Time `json:"checked_at,omitempty"`
	Applying        bool      `json:"applying"`
	Error           string    `json:"error,omitempty"`
}

type Updater struct {
	current string
	repo    string
	token   string
	client  *http.Client

	mu       sync.RWMutex
	status   Status
	applying bool
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Body    string        `json:"body"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func New(current, repo, token string) *Updater {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		repo = strings.TrimSpace(builtinRepo)
	}
	return &Updater{
		current: current,
		repo:    repo,
		token:   strings.TrimSpace(token),
		client:  &http.Client{Timeout: 30 * time.Second},
		status: Status{
			Current: current,
		},
	}
}

func (u *Updater) Start(ctx context.Context) {
	if u.repo == "" {
		slog.Info("updater: GITHUB_REPO not set, update checks disabled")
		return
	}
	if !isSemver(u.current) {
		slog.Info("updater: running dev build, update checks disabled")
		return
	}
	u.check(ctx)
	go func() {
		ticker := time.NewTicker(60 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.check(ctx)
			}
		}
	}()
}

func (u *Updater) Status() Status {
	u.mu.RLock()
	defer u.mu.RUnlock()
	st := u.status
	st.Applying = u.applying
	return st
}

func (u *Updater) check(ctx context.Context) {
	release, err := u.latestRelease(ctx)
	if err != nil {
		slog.Info("updater: check failed", "error", err)
		u.updateStatus(func(st *Status) {
			st.Error = err.Error()
			st.CheckedAt = time.Now()
		})
		return
	}

	tag := strings.TrimSpace(release.TagName)
	installerURL := findInstallerURL(release.Assets)
	st := Status{
		Current:         u.current,
		Latest:          tag,
		LatestNotes:     release.Body,
		UpdateAvailable: less(u.current, tag),
		InstallerURL:    installerURL,
		CheckedAt:       time.Now(),
	}
	u.updateStatus(func(status *Status) {
		*status = st
		status.Applying = u.applying
	})
	if st.UpdateAvailable {
		slog.Info("updater: new version available", "current", u.current, "latest", tag)
	}
}

func (u *Updater) latestRelease(ctx context.Context) (githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", strings.Trim(u.repo, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, err
	}
	u.addHeaders(req)
	resp, err := u.client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return githubRelease{}, fmt.Errorf("latest GitHub release not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubRelease{}, fmt.Errorf("GitHub release check returned %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return githubRelease{}, fmt.Errorf("latest GitHub release did not include a tag")
	}
	return release, nil
}

func (u *Updater) Apply(ctx context.Context) error {
	st := u.Status()
	if !st.UpdateAvailable {
		return fmt.Errorf("no update available")
	}
	if st.InstallerURL == "" {
		return fmt.Errorf("no installer asset found for this release")
	}
	u.mu.Lock()
	if u.applying {
		u.mu.Unlock()
		return fmt.Errorf("update already in progress")
	}
	u.applying = true
	u.status.Applying = true
	u.mu.Unlock()

	ext := filepath.Ext(st.InstallerURL)
	if ext == "" {
		ext = ".exe"
	}
	dest := filepath.Join(os.TempDir(), fmt.Sprintf("jarvis-update-%s%s", st.Latest, ext))
	if err := u.download(ctx, st.InstallerURL, dest); err != nil {
		u.setApplying(false)
		return err
	}
	if err := launchInstaller(dest); err != nil {
		u.setApplying(false)
		return err
	}
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	return nil
}

func (u *Updater) download(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	u.addHeaders(req)
	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return validateDownload(dest)
}

func (u *Updater) addHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Jarvis-Updater")
	req.Header.Set("Accept", "application/vnd.github+json")
	if u.token != "" {
		req.Header.Set("Authorization", "Bearer "+u.token)
	}
}

func (u *Updater) updateStatus(fn func(*Status)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fn(&u.status)
}

func (u *Updater) setApplying(applying bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.applying = applying
	u.status.Applying = applying
}

func findInstallerURL(assets []githubAsset) string {
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if runtime.GOOS == "windows" && strings.HasSuffix(name, ".exe") && strings.Contains(name, "setup") {
			return asset.BrowserDownloadURL
		}
	}
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if runtime.GOOS == "windows" && strings.HasSuffix(name, ".exe") {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

func validateDownload(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("downloaded installer is empty")
	}
	return nil
}
