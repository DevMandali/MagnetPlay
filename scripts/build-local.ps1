# MagnetPlay local pre-release build script (Windows)
# Usage: .\scripts\build-local.ps1 [-SkipJre] [-SkipSpring] [-Run]
param(
    [switch]$SkipJre,
    [switch]$SkipSpring,
    [switch]$Run
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Root = Split-Path $PSScriptRoot -Parent

function Step([string]$name) {
    Write-Host "`n==> $name" -ForegroundColor Cyan
}

function Ok([string]$msg) {
    Write-Host "    OK: $msg" -ForegroundColor Green
}

function Fail([string]$msg) {
    Write-Host "    FAIL: $msg" -ForegroundColor Red
    exit 1
}

# ── 1. Go binary ─────────────────────────────────────────────────────────────
Step "Building Go sidecar (windows/amd64)"
$goOut = "$Root\resources\go-server\current"
New-Item -ItemType Directory -Force -Path $goOut | Out-Null
Push-Location "$Root\backend\go-server"
go build -o "$goOut\server.exe" .
if ($LASTEXITCODE -ne 0) { Fail "go build failed" }
Pop-Location
Ok "resources\go-server\current\server.exe"

# ── 2. Spring JAR ────────────────────────────────────────────────────────────
if (-not $SkipSpring) {
    Step "Building Spring Boot JAR"
    $springOut = "$Root\resources\spring"
    New-Item -ItemType Directory -Force -Path $springOut | Out-Null
    Push-Location "$Root\backend\mp-spring"
    & .\mvnw.cmd clean package -DskipTests
    if ($LASTEXITCODE -ne 0) { Fail "mvn package failed" }
    $jar = Get-ChildItem target\mp-spring-*.jar | Select-Object -First 1
    if (-not $jar) { Fail "JAR not found in target/" }
    Copy-Item $jar.FullName "$springOut\mp-spring.jar" -Force
    Pop-Location
    Ok "resources\spring\mp-spring.jar"
} else {
    Write-Host "    (skipped Spring build)" -ForegroundColor Yellow
}

# ── 3. JRE ───────────────────────────────────────────────────────────────────
if (-not $SkipJre) {
    Step "Building jlink JRE"
    $jreOut = "$Root\resources\jre\current"
    if (Test-Path $jreOut) { Remove-Item $jreOut -Recurse -Force }
    New-Item -ItemType Directory -Force -Path "$Root\resources\jre" | Out-Null
    $modules = "java.base,java.logging,java.net.http,java.xml,java.naming,java.management,java.instrument,java.security.jgss,jdk.crypto.ec,jdk.unsupported,java.desktop,java.sql,jdk.naming.rmi,jdk.naming.dns"
    jlink --add-modules $modules --no-header-files --no-man-pages --compress=2 --output $jreOut
    if ($LASTEXITCODE -ne 0) { Fail "jlink failed" }
    $jreSizeMb = [math]::Round((Get-ChildItem $jreOut -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB, 1)
    Ok "resources\jre\current (~${jreSizeMb} MB)"
} else {
    Write-Host "    (skipped JRE build)" -ForegroundColor Yellow
}

# ── 4. Frontend ──────────────────────────────────────────────────────────────
Step "Building React frontend"
Push-Location "$Root\frontend"
npm run build
if ($LASTEXITCODE -ne 0) { Fail "frontend build failed" }
Pop-Location
Ok "frontend\dist\"

# ── 5. Electron ──────────────────────────────────────────────────────────────
Step "Compiling Electron TypeScript"
Push-Location $Root
npx tsc -p tsconfig.electron.json
if ($LASTEXITCODE -ne 0) { Fail "electron tsc failed" }
Ok "electron\dist\"

# ── 6. Package ───────────────────────────────────────────────────────────────
Step "Packaging with electron-builder (Windows MSI + NSIS)"
npx electron-builder --win msi nsis --publish never
if ($LASTEXITCODE -ne 0) { Fail "electron-builder failed" }
Pop-Location

$msi = Get-ChildItem "$Root\dist-electron\*.msi" | Select-Object -First 1
$exe = Get-ChildItem "$Root\dist-electron\*.exe" | Select-Object -First 1
if (-not $msi -and -not $exe) { Fail "No installer found in dist-electron\" }
if ($msi) { Ok $msi.Name }
if ($exe) { Ok $exe.Name }

# ── Done ─────────────────────────────────────────────────────────────────────
Write-Host "`nBuild complete!" -ForegroundColor Green
if ($msi) { Write-Host "MSI: $($msi.FullName)" -ForegroundColor White }
if ($exe) { Write-Host "EXE: $($exe.FullName)" -ForegroundColor White }

if ($Run) {
    $launcher = if ($msi) { $msi } else { $exe }
    Step "Launching installer for manual smoke test"
    Start-Process $launcher.FullName
}
