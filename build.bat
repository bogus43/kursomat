@echo off
setlocal

cd /d "%~dp0"

if not exist "bin" (
  mkdir "bin"
)

echo [BUILD] Building kursownik-nbp...
go build -o "bin\kursownik-nbp.exe" ".\cmd\kursownik-nbp"
if errorlevel 1 (
  echo [BUILD] Failed.
  exit /b %errorlevel%
)

echo [BUILD] Success: bin\kursownik-nbp.exe
exit /b 0

