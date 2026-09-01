@echo off
setlocal enabledelayedexpansion

cd /d "%~dp0"

set "services=crawler-service email-service parser-service redis_cache-service web_supervisor-manager"

REM Default targets: all platforms
set "default_targets=linux/arm64 linux/amd64 windows/amd64"

REM Use provided arguments if any, otherwise use default targets
if "%~1"=="" (
    set "targets=%default_targets%"
) else (
    set "targets=%*"
)

if not exist "bin" mkdir bin

for %%s in (%services%) do (
    echo === Building %%s ===
    cd %%s
    go mod tidy

    set "servicedir=..\bin\%%s"
    if not exist "!servicedir!" mkdir "!servicedir!"

    for %%t in (!targets!) do (
        echo %%t | find "/" >nul
        if errorlevel 1 (
            echo Error: Invalid target format '%%t'. Expected format: os/arch
            cd ..
            exit /b 1
        )

        for /f "tokens=1,2 delims=/" %%a in ("%%t") do (
            set "GOOS=%%a"
            set "GOARCH=%%b"
            set "CGO_ENABLED=0"

            set "outfile=!servicedir!\%%s_%%a_%%b"
            if "%%a"=="windows" set "outfile=!outfile!.exe"

            echo   [!GOOS!/!GOARCH!] Building...
            go build -o "!outfile!" .
            if errorlevel 1 (
                echo   [ERROR] Build failed for %%s on %%t
                cd ..
                exit /b 1
            )
        )
    )
    cd ..
)

echo All builds completed.