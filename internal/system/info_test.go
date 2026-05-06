package system

import "testing"

func TestEstimateModelSizesGemma4(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{model: "gemma4:e4b", want: 9600},
		{model: "gemma4:26b", want: 18000},
		{model: "gemma4:31b", want: 20000},
	}

	for _, tt := range tests {
		got := EstimateModelSizes(tt.model, "nomic-embed-text", "").ChatMB
		if got != tt.want {
			t.Fatalf("EstimateModelSizes(%q).ChatMB = %d, want %d", tt.model, got, tt.want)
		}
	}
}
