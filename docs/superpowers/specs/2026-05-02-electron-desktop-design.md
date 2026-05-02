# MagnetPlay — Electron Desktop App Design

**Date:** 2026-05-02  
**Status:** Approved  
**Scope:** Wrap existing MagnetPlay services in an Electron shell for Windows and Linux desktop distribution, with GitHub Actions CI/CD and auto-update.

---

## 1. Goals

- Ship MagnetPlay as a native desktop app (Windows + Linux) with a single installer
- Electron manages all service processes — no user-facing service management required
- Auto-update via GitHub Releases (silent on Windows/AppImage, link on deb/rpm)
- Hybrid versioning: PR merges build draft releases; `v*` tag promotes to published

## 2. Out of Scope

- macOS support (not targeted in this iteration)
- Docker-based deployment (replaced by Electron packaging)
- Replacing Spring Boot with Go HTTP (Spring Boot kept, JRE bundled)
- Prowlarr/FFmpeg bundling (already handled by Go server's download logic)

---

## 3. Repository Structure

```
MagnetPlay/
├── electron/                        # Electron main process (TypeScript)
│   ├── main.ts                      # App entry, BrowserWindow creation, IPC handlers
│   ├── service-manager.ts           # Spawn, monitor, restart Go + Spring Boot
│   ├── updater.ts                   # electron-updater integration
│   └── preload.ts                   # contextBridge IPC surface for renderer
├── frontend/                        # Existing React/Vite app (source unchanged)
├── backend/
│   ├── go-server/                   # Existing Go sidecar (source unchanged)
│   └── mp-spring/                   # Existing Spring Boot (source unchanged)
├── resources/                       # CI-populated only — git-ignored
│   ├── go-server/
│   │   ├── win-x64/server.exe
│   │   └── linux-x64/server
│   ├── spring/
│   │   └── mp-spring.jar
│   └── jre/
│       ├── win-x64/                 # jlink minimal JRE 17 (~85MB)
│       └── linux-x64/              # jlink minimal JRE 17 (~85MB)
├── build-assets/                    # Icons and installer branding
│   ├── icon.ico                     # Windows
│   ├── icon.icns                    # (future macOS)
│   └── icons/                       # Linux PNG set (16–512px)
├── .github/workflows/
│   ├── build.yml                    # PR merge → bump version + build draft release
│   └── release.yml                  # v* tag push → promote draft to published
├── package.json                     # Root — electron scripts + electron-builder config ref
├── electron-builder.yml             # electron-builder targets and extraResources
└── tsconfig.electron.json           # TypeScript config for electron/ only
```

`resources/` is populated exclusively by CI. It is `.gitignore`d. Developers running locally must run `npm run fetch-resources` (a helper script that downloads latest CI artifacts) or build each component manually.

---

## 4. Process Lifecycle

### 4.1 Startup Sequence

```
Electron app ready
        │
        ▼
Show splash / loading screen
        │
        ▼
ServiceManager.start()
        │
        ├─ 1. Resolve all binary paths via getResourcePath()
        │
        ├─ 2. Set execute permissions on Go binary + java (Linux: chmod 0o755)
        │
        ├─ 3. Spawn Go sidecar
        │      binary:  resources/go-server/{platform}/server[.exe]
        │      dataDir: app.getPath('userData')/downloads
        │      logs:    app.getPath('userData')/logs/go-server.log
        │      env:     GRPC_PORT=50051, DATA_DIR=<userData>/downloads
        │
        ├─ 4. Poll Go ready: TCP connect :50051 (500ms interval, 30s max)
        │
        ├─ 5. Spawn Spring Boot
        │      binary:  resources/jre/{platform}/bin/java[.exe]
        │      args:    -jar resources/spring/mp-spring.jar
        │      logs:    app.getPath('userData')/logs/spring.log
        │      env:     GRPC_TORRENT_SERVICE_ADDRESS=localhost:50051
        │                SERVER_PORT=8080
        │
        ├─ 6. Poll Spring Boot ready: GET http://localhost:8080/actuator/health
        │      (500ms interval, 45s max)
        │
        └─ 7. Hide splash → show BrowserWindow
               prod: file://<app>/frontend/dist/index.html
               dev:  http://localhost:5173
```

### 4.2 Shutdown Sequence

```
app.on('before-quit')
        │
        ├─ Spring Boot: SIGTERM → wait 8s → SIGKILL
        │   Windows: taskkill /F /PID <pid>
        │
        └─ Go sidecar: SIGTERM → wait 8s → SIGKILL
              Go sidecar's own defer handles Prowlarr + FFmpeg children
```

### 4.3 Crash Recovery

- Both processes monitored via `child_process` `exit` event
- Unexpected exit → restart with 2s delay, up to 3 attempts
- After 3 failures: show error dialog with log path, offer quit or open logs folder
- Restart counter resets after 60s of stable running

### 4.4 Platform Abstraction

```typescript
// electron/service-manager.ts

const IS_WIN = process.platform === 'win32';
const OS_DIR = IS_WIN ? 'win-x64' : 'linux-x64';

function getResourcePath(...parts: string[]): string {
  // Packaged: extraResources strips the platform subdir (e.g. win-x64/) → flat layout
  // Dev:      resources/ still has platform subdir → include OS_DIR
  const base = app.isPackaged
    ? process.resourcesPath
    : path.join(__dirname, '../../resources', OS_DIR);
  return path.join(base, ...parts);
}

// Packaged layout:  <resourcesPath>/go-server/server[.exe]
// Dev layout:       resources/win-x64/go-server/server[.exe]  (via OS_DIR above)
const goBinary  = getResourcePath('go-server', IS_WIN ? 'server.exe' : 'server');
const javaExe   = getResourcePath('jre', 'bin', IS_WIN ? 'java.exe' : 'java');
const springJar = getResourcePath('spring', 'mp-spring.jar');

function killProcess(pid: number): void {
  if (IS_WIN) {
    execSync(`taskkill /F /PID ${pid}`);
  } else {
    process.kill(pid, 'SIGTERM');
  }
}
```

---

## 5. Frontend Serving + IPC

### 5.1 Loading

- **Production**: `file://` protocol loads `frontend/dist/index.html` (Vite build output)
- **Development**: loads `http://localhost:5173` (Vite dev server, `app.isPackaged === false`)
- Frontend API calls to `http://localhost:8080` work identically in both modes

### 5.2 CORS

Spring Boot already allows `"null"` origin in CORS config (covers `file://`). No change needed.

### 5.3 Content Security Policy

Set on BrowserWindow `webPreferences`:
```
Content-Security-Policy:
  default-src 'self' file:;
  connect-src 'self' http://localhost:8080 ws://localhost:8080;
  media-src 'self' http://localhost:8080 blob:;
  script-src 'self';
  style-src 'self' 'unsafe-inline';
```

Note: Frontend never connects to Prowlarr directly — all search goes through Spring Boot → Go → Prowlarr. Prowlarr port not needed in CSP.

### 5.4 IPC Surface (minimal)

Renderer → Main (invoke):
```
app:quit               — quit application
app:get-version        — returns app version string
app:open-logs-dir      — opens userData/logs in OS file manager
app:service-status     — returns { go: 'running'|'starting'|'crashed', spring: same }
app:install-update     — call autoUpdater.quitAndInstall()
```

Main → Renderer (send):
```
service:status-changed  — { service: 'go'|'spring', status: string }
update:available        — { version: string }
update:downloaded       — { version: string }
```

---

## 6. Binary Packaging

### 6.1 electron-builder.yml

```yaml
appId: com.devmandali.magnetplay
productName: MagnetPlay
directories:
  output: dist-electron
  buildResources: build-assets

files:
  - electron/dist/**
  - frontend/dist/**
  - package.json

extraResources:
  - from: resources/go-server/win-x64/
    to: go-server/
    filter: ["**"]
  - from: resources/spring/
    to: spring/
    filter: ["*.jar"]
  - from: resources/jre/win-x64/
    to: jre/
    filter: ["**"]

win:
  target: nsis
  icon: build-assets/icon.ico
  artifactName: MagnetPlay-Setup-${version}.exe

linux:
  target:
    - AppImage
    - deb
    - rpm
  icon: build-assets/icons/
  category: AudioVideo
  artifactName: MagnetPlay-${version}-${arch}.${ext}
  executableName: magnetplay

nsis:
  oneClick: false
  allowToChangeInstallationDirectory: true
  createDesktopShortcut: true
  createStartMenuShortcut: true

publish:
  provider: github
  owner: devmandali2000
  repo: MagnetPlay
  releaseType: draft
```

Note: `extraResources` uses platform-specific `from:` paths — electron-builder is invoked separately per platform in CI, so each build only includes its own platform's binaries.

### 6.2 JRE — jlink Minimal Runtime

CI builds a stripped JRE per platform using `jlink`:

```bash
jlink \
  --add-modules java.base,java.logging,java.net.http,java.xml,\
java.naming,java.management,java.instrument,java.security.jgss,\
jdk.crypto.ec,jdk.unsupported \
  --no-header-files \
  --no-man-pages \
  --compress=2 \
  --output resources/jre/${PLATFORM}
```

Result: ~85MB per platform vs ~320MB full JRE.

### 6.3 Linux Execute Permissions

electron-builder does not preserve execute bits on `extraResources`. Electron main sets permissions on first launch:

```typescript
if (!IS_WIN) {
  fs.chmodSync(goBinary, 0o755);
  fs.chmodSync(javaExe, 0o755);
}
```

---

## 7. GitHub Actions Workflows

### 7.1 `build.yml` — PR merge → draft release

```yaml
on:
  push:
    branches: [master]

jobs:
  build-go:
    runs-on: ubuntu-latest
    steps:
      - cross-compile Go for win-x64 + linux-x64
      - upload-artifact: go-binaries

  build-spring:
    runs-on: ubuntu-latest
    steps:
      - mvn clean package -DskipTests
      - upload-artifact: spring-jar

  build-jre-win:
    runs-on: windows-latest
    steps:
      - jlink → resources/jre/win-x64/
      - upload-artifact: jre-win

  build-jre-linux:
    runs-on: ubuntu-latest
    steps:
      - jlink → resources/jre/linux-x64/
      - upload-artifact: jre-linux

  build-frontend:
    runs-on: ubuntu-latest
    steps:
      - npm ci && npm run build (in frontend/)
      - upload-artifact: frontend-dist

  package-linux:
    needs: [build-go, build-spring, build-jre-linux, build-frontend]
    runs-on: ubuntu-latest
    steps:
      - download all artifacts
      - npm ci (root) + tsc (electron/)
      - electron-builder --linux AppImage deb rpm
      - upload-artifact: linux-packages

  package-win:
    needs: [build-go, build-spring, build-jre-win, build-frontend]
    runs-on: windows-latest
    steps:
      - download all artifacts
      - npm ci (root) + tsc (electron/)
      - electron-builder --win nsis
      - upload-artifact: win-packages

  draft-release:
    needs: [package-linux, package-win]
    runs-on: ubuntu-latest
    steps:
      - bump patch version in package.json + commit to master
      - gh release create v$(version) --draft
          --title "MagnetPlay v$(version)"
          --generate-notes
          attach: all built artifacts
```

### 7.2 `release.yml` — `v*` tag → publish

```yaml
on:
  push:
    tags: ['v*']

jobs:
  promote:
    runs-on: ubuntu-latest
    steps:
      - gh release edit ${{ github.ref_name }} --draft=false
```

Pushing a `v*` tag promotes the matching draft release to published. `electron-updater` on user machines checks GitHub Releases and skips drafts — promotion is what triggers auto-update delivery.

**Required secret**: `GH_TOKEN` with `contents:write` scope.

---

## 8. Auto-Update

### 8.1 electron-updater Integration

```typescript
// electron/updater.ts
import { autoUpdater } from 'electron-updater';

export function initUpdater(mainWindow: BrowserWindow): void {
  autoUpdater.autoDownload = true;
  autoUpdater.autoInstallOnAppQuit = true;

  autoUpdater.on('update-available', (info) => {
    mainWindow.webContents.send('update:available', { version: info.version });
  });

  autoUpdater.on('update-downloaded', (info) => {
    mainWindow.webContents.send('update:downloaded', { version: info.version });
  });

  // Check on startup (delayed) + every 4 hours
  setTimeout(() => autoUpdater.checkForUpdates(), 10_000);
  setInterval(() => autoUpdater.checkForUpdates(), 4 * 60 * 60 * 1000);
}
```

### 8.2 Per-Platform Behavior

| Platform | Mechanism | User experience |
|----------|-----------|-----------------|
| Windows NSIS | Full installer re-runs silently in background | Banner → "Restart to update" prompt |
| Linux AppImage | Differential blockmap update, replaces AppImage in-place | Banner → "Restart to update" prompt |
| Linux deb/rpm | electron-updater not supported | Banner with link to GitHub Releases page |

### 8.3 User-Facing Update UI

- `update:available` → show dismissible banner in renderer: "Update v1.2.3 available, downloading..."
- `update:downloaded` → banner changes to: "Ready to install — Restart now?"
- "Restart now" button → IPC `app:install-update` → `autoUpdater.quitAndInstall(false, true)`
- deb/rpm: banner shows "Update v1.2.3 available" + "Download" link to GitHub Release

---

## 9. Data Isolation

All runtime data stored under `app.getPath('userData')`, never next to the installed binary:

```
userData/                          # platform default:
├── downloads/                     # Windows: %APPDATA%/MagnetPlay
├── logs/                          # Linux:   ~/.config/MagnetPlay
│   ├── go-server.log
│   └── spring.log
├── prowlarr-data/                 # Prowlarr config + DB
└── bin/                           # FFmpeg/FFprobe (downloaded by Go server)
```

Go server receives `DATA_DIR` env pointing to `userData/downloads`. Prowlarr and FFmpeg bin dirs also passed as env vars so Go server writes there instead of relative paths.

Clean uninstall: remove installed app + delete `userData` directory. No registry pollution on Windows beyond the standard NSIS uninstaller entry.

---

## 10. Development Workflow

```bash
# Terminal 1 — Vite dev server
cd frontend && npm run dev

# Terminal 2 — Go sidecar (existing workflow unchanged)
cd backend && make -f MakeFile run

# Terminal 3 — Spring Boot (existing workflow unchanged)
cd backend/mp-spring && ./mvnw spring-boot:run

# Terminal 4 — Electron (loads :5173, does NOT spawn services when DEV=true)
npm run electron:dev
```

`DEV=true` (or `app.isPackaged === false`) makes ServiceManager skip spawning Go + Spring Boot — assumes they are already running manually. This preserves existing developer workflow entirely.

---

## 11. Approximate Installer Sizes

| Platform | Components | Estimated size |
|----------|-----------|----------------|
| Windows NSIS | Electron + React + Go binary + Spring JAR + jlink JRE | ~280MB |
| Linux AppImage | Same | ~260MB |
| Linux deb | Same | ~255MB |
| Linux rpm | Same | ~255MB |

Prowlarr (~80MB) and FFmpeg (~60MB) downloaded on first use of search/HLS features — not included in installer.
