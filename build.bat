@echo off
setlocal

cd /d "%~dp0"

if not exist "bin" (
  mkdir "bin"
)

echo [BUILD] Building kursomat...
go build -o "bin\kursomat.exe" ".\cmd\kursomat"
if errorlevel 1 (
  echo [BUILD] Failed.
  exit /b %errorlevel%
)

echo [BUILD] Success: bin\kursomat.exe
exit /b 0
