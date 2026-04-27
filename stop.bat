@echo off
echo ============================================================
echo  MagnetPlay - Stopping Services
echo ============================================================

call :kill_port 5173  "Frontend   (Vite)"
call :kill_port 8080  "Spring Boot (REST)"
call :kill_port 50051 "Go sidecar  (gRPC)"

echo.
echo All services stopped.
exit /b 0

:kill_port
setlocal
set PORT=%~1
set NAME=%~2
echo Stopping %NAME% on port %PORT%...
for /f "tokens=5" %%p in ('netstat -ano 2^>nul ^| findstr /R ":%PORT% .*LISTENING"') do (
    taskkill /PID %%p /F >nul 2>&1
    if not errorlevel 1 (
        echo   Killed PID %%p
    ) else (
        echo   Not running or already stopped.
    )
)
endlocal
exit /b 0
