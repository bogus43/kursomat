@echo off
setlocal

cd /d "%~dp0"

where wails >nul 2>nul
if errorlevel 1 (
  echo [BUILD] Wails CLI is not installed.
  echo [BUILD] Run: go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
  exit /b 1
)

echo [BUILD] Building Kursomat desktop application...
wails build -clean
if errorlevel 1 (
  echo [BUILD] Failed.
  exit /b %errorlevel%
)

echo [BUILD] Success: build\bin\Kursomat.exe
exit /b 0
