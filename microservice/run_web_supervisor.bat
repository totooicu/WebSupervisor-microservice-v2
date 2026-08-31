@REM  @echo off
 cd /d "%~dp0"
@REM ./build_web_supervisor.bat

cd crawler-service
start "crawler-service.exe" cmd /k "crawler-service.exe"
cd ..

cd email-service
start "email-service.exe" cmd /k "email-service.exe"
cd ..

cd parser-service
start "parser-service.exe" cmd /k "parser-service.exe"
cd ..

cd redis_cache-service
start "redis_cache-service.exe" cmd /k "redis_cache-service.exe"
cd ..

cd web_supervisor-manager
start "web_supervisor-manager.exe" cmd /k "web_supervisor-manager.exe"
cd ..