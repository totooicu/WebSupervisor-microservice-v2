@echo off
cd /d "%~dp0"
echo [redis_cache-service] starting...
redis_cache-service_windows_amd64.exe -config=./config.json
if errorlevel 1 (
    echo.
    echo [redis_cache-service] exited with error %errorlevel%
    pause
)
