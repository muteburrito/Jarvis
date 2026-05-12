package gemma

import "strings"

type PromptProfile string

const (
	ProfileChat     PromptProfile = "chat"
	ProfileThinking PromptProfile = "thinking"
	ProfileResearch PromptProfile = "research"
	ProfileAgent    PromptProfile = "agent"
)

const ThinkingToken = "<|think|>"

type Capabilities struct {
	TextGeneration    bool
	TextInput         bool
	ImageInput        bool
	AudioInput        bool
	VideoInput        bool
	Thinking          bool
	FunctionCalling   bool
	Coding            bool
	Multilingual      bool
	SystemPrompt      bool
	ImageGeneration   bool
	AudioGeneration   bool
	MaxContextTokens  int
	RecommendedUse    string
	RecommendedMemory string
}

func IsGemma4(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return strings.Contains(normalized, "gemma4") || strings.Contains(normalized, "gemma-4")
}

func ThinkingEnabled(profile PromptProfile) bool {
	return profile == ProfileThinking || profile == ProfileResearch || profile == ProfileAgent
}

func SystemPrefix(model string, profile PromptProfile) string {
	if !IsGemma4(model) || !ThinkingEnabled(profile) {
		return ""
	}
	return ThinkingToken + "\n"
}

func SystemGuidance(model string, profile PromptProfile) string {
	if !IsGemma4(model) {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Gemma 4 runtime guidance\n")
	b.WriteString("- Use Gemma 4's final answer channel for user-visible text.\n")
	b.WriteString("- Do not include hidden thinking, scratch notes, or analysis markers in the answer.\n")
	b.WriteString("- Keep only final visible answers as conversation history.\n")
	b.WriteString("- Use temperature 1.0, top_p 0.95, and top_k 64 unless a deterministic tool task overrides them.\n")

	switch profile {
	case ProfileThinking:
		b.WriteString("- Reason internally before answering, but only show the final user-facing answer.\n")
	case ProfileResearch:
		b.WriteString("- In research mode, reason carefully before the final answer, then cite only source-backed claims.\n")
	case ProfileAgent:
		b.WriteString("- For agent work, plan internally, use tools deliberately, and make file or command effects visible before acting.\n")
	default:
		b.WriteString("- For everyday chat, keep responses direct and avoid exposing internal reasoning.\n")
	}

	return b.String()
}

func Options(model string, deterministic bool) map[string]interface{} {
	if deterministic {
		return map[string]interface{}{
			"temperature": 0.0,
		}
	}
	if IsGemma4(model) {
		return map[string]interface{}{
			"temperature": 1.0,
			"top_p":       0.95,
			"top_k":       64,
		}
	}
	return map[string]interface{}{
		"temperature": 0.0,
	}
}

func KnownCapabilities(model string) Capabilities {
	normalized := strings.ToLower(strings.TrimSpace(model))
	normalized = strings.ReplaceAll(normalized, "_", "-")

	caps := Capabilities{
		TextGeneration:  true,
		TextInput:       true,
		Thinking:        true,
		FunctionCalling: true,
		Coding:          true,
		Multilingual:    true,
		SystemPrompt:    true,
		ImageGeneration: false,
		AudioGeneration: false,
	}

	switch {
	case strings.Contains(normalized, "e2b"):
		caps.ImageInput = true
		caps.AudioInput = true
		caps.VideoInput = true
		caps.MaxContextTokens = 128000
		caps.RecommendedUse = "edge, laptop, speech, translation, and fast multimodal chat"
		caps.RecommendedMemory = "about 4 GB for 4-bit, 5 to 8 GB for 8-bit"
	case strings.Contains(normalized, "e4b"):
		caps.ImageInput = true
		caps.AudioInput = true
		caps.VideoInput = true
		caps.MaxContextTokens = 128000
		caps.RecommendedUse = "default fast local chat and multimodal understanding"
		caps.RecommendedMemory = "about 5.5 to 6 GB for 4-bit, 9 to 12 GB for 8-bit"
	case strings.Contains(normalized, "26b"):
		caps.ImageInput = true
		caps.VideoInput = true
		caps.MaxContextTokens = 256000
		caps.RecommendedUse = "best speed and quality tradeoff for workstation agent work"
		caps.RecommendedMemory = "about 16 to 18 GB for 4-bit, 28 to 30 GB for 8-bit"
	case strings.Contains(normalized, "31b"):
		caps.ImageInput = true
		caps.VideoInput = true
		caps.MaxContextTokens = 256000
		caps.RecommendedUse = "strongest local quality when memory and latency allow"
		caps.RecommendedMemory = "about 17 to 20 GB for 4-bit, 34 to 38 GB for 8-bit"
	}

	return caps
}
