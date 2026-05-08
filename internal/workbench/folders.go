package workbench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WatchedFolder struct {
	Path          string    `json:"path"`
	LastIndexedAt time.Time `json:"last_indexed_at"`
}

type watchedFolderState struct {
	Folders []WatchedFolder `json:"folders"`
}

func LoadWatchedFolders(dataDir string) ([]WatchedFolder, error) {
	data, err := os.ReadFile(watchedFoldersPath(dataDir))
	if err != nil {
		return nil, err
	}

	var state watchedFolderState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state.Folders, nil
}

func SaveWatchedFolder(dataDir string, folderPath string) error {
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return err
	}

	folders, err := LoadWatchedFolders(dataDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	updated := WatchedFolder{
		Path:          filepath.Clean(absPath),
		LastIndexedAt: time.Now(),
	}

	for i, folder := range folders {
		if sameFolderPath(folder.Path, updated.Path) {
			folders[i] = updated
			return SaveWatchedFolders(dataDir, folders)
		}
	}

	folders = append(folders, updated)
	return SaveWatchedFolders(dataDir, folders)
}

func SaveWatchedFolders(dataDir string, folders []WatchedFolder) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(watchedFolderState{Folders: folders}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(watchedFoldersPath(dataDir), data, 0o644)
}

func sameFolderPath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil {
		left = leftAbs
	}
	if rightErr == nil {
		right = rightAbs
	}
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

func watchedFoldersPath(dataDir string) string {
	return filepath.Join(dataDir, "watched_folders.json")
}
