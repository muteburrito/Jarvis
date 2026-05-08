package rag

import (
	"testing"

	"go-chatbot/internal/vectorstore"
)

func TestFilterUserVisibleResultsRemovesInternalStateSources(t *testing.T) {
	results := []vectorstore.SearchResult{
		{Entry: vectorstore.Entry{Metadata: map[string]string{"source": `D:\repo\data\chat_sessions.json`}}},
		{Entry: vectorstore.Entry{Metadata: map[string]string{"source": `D:\repo\docs\guide.md`}}},
		{Entry: vectorstore.Entry{Metadata: map[string]string{"source": `repo_map.json`}}},
	}

	filtered := filterUserVisibleResults(results)
	if len(filtered) != 1 {
		t.Fatalf("expected one visible result, got %d", len(filtered))
	}
	if filtered[0].Metadata["source"] != `D:\repo\docs\guide.md` {
		t.Fatalf("unexpected visible source: %#v", filtered[0].Metadata)
	}
}
