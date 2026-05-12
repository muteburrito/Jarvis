package rag

import (
	"strings"
	"testing"

	"go-chatbot/internal/gemma"
)

func TestBuildSystemPromptOnlyAddsThinkingWhenRequested(t *testing.T) {
	withoutThinking := buildSystemPromptWithThinking("gemma4:e4b", gemma.ProfileResearch, "body", false)
	if strings.Contains(withoutThinking, gemma.ThinkingToken) {
		t.Fatalf("expected no thinking token, got %q", withoutThinking)
	}

	withThinking := buildSystemPromptWithThinking("gemma4:e4b", gemma.ProfileThinking, "body", true)
	if !strings.HasPrefix(withThinking, gemma.ThinkingToken) {
		t.Fatalf("expected thinking token prefix, got %q", withThinking)
	}
}
