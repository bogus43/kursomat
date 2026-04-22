@echo off
setlocal

cd /d "%~dp0"

if not exist "bin" (
  mkdir "bin"
)

echo [BUILD] Building kursomat-nbp...
go build -o "bin\kursomat-nbp.exe" ".\cmd\kursomat-nbp"
if errorlevel 1 (
  echo [BUILD] Failed.
  exit /b %errorlevel%
)

echo [BUILD] Success: bin\kursomat-nbp.exe
exit /b 0

