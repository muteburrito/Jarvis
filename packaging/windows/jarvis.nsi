; Jarvis Windows installer (NSIS)
; Installs per-user to %LOCALAPPDATA%\Programs\Jarvis - no UAC required.
; Usage: makensis -DVERSION=1.2.3 jarvis.nsi

!ifndef VERSION
  !define VERSION "0.0.0"
!endif

Name "Jarvis"
OutFile "jarvis-setup-${VERSION}.exe"
InstallDir "$LOCALAPPDATA\Programs\Jarvis"
InstallDirRegKey HKCU "Software\Jarvis" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma

!include "MUI2.nsh"

!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"
!define MUI_WELCOMEPAGE_TITLE "Welcome to Jarvis ${VERSION}"
!define MUI_WELCOMEPAGE_TEXT "Jarvis is a local, privacy-first document chatbot. This will install Jarvis to $LOCALAPPDATA\Programs\Jarvis.$\n$\nNo data ever leaves your machine."
!define MUI_FINISHPAGE_RUN "$INSTDIR\jarvis.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Launch Jarvis after install"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
  ; Kill any running instance so we can replace the exe
  nsExec::ExecToLog 'taskkill /IM jarvis.exe /F'

  SetOutPath "$INSTDIR"
  File "jarvis.exe"
  File "bootstrap-ollama.ps1"

  ; Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\Jarvis"
  CreateShortcut "$SMPROGRAMS\Jarvis\Jarvis.lnk" "$INSTDIR\jarvis.exe"
  CreateShortcut "$SMPROGRAMS\Jarvis\Uninstall Jarvis.lnk" "$INSTDIR\Uninstall.exe"

  ; Desktop shortcut (optional - comment out if unwanted)
  CreateShortcut "$DESKTOP\Jarvis.lnk" "$INSTDIR\jarvis.exe"

  ; Write uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ; Register in Add/Remove Programs (per-user)
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Jarvis" "DisplayName" "Jarvis"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Jarvis" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Jarvis" "Publisher" "Jarvis"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Jarvis" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Jarvis" "UninstallString" "$INSTDIR\Uninstall.exe"
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Jarvis" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Jarvis" "NoRepair" 1

  ; Save install dir for future reference
  WriteRegStr HKCU "Software\Jarvis" "InstallDir" "$INSTDIR"

  ; Manual installs bootstrap Ollama and Jarvis models. Silent auto-updates skip this.
  IfSilent skip_ollama_bootstrap 0
    DetailPrint "Checking Ollama and required Jarvis models..."
    nsExec::ExecToLog 'powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$INSTDIR\bootstrap-ollama.ps1"'
  skip_ollama_bootstrap:

  ; If this was a silent auto-update (launched by the running app), restart Jarvis automatically.
  ; The /S flag is intended for automated installs, not manual installs.
  IfSilent 0 done
    Exec '"$INSTDIR\jarvis.exe"'
  done:
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\jarvis.exe"
  Delete "$INSTDIR\bootstrap-ollama.ps1"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\Jarvis\Jarvis.lnk"
  Delete "$SMPROGRAMS\Jarvis\Uninstall Jarvis.lnk"
  RMDir "$SMPROGRAMS\Jarvis"
  Delete "$DESKTOP\Jarvis.lnk"

  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Jarvis"
  DeleteRegKey HKCU "Software\Jarvis"
SectionEnd
