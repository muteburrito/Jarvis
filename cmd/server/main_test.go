package main

import (
	"testing"

	"go-chatbot/internal/system"
)

func TestSelectChatModel(t *testing.T) {
	tests := []struct {
		name string
		hw   system.HardwareInfo
		want string
	}{
		{
			name: "very low RAM uses tiny model",
			hw:   system.HardwareInfo{TotalRAMMB: 8000},
			want: "gemma4:e2b",
		},
		{
			name: "low RAM uses light model",
			hw:   system.HardwareInfo{TotalRAMMB: 16000},
			want: "gemma4:e4b",
		},
		{
			name: "very low VRAM uses tiny model",
			hw:   system.HardwareInfo{TotalRAMMB: 16000, GPUTotalMB: 6000, GPUDetected: true},
			want: "gemma4:e2b",
		},
		{
			name: "workstation RAM still uses light model without GPU",
			hw:   system.HardwareInfo{TotalRAMMB: 30000},
			want: "gemma4:e4b",
		},
		{
			name: "large GPU uses light model by default",
			hw:   system.HardwareInfo{TotalRAMMB: 16000, GPUTotalMB: 20000},
			want: "gemma4:e4b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selectChatModel(tt.hw); got != tt.want {
				t.Fatalf("selectChatModel() = %q, want %q", got, tt.want)
			}
		})
	}
}
