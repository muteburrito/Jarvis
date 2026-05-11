package workbench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const maxProjectCommandHistory = 100
const maxProjectEditHistory = 250
const maxProjectPatchHistory = 100

type ProjectActivity struct {
	ProjectID      string               `json:"project_id"`
	CommandPolicy  ProjectCommandPolicy `json:"command_policy"`
	CommandHistory []CommandResult      `json:"command_history"`
	EditHistory    []EditRecord         `json:"edit_history"`
	PatchHistory   []PatchRecord        `json:"patch_history"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

type ProjectCommandPolicy struct {
	ApprovedCommands []ProjectCommandApproval `json:"approved_commands"`
}

type ProjectCommandApproval struct {
	Key        string    `json:"key"`
	ApprovedAt time.Time `json:"approved_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	UseCount   int       `json:"use_count"`
}

type PatchRecord struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	Patch     string    `json:"patch,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func LoadProjectActivity(dataDir string, project Project) (ProjectActivity, error) {
	data, err := os.ReadFile(ProjectActivityPath(dataDir, project))
	if err != nil {
		return ProjectActivity{ProjectID: project.ID}, err
	}
	var activity ProjectActivity
	if err := json.Unmarshal(data, &activity); err != nil {
		return ProjectActivity{ProjectID: project.ID}, err
	}
	if activity.ProjectID == "" {
		activity.ProjectID = project.ID
	}
	return activity, nil
}

func RecordProjectCommand(dataDir string, project Project, result CommandResult) error {
	activity, err := LoadProjectActivity(dataDir, project)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	activity.ProjectID = project.ID
	activity.CommandHistory = append(activity.CommandHistory, result)
	if len(activity.CommandHistory) > maxProjectCommandHistory {
		activity.CommandHistory = activity.CommandHistory[len(activity.CommandHistory)-maxProjectCommandHistory:]
	}
	recordCommandApproval(&activity, commandKey(result.Command, result.Args), result.StartedAt)
	activity.UpdatedAt = time.Now()
	return SaveProjectActivity(dataDir, project, activity)
}

func RecordProjectEdit(dataDir string, project Project, edit EditRecord) error {
	activity, err := LoadProjectActivity(dataDir, project)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if edit.CreatedAt.IsZero() {
		edit.CreatedAt = time.Now()
	}
	activity.ProjectID = project.ID
	activity.EditHistory = append(activity.EditHistory, edit)
	if len(activity.EditHistory) > maxProjectEditHistory {
		activity.EditHistory = activity.EditHistory[len(activity.EditHistory)-maxProjectEditHistory:]
	}
	activity.UpdatedAt = time.Now()
	return SaveProjectActivity(dataDir, project, activity)
}

func RecordProjectPatch(dataDir string, project Project, patch PatchRecord) error {
	activity, err := LoadProjectActivity(dataDir, project)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	now := time.Now()
	if patch.ID == "" {
		patch.ID = patchRecordID(patch.Path, now)
	}
	if patch.Status == "" {
		patch.Status = "proposed"
	}
	if patch.CreatedAt.IsZero() {
		patch.CreatedAt = now
	}
	if patch.UpdatedAt.IsZero() {
		patch.UpdatedAt = now
	}
	activity.ProjectID = project.ID
	activity.PatchHistory = append(activity.PatchHistory, patch)
	if len(activity.PatchHistory) > maxProjectPatchHistory {
		activity.PatchHistory = activity.PatchHistory[len(activity.PatchHistory)-maxProjectPatchHistory:]
	}
	activity.UpdatedAt = now
	return SaveProjectActivity(dataDir, project, activity)
}

func patchRecordID(path string, createdAt time.Time) string {
	key := path + createdAt.UTC().Format(time.RFC3339Nano)
	return "patch_" + projectID(key)[5:]
}

func SaveProjectActivity(dataDir string, project Project, activity ProjectActivity) error {
	path := ProjectActivityPath(dataDir, project)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	sort.Slice(activity.CommandPolicy.ApprovedCommands, func(i, j int) bool {
		return activity.CommandPolicy.ApprovedCommands[i].LastUsedAt.After(activity.CommandPolicy.ApprovedCommands[j].LastUsedAt)
	})
	data, err := json.MarshalIndent(activity, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ProjectActivityPath(dataDir string, project Project) string {
	projectDir := filepath.Dir(ResolveProjectVectorStoreDir(dataDir, project))
	return filepath.Join(projectDir, "activity.json")
}

func recordCommandApproval(activity *ProjectActivity, key string, usedAt time.Time) {
	if key == "" {
		return
	}
	if usedAt.IsZero() {
		usedAt = time.Now()
	}
	for i := range activity.CommandPolicy.ApprovedCommands {
		if activity.CommandPolicy.ApprovedCommands[i].Key == key {
			activity.CommandPolicy.ApprovedCommands[i].LastUsedAt = usedAt
			activity.CommandPolicy.ApprovedCommands[i].UseCount++
			return
		}
	}
	activity.CommandPolicy.ApprovedCommands = append(activity.CommandPolicy.ApprovedCommands, ProjectCommandApproval{
		Key:        key,
		ApprovedAt: usedAt,
		LastUsedAt: usedAt,
		UseCount:   1,
	})
}
