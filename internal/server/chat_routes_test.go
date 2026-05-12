package server

import "testing"

func TestShouldGatherLiveContextOnlyForCurrentQuestions(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "translation", query: "can you translate this sentence in german", want: false},
		{name: "greeting", query: "what is Hola?", want: false},
		{name: "weather", query: "what is the weather today?", want: true},
		{name: "latest", query: "latest Go release version", want: true},
		{name: "url", query: "summarize https://example.com", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldGatherLiveContext(tc.query, "")
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestShouldGatherLiveContextSkipsCodeSnippet(t *testing.T) {
	if shouldGatherLiveContext("latest Go release", "fmt.Println(1)") {
		t.Fatal("expected code snippet questions to skip quiet live context")
	}
}

func TestIsAutoPullChatModelOnlyAllowsGemmaDropdownModels(t *testing.T) {
	for _, model := range []string{"gemma4:e2b", "gemma4:e4b", "gemma4:26b", "gemma4:e4b:latest"} {
		if !isAutoPullChatModel(model) {
			t.Fatalf("expected %s to be auto-pullable", model)
		}
	}
	for _, model := range []string{"llama3.2", "gemma4:31b", "nomic-embed-text", ""} {
		if isAutoPullChatModel(model) {
			t.Fatalf("expected %s to be rejected", model)
		}
	}
}
