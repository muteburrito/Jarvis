package workbench

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Project struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Path           string    `json:"path"`
	VectorStoreDir string    `json:"vector_store_dir"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ProjectState struct {
	ActiveProjectID string    `json:"active_project_id"`
	Projects        []Project `json:"projects"`
}

func LoadProjectState(dataDir string) (ProjectState, error) {
	data, err := os.ReadFile(projectsPath(dataDir))
	if err != nil {
		return ProjectState{}, err
	}

	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return ProjectState{}, err
	}
	normalizeProjectState(&state)
	return state, nil
}

func UpsertProject(dataDir string, projectPath string, name string) (ProjectState, Project, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return ProjectState{}, Project{}, err
	}
	absPath = filepath.Clean(absPath)

	state, err := LoadProjectState(dataDir)
	if err != nil && !os.IsNotExist(err) {
		return ProjectState{}, Project{}, err
	}

	now := time.Now()
	id := projectID(absPath)
	projectName := cleanProjectName(name)
	if projectName == "" {
		projectName = filepath.Base(absPath)
	}
	project := Project{
		ID:             id,
		Name:           projectName,
		Path:           absPath,
		VectorStoreDir: filepath.Join("projects", id, "vectorstore"),
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	found := false
	for i := range state.Projects {
		state.Projects[i].Active = false
		if sameProjectPath(state.Projects[i].Path, absPath) {
			project.CreatedAt = state.Projects[i].CreatedAt
			state.Projects[i] = project
			found = true
		}
	}
	if !found {
		state.Projects = append(state.Projects, project)
	}

	state.ActiveProjectID = project.ID
	normalizeProjectState(&state)
	if err := SaveProjectState(dataDir, state); err != nil {
		return ProjectState{}, Project{}, err
	}
	return state, project, nil
}

func SaveProjectState(dataDir string, state ProjectState) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	normalizeProjectState(&state)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(projectsPath(dataDir), data, 0o644)
}

func normalizeProjectState(state *ProjectState) {
	sort.Slice(state.Projects, func(i, j int) bool {
		return state.Projects[i].UpdatedAt.After(state.Projects[j].UpdatedAt)
	})
	for i := range state.Projects {
		state.Projects[i].Active = state.Projects[i].ID == state.ActiveProjectID
	}
}

func projectID(path string) string {
	sum := sha1.Sum([]byte(strings.ToLower(filepath.Clean(path))))
	return "proj_" + hex.EncodeToString(sum[:])[:12]
}

func cleanProjectName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

func sameProjectPath(left string, right string) bool {
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

func projectsPath(dataDir string) string {
	return filepath.Join(dataDir, "projects.json")
}
