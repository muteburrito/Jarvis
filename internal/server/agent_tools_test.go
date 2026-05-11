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

func TestParseToolCallsAcceptsFencedJSON(t *testing.T) {
	raw := "```json\n{\"tool_calls\":[{\"tool\":\"search_files\",\"query\":\"routes\",\"limit\":8},{\"tool\":\"summarize_file\",\"path\":\"internal/server/chat_routes.go\"}]}\n```"
	got := parseToolCalls(raw)
	if len(got) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(got))
	}
	if got[0].Tool != "search_files" || got[0].Query != "routes" {
		t.Fatalf("unexpected first call: %#v", got[0])
	}
	if got[1].Tool != "summarize_file" || got[1].Path != "internal/server/chat_routes.go" {
		t.Fatalf("unexpected second call: %#v", got[1])
	}
}

func TestParseToolCallsFiltersUnsafeOrIncompleteCalls(t *testing.T) {
	raw := `{
		"tool_calls": [
			{"tool":"run_shell","path":"README.md"},
			{"tool":"search_files"},
			{"tool":"read_file","path":"README.md"}
		]
	}`
	got := parseToolCalls(raw)
	if len(got) != 1 {
		t.Fatalf("expected 1 valid call, got %d", len(got))
	}
	if got[0].Tool != "read_file" || got[0].Path != "README.md" {
		t.Fatalf("unexpected call: %#v", got[0])
	}
}

func TestParseToolCallsCapsResults(t *testing.T) {
	raw := `{"tool_calls":[
		{"tool":"read_file","path":"a.md"},
		{"tool":"read_file","path":"b.md"},
		{"tool":"read_file","path":"c.md"},
		{"tool":"read_file","path":"d.md"},
		{"tool":"read_file","path":"e.md"}
	]}`
	got := parseToolCalls(raw)
	if len(got) != maxAgentToolCalls {
		t.Fatalf("expected cap at %d, got %d", maxAgentToolCalls, len(got))
	}
}
