@echo off
cd /d "%~dp0"
echo [crawler-service] starting...
crawler-service_windows_amd64.exe -config=./config.json
if errorlevel 1 (
    echo.
    echo [crawler-service] exited with error %errorlevel%
    pause
)
