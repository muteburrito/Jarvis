package workbench

import "testing"

func TestChatStorePersistsSessions(t *testing.T) {
	dir := t.TempDir()
	store, err := NewChatStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	session, err := store.Create("First question", "")
	if err != nil {
		t.Fatal(err)
	}
	_, ok, err := store.Update(session.ID, "", []ChatMessage{
		{Role: "user", Content: "How does indexing work?"},
		{Role: "assistant", Content: "It chunks and embeds files."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected session to exist")
	}

	reloaded, err := NewChatStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, ok := reloaded.Get(session.ID)
	if !ok {
		t.Fatal("expected reloaded session")
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loaded.Messages))
	}
	if loaded.Title != "How does indexing work?" {
		t.Fatalf("expected derived title, got %q", loaded.Title)
	}
}

func TestChatStoreFiltersByProject(t *testing.T) {
	dir := t.TempDir()
	store, err := NewChatStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	global, err := store.Create("Global chat", "")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.Create("Project chat", "proj_123")
	if err != nil {
		t.Fatal(err)
	}

	globalOnlyList := store.List("")
	if len(globalOnlyList) != 1 || globalOnlyList[0].ID != global.ID {
		t.Fatalf("expected only global chat without active project, got %#v", globalOnlyList)
	}

	projectList := store.List("proj_123")
	if len(projectList) != 2 {
		t.Fatalf("expected project chat plus legacy global chat, got %#v", projectList)
	}
	foundProject := false
	foundGlobal := false
	for _, summary := range projectList {
		if summary.ID == project.ID && summary.ProjectID == "proj_123" {
			foundProject = true
		}
		if summary.ID == global.ID && summary.ProjectID == "" {
			foundGlobal = true
		}
	}
	if !foundProject || !foundGlobal {
		t.Fatalf("expected project and global summaries, got %#v", projectList)
	}
}
