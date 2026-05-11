package gemma

import "strings"

type ThinkingFilter struct {
	onToken    func(string) error
	suppress   bool
	pending    string
	finalSeen  bool
	maxPending int
}

func NewThinkingFilter(onToken func(string) error) *ThinkingFilter {
	return &ThinkingFilter{
		onToken:    onToken,
		maxPending: 64,
	}
}

func (f *ThinkingFilter) Write(token string) error {
	if token == "" {
		return nil
	}

	text := f.pending + token
	f.pending = ""

	for text != "" {
		if f.suppress {
			end := strings.Index(text, "</think>")
			if end >= 0 {
				text = text[end+len("</think>"):]
				f.suppress = false
				continue
			}

			final := strings.Index(text, "<|channel|>final")
			if final >= 0 {
				text = text[final+len("<|channel|>final"):]
				f.suppress = false
				f.finalSeen = true
				continue
			}

			f.pending = keepTagTail(text, f.maxPending)
			return nil
		}

		start := firstMarker(text, []string{"<think>", "<|channel|>analysis"})
		if start < 0 {
			emit, pending := splitPendingTag(text, f.maxPending)
			if emit != "" {
				if err := f.onToken(emit); err != nil {
					return err
				}
			}
			f.pending = pending
			return nil
		}

		if start > 0 {
			if err := f.onToken(text[:start]); err != nil {
				return err
			}
		}

		if strings.HasPrefix(text[start:], "<think>") {
			text = text[start+len("<think>"):]
		} else {
			text = text[start+len("<|channel|>analysis"):]
		}
		f.suppress = true
	}

	return nil
}

func (f *ThinkingFilter) Flush() error {
	if !f.suppress && f.pending != "" {
		err := f.onToken(f.pending)
		f.pending = ""
		return err
	}
	f.pending = ""
	return nil
}

func StripThinking(text string) string {
	filtered := strings.Builder{}
	filter := NewThinkingFilter(func(token string) error {
		filtered.WriteString(token)
		return nil
	})
	_ = filter.Write(text)
	_ = filter.Flush()
	return strings.TrimSpace(filtered.String())
}

func firstMarker(text string, markers []string) int {
	best := -1
	for _, marker := range markers {
		idx := strings.Index(text, marker)
		if idx >= 0 && (best < 0 || idx < best) {
			best = idx
		}
	}
	return best
}

func splitPendingTag(text string, maxPending int) (string, string) {
	start := strings.LastIndex(text, "<")
	if start < 0 || len(text)-start > maxPending {
		return text, ""
	}
	return text[:start], text[start:]
}

func keepTagTail(text string, maxPending int) string {
	if len(text) <= maxPending {
		return text
	}
	return text[len(text)-maxPending:]
}
