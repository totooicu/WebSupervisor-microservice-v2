@echo off
REM Launch all 5 services in separate console windows.
REM Each window runs <service>\run_windows.bat and stays open (cmd /k).
REM Requires: Redis reachable per each config.json.

setlocal
cd /d "%~dp0"

set "services=crawler-service email-service parser-service redis_cache-service web_supervisor-manager"

echo === Launching all services ===
for %%s in (%services%) do (
    if exist "%%s\run_windows.bat" (
        echo   starting %%s ...
        start "%%s" cmd /k "cd /d %%s && run_windows.bat"
    ) else (
        echo   [SKIP] %%s\run_windows.bat not found
    )
)
echo.
echo All services launched in separate windows.
echo Close each window to stop the corresponding service.
endlocal
