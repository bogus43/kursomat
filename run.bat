@echo off
setlocal

cd /d "%~dp0"

call "%~dp0build.bat"
if errorlevel 1 (
  exit /b %errorlevel%
)

echo [RUN] Starting kursownik-nbp...
"%~dp0bin\kursownik-nbp.exe" %*
set "EXIT_CODE=%ERRORLEVEL%"
exit /b %EXIT_CODE%

