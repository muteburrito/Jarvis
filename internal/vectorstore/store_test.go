package vectorstore

import "testing"

func TestHybridSearchBoostsExactKeywordMatches(t *testing.T) {
	store := New(2)
	store.Add([]Entry{
		{
			ID:        "semantic",
			Content:   "general startup configuration notes",
			Embedding: []float32{1, 0},
		},
		{
			ID:        "exact",
			Content:   "the ZX_900_TIMEOUT setting controls startup retries",
			Embedding: []float32{9, 1},
		},
	})

	results := store.HybridSearch([]float32{1, 0}, "ZX_900_TIMEOUT", 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "exact" {
		t.Fatalf("expected exact keyword match first, got %q", results[0].ID)
	}
}

func TestHybridSearchFallsBackToVectorSearchForEmptyText(t *testing.T) {
	store := New(2)
	store.Add([]Entry{
		{
			ID:        "weak",
			Content:   "one",
			Embedding: []float32{0, 1},
		},
		{
			ID:        "strong",
			Content:   "two",
			Embedding: []float32{1, 0},
		},
	})

	results := store.HybridSearch([]float32{1, 0}, "", 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "strong" {
		t.Fatalf("expected vector match first, got %q", results[0].ID)
	}
}

func TestHybridSearchByDocumentIDsLimitsResults(t *testing.T) {
	store := New(2)
	store.Add([]Entry{
		{
			ID:         "first",
			DocumentID: "doc-a",
			Content:    "payment retry policy",
			Embedding:  []float32{1, 0},
		},
		{
			ID:         "second",
			DocumentID: "doc-b",
			Content:    "payment retry policy",
			Embedding:  []float32{1, 0},
		},
	})

	results := store.HybridSearchByDocumentIDs([]float32{1, 0}, "payment", 5, []string{"doc-b"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].DocumentID != "doc-b" {
		t.Fatalf("expected doc-b result, got %q", results[0].DocumentID)
	}
}
