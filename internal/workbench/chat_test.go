package workbench

import "testing"

func TestChatStorePersistsSessions(t *testing.T) {
	dir := t.TempDir()
	store, err := NewChatStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	session, err := store.Create("First question")
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
