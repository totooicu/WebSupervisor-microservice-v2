@echo off
cd /d "%~dp0"
echo [parser-service] starting...
parser-service_windows_amd64.exe -config=./config.json
if errorlevel 1 (
    echo.
    echo [parser-service] exited with error %errorlevel%
    pause
)
