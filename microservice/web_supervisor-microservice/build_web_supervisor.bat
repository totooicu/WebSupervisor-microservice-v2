@echo off
setlocal enabledelayedexpansion

cd /d "%~dp0"

set "services=crawler-service email-service parser-service redis_cache-service web_supervisor-manager"
set "targets=linux/arm64 linux/amd64 windows/amd64"

if not exist "bin" mkdir bin

for %%s in (%services%) do (
    echo === Building %%s ===
    cd %%s
    go mod tidy

    set "servicedir=..\bin\%%s"
    if not exist "!servicedir!" mkdir "!servicedir!"

    for %%t in (%targets%) do (
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