@echo off
setlocal
set ROOT=%~dp0

echo ============================================================
echo  MagnetPlay - Starting Services
echo ============================================================

echo [1/3] Go sidecar  (gRPC :50051)...
start "MP-Go" cmd /k "cd /d "%ROOT%backend\go-server" && go run main.go"
timeout /t 6 /nobreak >nul

echo [2/3] Spring Boot (REST :8080)...
start "MP-Spring" cmd /k "cd /d "%ROOT%backend\mp-spring" && mvnw.cmd spring-boot:run"
timeout /t 12 /nobreak >nul

echo [3/3] Frontend    (Vite :5173)...
start "MP-Frontend" cmd /k "cd /d "%ROOT%frontend" && npm run dev"

echo.
echo All services launched:
echo   gRPC       localhost:50051
echo   REST API   http://localhost:8080
echo   Frontend   http://localhost:5173
echo   Swagger    http://localhost:8080/swagger-ui.html
echo.
echo Close this window or run stop.bat to shut down.
endlocal
