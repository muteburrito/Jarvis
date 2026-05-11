package server

import (
	"strings"
	"testing"
)

func TestToolSearchQueryPrefersFileLikeTerms(t *testing.T) {
	got := toolSearchQuery("Can you inspect internal/server/chat_routes.go for streaming?")
	if got != "internal/server/chat_routes.go" {
		t.Fatalf("query = %q", got)
	}
}

func TestToolSearchQuerySkipsCommonWords(t *testing.T) {
	got := toolSearchQuery("Can you explain the command runner?")
	if got != "command" {
		t.Fatalf("query = %q", got)
	}
}

func TestTrimToolExcerpt(t *testing.T) {
	got := trimToolExcerpt(strings.Repeat("a", 20), 10)
	if !strings.Contains(got, "excerpt truncated") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}
