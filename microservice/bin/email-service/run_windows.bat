@echo off
cd /d "%~dp0"
echo [email-service] starting...
email-service_windows_amd64.exe -config=./config.json
if errorlevel 1 (
    echo.
    echo [email-service] exited with error %errorlevel%
    pause
)
