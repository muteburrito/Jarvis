package gemma

import "testing"

func TestSystemPrefixOnlyEnablesThinkingForGemmaResearchOrAgent(t *testing.T) {
	if got := SystemPrefix("gemma4:e4b", ProfileResearch); got != ThinkingToken+"\n" {
		t.Fatalf("research prefix = %q", got)
	}
	if got := SystemPrefix("gemma-4-26b-a4b", ProfileAgent); got != ThinkingToken+"\n" {
		t.Fatalf("agent prefix = %q", got)
	}
	if got := SystemPrefix("gemma4:e4b", ProfileChat); got != "" {
		t.Fatalf("chat prefix = %q", got)
	}
	if got := SystemPrefix("llama3.2", ProfileResearch); got != "" {
		t.Fatalf("non-gemma prefix = %q", got)
	}
}

func TestKnownCapabilitiesDoNotClaimGeneration(t *testing.T) {
	caps := KnownCapabilities("gemma4:e4b")
	if !caps.TextGeneration || !caps.Thinking || !caps.FunctionCalling || !caps.SystemPrompt {
		t.Fatalf("expected core Gemma 4 capabilities: %+v", caps)
	}
	if !caps.ImageInput || !caps.AudioInput {
		t.Fatalf("expected e4b image and audio input support: %+v", caps)
	}
	if caps.ImageGeneration || caps.AudioGeneration {
		t.Fatalf("gemma capabilities should not claim native generation: %+v", caps)
	}
}

func TestKnownCapabilitiesByModelSize(t *testing.T) {
	small := KnownCapabilities("gemma4:e2b")
	if small.MaxContextTokens != 128000 || !small.AudioInput {
		t.Fatalf("unexpected e2b capabilities: %+v", small)
	}

	large := KnownCapabilities("google/gemma-4-31B")
	if large.MaxContextTokens != 256000 || large.AudioInput {
		t.Fatalf("unexpected 31b capabilities: %+v", large)
	}
}

func TestStripThinkingRemovesKnownThoughtBlocks(t *testing.T) {
	got := StripThinking("hello <think>private notes</think> world")
	if got != "hello  world" {
		t.Fatalf("strip think block = %q", got)
	}

	got = StripThinking("<|channel|>analysis hidden <|channel|>final visible answer")
	if got != "visible answer" {
		t.Fatalf("strip analysis channel = %q", got)
	}
}
