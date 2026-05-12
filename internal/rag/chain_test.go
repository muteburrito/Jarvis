package rag

import (
	"path/filepath"
	"testing"
	"time"

	"go-chatbot/internal/config"
	"go-chatbot/internal/vectorstore"
	"go-chatbot/internal/workbench"
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

func TestSearchStorePrefersActiveProjectStore(t *testing.T) {
	dataDir := t.TempDir()
	globalStore := vectorstore.New(2)
	globalStore.AddDocument(vectorstore.DocumentInfo{
		ID:         "global",
		FilePath:   "global.md",
		UploadedAt: time.Now(),
	})

	projectRoot := filepath.Join(t.TempDir(), "Project")
	state, project, err := workbench.UpsertProject(dataDir, projectRoot, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := workbench.ActiveProject(state); !ok {
		t.Fatal("expected active project")
	}

	projectStore := vectorstore.New(2)
	projectStore.AddDocument(vectorstore.DocumentInfo{
		ID:         "project",
		FilePath:   "project.md",
		UploadedAt: time.Now(),
	})
	projectStore.Add([]vectorstore.Entry{{
		ID:         "project_chunk_0",
		DocumentID: "project",
		Content:    "project only",
		Metadata:   map[string]string{"source": "project.md"},
		Embedding:  []float32{1, 0},
	}})
	if err := projectStore.Save(workbench.ResolveProjectVectorStoreDir(dataDir, project)); err != nil {
		t.Fatal(err)
	}

	chain := NewChain(nil, globalStore, &config.Config{DataDir: dataDir})
	got := chain.searchStore()
	if got.DocumentCount() != 1 {
		t.Fatalf("expected project store document count, got %d", got.DocumentCount())
	}
	if docs := got.ListDocuments(); docs[0].ID != "project" {
		t.Fatalf("expected project store, got documents %#v", docs)
	}
}

func TestShouldUseIndexedContextRoutesGenericQuestionsToGeneralChat(t *testing.T) {
	docs := []vectorstore.DocumentInfo{{Filename: "benefits-policy.pdf", FilePath: "docs/benefits-policy.pdf"}}

	if ShouldUseIndexedContext("what is the capital of France?", nil, docs) {
		t.Fatal("expected generic question to skip indexed context")
	}
}

func TestShouldUseIndexedContextUsesFocusedOrDocumentQuestions(t *testing.T) {
	docs := []vectorstore.DocumentInfo{{Filename: "benefits-policy.pdf", FilePath: "docs/benefits-policy.pdf"}}

	cases := []struct {
		name     string
		question string
		opts     *QueryOptions
	}{
		{
			name:     "focused document",
			question: "what does it say?",
			opts:     &QueryOptions{FocusDocumentIDs: []string{"doc_1"}},
		},
		{
			name:     "document phrase",
			question: "summarize the document",
		},
		{
			name:     "filename",
			question: "what does benefits-policy.pdf say about coverage?",
		},
		{
			name:     "filename without extension",
			question: "summarize benefits-policy",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !ShouldUseIndexedContext(tc.question, tc.opts, docs) {
				t.Fatal("expected indexed context")
			}
		})
	}
}
