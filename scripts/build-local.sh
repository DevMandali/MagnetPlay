#!/usr/bin/env bash
# MagnetPlay local pre-release build script (Linux)
# Usage: ./scripts/build-local.sh [--skip-jre] [--skip-spring] [--run]
set -euo pipefail

SKIP_JRE=false
SKIP_SPRING=false
RUN_AFTER=false

for arg in "$@"; do
  case $arg in
    --skip-jre)    SKIP_JRE=true ;;
    --skip-spring) SKIP_SPRING=true ;;
    --run)         RUN_AFTER=true ;;
  esac
done

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cyan='\033[0;36m'; green='\033[0;32m'; yellow='\033[0;33m'; red='\033[0;31m'; nc='\033[0m'

step()  { echo -e "\n${cyan}==> $1${nc}"; }
ok()    { echo -e "    ${green}OK: $1${nc}"; }
fail()  { echo -e "    ${red}FAIL: $1${nc}"; exit 1; }

# ── 1. Go binary ─────────────────────────────────────────────────────────────
step "Building Go sidecar (linux/amd64)"
mkdir -p "$ROOT/resources/go-server/current"
(cd "$ROOT/backend/go-server" && go build -o "$ROOT/resources/go-server/current/server" .) \
  || fail "go build failed"
ok "resources/go-server/current/server"

# ── 2. Spring JAR ────────────────────────────────────────────────────────────
if [ "$SKIP_SPRING" = false ]; then
  step "Building Spring Boot JAR"
  mkdir -p "$ROOT/resources/spring"
  (cd "$ROOT/backend/mp-spring" && ./mvnw clean package -DskipTests) \
    || fail "mvn package failed"
  jar=$(ls "$ROOT/backend/mp-spring/target/mp-spring-"*.jar 2>/dev/null | head -1)
  [ -z "$jar" ] && fail "JAR not found in target/"
  cp "$jar" "$ROOT/resources/spring/mp-spring.jar"
  ok "resources/spring/mp-spring.jar"
else
  echo -e "    ${yellow}(skipped Spring build)${nc}"
fi

# ── 3. JRE ───────────────────────────────────────────────────────────────────
if [ "$SKIP_JRE" = false ]; then
  step "Building jlink JRE"
  jre_out="$ROOT/resources/jre/current"
  rm -rf "$jre_out"
  mkdir -p "$ROOT/resources/jre"
  jlink \
    --add-modules java.base,java.logging,java.net.http,java.xml,java.naming,java.management,java.instrument,java.security.jgss,jdk.crypto.ec,jdk.unsupported,java.desktop,java.sql,jdk.naming.rmi,jdk.naming.dns \
    --no-header-files --no-man-pages --compress=2 \
    --output "$jre_out" || fail "jlink failed"
  size_mb=$(du -sm "$jre_out" | cut -f1)
  ok "resources/jre/current (~${size_mb} MB)"
else
  echo -e "    ${yellow}(skipped JRE build)${nc}"
fi

# ── 4. Frontend ──────────────────────────────────────────────────────────────
step "Building React frontend"
(cd "$ROOT/frontend" && npm run build) || fail "frontend build failed"
ok "frontend/dist/"

# ── 5. Electron ──────────────────────────────────────────────────────────────
step "Compiling Electron TypeScript"
(cd "$ROOT" && npx tsc -p tsconfig.electron.json) || fail "electron tsc failed"
ok "electron/dist/"

# ── 6. Package ───────────────────────────────────────────────────────────────
step "Packaging with electron-builder (AppImage + deb + rpm)"
(cd "$ROOT" && npx electron-builder --linux AppImage deb rpm --publish never) \
  || fail "electron-builder failed"

appimage=$(ls "$ROOT/dist-electron/"*.AppImage 2>/dev/null | head -1)
[ -z "$appimage" ] && fail "No AppImage found in dist-electron/"
ok "$(basename "$appimage")"

# ── Done ─────────────────────────────────────────────────────────────────────
echo -e "\n${green}Build complete!${nc}"
echo "AppImage: $appimage"

if [ "$RUN_AFTER" = true ]; then
  step "Launching AppImage for manual smoke test"
  chmod +x "$appimage"
  "$appimage" &
fi
