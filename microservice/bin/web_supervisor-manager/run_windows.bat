@echo off
cd /d "%~dp0"
echo [web_supervisor-manager] starting...
echo config: config.json   jobs: jobs.json
web_supervisor-manager_windows_amd64.exe -config=./config.json
if errorlevel 1 (
    echo.
    echo [web_supervisor-manager] exited with error %errorlevel%
    pause
)
