package workbench

import (
	"os"
	"testing"
	"time"
)

func TestRecordProjectCommandPersistsPolicyAndHistory(t *testing.T) {
	dataDir := t.TempDir()
	project := Project{
		ID:             "proj_test",
		Name:           "Test",
		Path:           t.TempDir(),
		VectorStoreDir: "projects/proj_test/vectorstore",
	}

	result := CommandResult{
		Command:   "git",
		Args:      []string{"status", "--short"},
		Directory: project.Path,
		ExitCode:  0,
		StartedAt: time.Now(),
	}
	if err := RecordProjectCommand(dataDir, project, result); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ProjectActivityPath(dataDir, project)); err != nil {
		t.Fatal(err)
	}

	activity, err := LoadProjectActivity(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	if len(activity.CommandHistory) != 1 {
		t.Fatalf("expected one command, got %d", len(activity.CommandHistory))
	}
	if len(activity.CommandPolicy.ApprovedCommands) != 1 {
		t.Fatalf("expected one approval, got %d", len(activity.CommandPolicy.ApprovedCommands))
	}
	if activity.CommandPolicy.ApprovedCommands[0].Key != "git status --short" {
		t.Fatalf("unexpected approval key: %#v", activity.CommandPolicy.ApprovedCommands[0])
	}
}

func TestRecordProjectCommandUpdatesApprovalUseCount(t *testing.T) {
	dataDir := t.TempDir()
	project := Project{
		ID:             "proj_test",
		Name:           "Test",
		Path:           t.TempDir(),
		VectorStoreDir: "projects/proj_test/vectorstore",
	}
	result := CommandResult{
		Command:   "git",
		Args:      []string{"status", "--short"},
		Directory: project.Path,
		ExitCode:  0,
		StartedAt: time.Now(),
	}

	if err := RecordProjectCommand(dataDir, project, result); err != nil {
		t.Fatal(err)
	}
	if err := RecordProjectCommand(dataDir, project, result); err != nil {
		t.Fatal(err)
	}

	activity, err := LoadProjectActivity(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	if len(activity.CommandPolicy.ApprovedCommands) != 1 {
		t.Fatalf("expected one approval, got %d", len(activity.CommandPolicy.ApprovedCommands))
	}
	if activity.CommandPolicy.ApprovedCommands[0].UseCount != 2 {
		t.Fatalf("expected use count 2, got %d", activity.CommandPolicy.ApprovedCommands[0].UseCount)
	}
}

func TestRecordProjectEditPersistsEditHistory(t *testing.T) {
	dataDir := t.TempDir()
	project := Project{
		ID:             "proj_test",
		Name:           "Test",
		Path:           t.TempDir(),
		VectorStoreDir: "projects/proj_test/vectorstore",
	}

	if err := RecordProjectEdit(dataDir, project, EditRecord{
		Path:    "README.md",
		Action:  "proposed",
		Summary: "Updated docs",
	}); err != nil {
		t.Fatal(err)
	}

	activity, err := LoadProjectActivity(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	if len(activity.EditHistory) != 1 {
		t.Fatalf("expected one edit, got %d", len(activity.EditHistory))
	}
	if activity.EditHistory[0].Path != "README.md" {
		t.Fatalf("unexpected edit history: %#v", activity.EditHistory[0])
	}
}

func TestRecordProjectPatchPersistsPatchState(t *testing.T) {
	dataDir := t.TempDir()
	project := Project{
		ID:             "proj_test",
		Name:           "Test",
		Path:           t.TempDir(),
		VectorStoreDir: "projects/proj_test/vectorstore",
	}

	if err := RecordProjectPatch(dataDir, project, PatchRecord{
		Path:    "main.go",
		Summary: "Proposed change",
		Patch:   "@@ -1 +1 @@",
	}); err != nil {
		t.Fatal(err)
	}

	activity, err := LoadProjectActivity(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	if len(activity.PatchHistory) != 1 {
		t.Fatalf("expected one patch, got %d", len(activity.PatchHistory))
	}
	if activity.PatchHistory[0].Status != "proposed" || activity.PatchHistory[0].ID == "" {
		t.Fatalf("unexpected patch state: %#v", activity.PatchHistory[0])
	}
}
