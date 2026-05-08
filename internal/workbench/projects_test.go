package workbench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpsertProjectPersistsActiveProject(t *testing.T) {
	dataDir := t.TempDir()
	projectDir := filepath.Join(t.TempDir(), "Project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, project, err := UpsertProject(dataDir, projectDir, "")
	if err != nil {
		t.Fatal(err)
	}
	if project.ID == "" {
		t.Fatal("expected project id")
	}
	if project.Name != "Project" {
		t.Fatalf("expected folder name, got %q", project.Name)
	}
	if !project.Active {
		t.Fatal("expected project to be active")
	}
	if project.VectorStoreDir == "" {
		t.Fatal("expected future vector store dir")
	}

	state, err := LoadProjectState(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if state.ActiveProjectID != project.ID {
		t.Fatalf("expected active project %q, got %q", project.ID, state.ActiveProjectID)
	}
	if len(state.Projects) != 1 {
		t.Fatalf("expected one project, got %d", len(state.Projects))
	}
}

func TestUpsertProjectDedupesByPath(t *testing.T) {
	dataDir := t.TempDir()
	projectDir := t.TempDir()

	if _, _, err := UpsertProject(dataDir, projectDir, "One"); err != nil {
		t.Fatal(err)
	}
	if _, project, err := UpsertProject(dataDir, filepath.Clean(projectDir), "Two"); err != nil {
		t.Fatal(err)
	} else if project.Name != "Two" {
		t.Fatalf("expected updated name, got %q", project.Name)
	}

	state, err := LoadProjectState(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Projects) != 1 {
		t.Fatalf("expected one deduped project, got %d", len(state.Projects))
	}
}
