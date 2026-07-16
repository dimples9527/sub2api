@echo off
setlocal EnableExtensions
cd /d "%~dp0"

if "%~1"=="" (
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\local-dev\sub2api-dev.ps1" menu
) else (
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\local-dev\sub2api-dev.ps1" %*
)
exit /b %errorlevel%
