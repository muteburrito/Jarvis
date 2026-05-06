package system

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type HardwareInfo struct {
	TotalRAMMB  int    `json:"total_ram_mb"`
	AvailRAMMB  int    `json:"avail_ram_mb"`
	GPUName     string `json:"gpu_name,omitempty"`
	GPUTotalMB  int    `json:"gpu_total_mb,omitempty"`
	GPUFreeMB   int    `json:"gpu_free_mb,omitempty"`
	GPUDetected bool   `json:"gpu_detected"`
}

type ModelEstimate struct {
	ChatMB      int `json:"chat_mb"`
	EmbeddingMB int `json:"embedding_mb"`
	VisionMB    int `json:"vision_mb"`
	PeakMB      int `json:"peak_mb"`
}

type SystemStatus struct {
	Hardware HardwareInfo  `json:"hardware"`
	Models   ModelEstimate `json:"models"`
	Status   string        `json:"status"`
	Message  string        `json:"message"`
}

func DetectHardware() HardwareInfo {
	info := HardwareInfo{}
	info.TotalRAMMB, info.AvailRAMMB = detectRAM()
	info.GPUName, info.GPUTotalMB, info.GPUFreeMB = detectGPU()
	info.GPUDetected = info.GPUTotalMB > 0
	return info
}

func EstimateModelSizes(chatModel, embeddingModel, visionModel string) ModelEstimate {
	chat := estimateVRAM(chatModel)
	embedding := estimateVRAM(embeddingModel)
	vision := estimateVRAM(visionModel)

	peak := chat + embedding
	if vision > chat {
		peak = vision + embedding
	}

	return ModelEstimate{
		ChatMB:      chat,
		EmbeddingMB: embedding,
		VisionMB:    vision,
		PeakMB:      peak,
	}
}

func EvaluateStatus(hw HardwareInfo, models ModelEstimate) (string, string) {
	if hw.GPUDetected {
		if hw.GPUTotalMB >= models.PeakMB {
			return "good", "GPU has enough VRAM for all models"
		}
		if hw.GPUTotalMB >= models.ChatMB {
			return "warning", "GPU VRAM is tight, some model swapping may occur"
		}
		return "warning", "GPU VRAM is low, models will partially offload to RAM"
	}

	ramNeededMB := models.PeakMB + 4096
	if hw.TotalRAMMB >= ramNeededMB*2 {
		return "good", "Running on CPU, enough RAM available"
	}
	if hw.TotalRAMMB >= ramNeededMB {
		return "warning", "Running on CPU, RAM is sufficient but responses will be slow"
	}
	if hw.TotalRAMMB > 0 {
		return "low", "RAM may not be enough for the configured models"
	}

	return "unknown", "Could not detect hardware"
}

func GetSystemStatus(chatModel, embeddingModel, visionModel string) SystemStatus {
	hw := DetectHardware()
	models := EstimateModelSizes(chatModel, embeddingModel, visionModel)
	status, message := EvaluateStatus(hw, models)
	return SystemStatus{
		Hardware: hw,
		Models:   models,
		Status:   status,
		Message:  message,
	}
}

func estimateVRAM(modelName string) int {
	if modelName == "" {
		return 0
	}

	lower := strings.ToLower(modelName)

	if strings.Contains(lower, "nomic-embed") || strings.Contains(lower, "all-minilm") {
		return 300
	}
	if strings.Contains(lower, "gemma4:26b") {
		return 18000
	}
	if strings.Contains(lower, "gemma4:31b") {
		return 20000
	}
	if strings.Contains(lower, "gemma4:e4b") || lower == "gemma4" || strings.Contains(lower, "gemma4:latest") {
		return 9600
	}
	if strings.Contains(lower, "gemma4:e2b") {
		return 7200
	}

	params := extractParamCount(modelName)
	if params > 0 {
		return int(params * 650)
	}

	if strings.Contains(lower, "llava") {
		return 5000
	}
	if strings.Contains(lower, "moondream") {
		return 1800
	}
	if strings.Contains(lower, "minicpm") {
		return 5000
	}

	return 5000
}

func extractParamCount(model string) float64 {
	parts := strings.Split(model, ":")
	search := model
	if len(parts) > 1 {
		search = parts[1]
	}

	search = strings.ToLower(search)
	idx := strings.Index(search, "b")
	if idx <= 0 {
		return 0
	}

	numStr := ""
	for i := idx - 1; i >= 0; i-- {
		c := search[i]
		if (c >= '0' && c <= '9') || c == '.' {
			numStr = string(c) + numStr
		} else {
			break
		}
	}

	n, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	return n
}

func detectRAM() (totalMB, availMB int) {
	switch runtime.GOOS {
	case "windows":
		return detectRAMWindows()
	case "linux":
		return detectRAMLinux()
	case "darwin":
		return detectRAMDarwin()
	}
	return 0, 0
}

func detectRAMWindows() (totalMB, availMB int) {
	out, err := exec.Command("wmic", "OS", "get",
		"TotalVisibleMemorySize,FreePhysicalMemory",
		"/format:list").Output()
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
			val := strings.TrimPrefix(line, "TotalVisibleMemorySize=")
			if n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
				totalMB = int(n / 1024)
			}
		}
		if strings.HasPrefix(line, "FreePhysicalMemory=") {
			val := strings.TrimPrefix(line, "FreePhysicalMemory=")
			if n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
				availMB = int(n / 1024)
			}
		}
	}
	return
}

func detectRAMLinux() (totalMB, availMB int) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			totalMB = int(kb / 1024)
		case "MemAvailable:":
			availMB = int(kb / 1024)
		}
	}
	return
}

func detectRAMDarwin() (totalMB, availMB int) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, 0
	}
	if n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
		totalMB = int(n / 1024 / 1024)
	}
	return totalMB, 0
}

func detectGPU() (name string, totalMB, freeMB int) {
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=name,memory.total,memory.free",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return "", 0, 0
	}

	line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return "", 0, 0
	}

	name = strings.TrimSpace(parts[0])
	if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
		totalMB = n
	}
	if n, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
		freeMB = n
	}
	return
}
