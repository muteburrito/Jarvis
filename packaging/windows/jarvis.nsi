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
!define MUI_WELCOMEPAGE_TEXT "Jarvis is a private, 100% on-device AI assistant. This will install Jarvis to $LOCALAPPDATA\Programs\Jarvis.$\n$\nModels download automatically inside the app on first launch."
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
  ; Stop any running instance or background AI server so we can replace the binary
  nsExec::ExecToLog 'taskkill /IM jarvis.exe /F'
  nsExec::ExecToLog 'taskkill /IM llama-server.exe /F'

  SetOutPath "$INSTDIR"
  File "jarvis.exe"

  ; Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\Jarvis"
  CreateShortcut "$SMPROGRAMS\Jarvis\Jarvis.lnk" "$INSTDIR\jarvis.exe"
  CreateShortcut "$SMPROGRAMS\Jarvis\Uninstall Jarvis.lnk" "$INSTDIR\Uninstall.exe"

  ; Desktop shortcut
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

  ; If this was a silent auto-update (launched by the running app), restart Jarvis automatically.
  IfSilent 0 done
    Exec '"$INSTDIR\jarvis.exe"'
  done:
SectionEnd

Section "Uninstall"
  nsExec::ExecToLog 'taskkill /IM jarvis.exe /F'
  nsExec::ExecToLog 'taskkill /IM llama-server.exe /F'

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
