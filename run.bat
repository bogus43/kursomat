@echo off
setlocal

cd /d "%~dp0"

where wails >nul 2>nul
if errorlevel 1 (
  echo [RUN] Wails CLI is not installed.
  echo [RUN] Run: go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
  exit /b 1
)

echo [RUN] Starting Kursomat in development mode...
wails dev
set "EXIT_CODE=%ERRORLEVEL%"
exit /b %EXIT_CODE%
