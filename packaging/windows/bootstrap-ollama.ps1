param(
    [string[]]$Models = @(
        "nomic-embed-text",
        "gemma4:e2b",
        "gemma4:e4b",
        "llava"
    )
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host "[Jarvis setup] $Message"
}

function Find-Ollama {
    $command = Get-Command ollama -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    $localPath = Join-Path $env:LOCALAPPDATA "Programs\Ollama\ollama.exe"
    if (Test-Path $localPath) {
        return $localPath
    }

    return ""
}

function Install-Ollama {
    Write-Step "Ollama was not found. Installing Ollama for this user."
    $scriptPath = Join-Path $env:TEMP "jarvis-ollama-install.ps1"
    Invoke-WebRequest -Uri "https://ollama.com/install.ps1" -OutFile $scriptPath
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $scriptPath
}

function Start-Ollama {
    param([string]$OllamaPath)

    Write-Step "Starting Ollama."
    Start-Process -FilePath $OllamaPath -WindowStyle Hidden | Out-Null

    for ($i = 0; $i -lt 60; $i++) {
        try {
            Invoke-RestMethod -Uri "http://127.0.0.1:11434/api/tags" -Method Get -TimeoutSec 2 | Out-Null
            Write-Step "Ollama API is ready."
            return
        } catch {
            Start-Sleep -Seconds 2
        }
    }

    throw "Ollama did not become ready in time."
}

function Test-ModelInstalled {
    param(
        [string]$OllamaPath,
        [string]$Model
    )

    $names = & $OllamaPath list 2>$null | Select-Object -Skip 1 | ForEach-Object {
        ($_ -split "\s+")[0]
    }
    return $names -contains $Model
}

function Pull-Model {
    param(
        [string]$OllamaPath,
        [string]$Model
    )

    if (Test-ModelInstalled -OllamaPath $OllamaPath -Model $Model) {
        Write-Step "Model already installed: $Model"
        return
    }

    Write-Step "Pulling model: $Model"
    & $OllamaPath pull $Model
}

try {
    $ollamaPath = Find-Ollama
    if (-not $ollamaPath) {
        Install-Ollama
        $ollamaPath = Find-Ollama
    }

    if (-not $ollamaPath) {
        throw "Ollama install completed, but ollama.exe was not found."
    }

    Start-Ollama -OllamaPath $ollamaPath

    foreach ($model in $Models) {
        Pull-Model -OllamaPath $ollamaPath -Model $model
    }

    Write-Step "Ollama and Jarvis models are ready."
} catch {
    Write-Step "Ollama bootstrap failed: $($_.Exception.Message)"
    Write-Step "Jarvis was installed. You can install Ollama manually later from https://ollama.com/download/windows."
    exit 0
}
