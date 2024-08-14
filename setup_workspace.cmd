@echo off
setlocal

REM Check for Python installation
where python >nul 2>&1
if %errorlevel% neq 0 (
    echo Python is not installed. Proceeding to install Python 3.11.9...
    goto :install_python
) else (
    echo Python is already installed.
    goto :check_ollama
)

:install_python
echo Installing Python 3.11.9...

REM Download the installer for Python 3.11.9
powershell -Command "Invoke-WebRequest -Uri https://www.python.org/ftp/python/3.11.9/python-3.11.9-amd64.exe -OutFile python-3.11.9-amd64.exe"

REM Run the installer
python-3.11.9-amd64.exe /quiet InstallAllUsers=1 PrependPath=1

REM Clean up the installer
del python-3.11.9-amd64.exe

REM Refresh the environment to recognize the new Python installation
set PATH=%PATH%;C:\Users\%USERNAME%\AppData\Local\Programs\Python\Python311\Scripts;C:\Users\%USERNAME%\AppData\Local\Programs\Python\Python311

REM Verify Python installation
where python >nul 2>&1
if %errorlevel% neq 0 (
    echo Python installation failed.
    goto :end
) else (
    echo Python 3.11.9 successfully installed.
    goto :check_ollama
)

:check_ollama
REM Check for Ollama installation
where ollama >nul 2>&1
if %errorlevel% neq 0 (
    echo Ollama is not installed. Proceeding to install Ollama...
    goto :install_ollama
)

echo Ollama is already installed.
goto :pull_llama3

:install_ollama
echo Installing Ollama...

REM Download Ollama installer
powershell -Command "Invoke-WebRequest -Uri https://ollama.com/download/OllamaSetup.exe -OutFile OllamaSetup.exe"

REM Run the installer
OllamaSetup.exe /quiet

REM Clean up the installer
del OllamaSetup.exe

:pull_llama3
echo Pulling Llama 3 model...
ollama pull llama3

:end
echo Done.
endlocal
pause
