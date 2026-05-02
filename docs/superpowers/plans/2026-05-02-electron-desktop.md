# Electron Desktop App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wrap the existing MagnetPlay services (Go sidecar + Spring Boot) in an Electron shell that manages process lifecycle, ships as a native installer for Windows and Linux, and auto-updates via GitHub Releases.

**Architecture:** Electron main process spawns Go binary and Spring Boot (via bundled jlink JRE) as child processes, polls their health before showing the BrowserWindow, and monitors them for crashes. The React frontend is served from `file://` in production and from Vite dev server in development. GitHub Actions builds all artifacts and creates draft releases on every master push; a `v*` tag push promotes the draft to published and triggers auto-update on user machines.

**Tech Stack:** Electron 33, electron-builder 25, electron-updater 6, TypeScript 5, Node `child_process`/`net`/`http`, Go 1.22 (cross-compile), Java 17 + jlink, GitHub Actions.

---

## File Map

### New files
| Path | Responsibility |
|------|---------------|
| `package.json` (root) | Electron deps, build scripts, electron-builder config pointer |
| `tsconfig.electron.json` | TypeScript config for `electron/` → compiles to `electron/dist/` |
| `electron-builder.yml` | Build targets, extraResources, publish config |
| `build-assets/icon.png` | App icon (512×512 PNG, all platforms) |
| `electron/main.ts` | App entry: BrowserWindow, IPC handlers, orchestrates service-manager + updater |
| `electron/preload.ts` | contextBridge — exposes `window.electronAPI` to renderer |
| `electron/service-manager.ts` | Spawns Go + Spring Boot, health polls, crash recovery, graceful stop |
| `electron/updater.ts` | electron-updater init, check schedule, push events to renderer |
| `frontend/src/electron-api.d.ts` | `Window.electronAPI` type declarations for renderer TypeScript |
| `frontend/src/components/UpdateBanner.tsx` | Dismissible update notification banner |
| `.github/workflows/build.yml` | PR merge → build all artifacts + create draft GitHub Release |
| `.github/workflows/release.yml` | `v*` tag push → promote draft release to published |

### Modified files
| Path | Change |
|------|--------|
| `backend/go-server/config/config.go` | Add `FFmpegBinDir` field + `FromEnv()` that reads env overrides |
| `backend/go-server/main.go` | `config.Default()` → `config.FromEnv()` |
| `backend/go-server/internal/grpc/server.go` | `ffmpegBinDir := "./bin"` → `ffmpegBinDir := cfg.FFmpegBinDir` |
| `frontend/vite.config.ts` | Add `base: './'` so asset paths work under `file://` |
| `frontend/src/App.tsx` | Register IPC listeners, render `<UpdateBanner>` |
| `.gitignore` | Add `resources/`, `dist-electron/`, `electron/dist/` |

---

## Task 1: Root project scaffolding

**Files:**
- Create: `package.json` (root)
- Create: `tsconfig.electron.json`
- Modify: `.gitignore`

- [ ] **Step 1: Create root `package.json`**

```json
{
  "name": "magnetplay",
  "version": "0.1.0",
  "description": "P2P video streaming desktop app",
  "main": "electron/dist/main.js",
  "private": true,
  "scripts": {
    "build:electron": "tsc -p tsconfig.electron.json",
    "build:frontend": "cd frontend && npm run build",
    "build": "npm run build:frontend && npm run build:electron",
    "electron:dev": "tsc -p tsconfig.electron.json && electron .",
    "dist:linux": "npm run build && electron-builder --linux AppImage deb rpm",
    "dist:win": "npm run build && electron-builder --win nsis"
  },
  "dependencies": {
    "electron-updater": "^6.3.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "electron": "^33.0.0",
    "electron-builder": "^25.0.0",
    "typescript": "^5.5.0"
  }
}
```

- [ ] **Step 2: Create `tsconfig.electron.json`**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": ["ES2020"],
    "outDir": "electron/dist",
    "rootDir": "electron",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "resolveJsonModule": true
  },
  "include": ["electron/**/*.ts"],
  "exclude": ["node_modules"]
}
```

- [ ] **Step 3: Update `.gitignore`** — append these lines (create the file if it doesn't exist at repo root):

```
# Electron
dist-electron/
electron/dist/

# CI-populated binaries — never committed
resources/
```

- [ ] **Step 4: Install root dependencies**

Run from repo root:
```bash
npm install
```

Expected: `node_modules/` created at repo root, `electron` and `electron-builder` present.

- [ ] **Step 5: Commit**

```bash
git add package.json tsconfig.electron.json .gitignore
git commit -m "chore(electron): root scaffolding — package.json, tsconfig, gitignore"
```

---

## Task 2: electron-builder config + build assets

**Files:**
- Create: `electron-builder.yml`
- Create: `build-assets/icon.png`

- [ ] **Step 1: Create `build-assets/` directory and placeholder icon**

Place a 512×512 PNG at `build-assets/icon.png`. This is the app icon for all platforms — replace with the real icon before shipping. For a placeholder you can use any 512×512 PNG renamed to `icon.png`.

On Linux/CI you can generate a solid-color placeholder:
```bash
convert -size 512x512 xc:#1a6b2a build-assets/icon.png
```
On Windows, any image editing tool or online converter works.

- [ ] **Step 2: Create `electron-builder.yml`**

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
  - from: resources/go-server/current/
    to: go-server/
    filter: ["**"]
  - from: resources/spring/
    to: spring/
    filter: ["*.jar"]
  - from: resources/jre/current/
    to: jre/
    filter: ["**"]

win:
  target: nsis
  icon: build-assets/icon.png
  artifactName: MagnetPlay-Setup-${version}.exe

linux:
  target:
    - AppImage
    - deb
    - rpm
  icon: build-assets/icon.png
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

Note: `from: resources/go-server/current/` and `from: resources/jre/current/` — CI jobs normalize the platform-specific directory to `current/` before running electron-builder (e.g. `mv resources/jre/linux-x64 resources/jre/current`).

- [ ] **Step 3: Commit**

```bash
git add electron-builder.yml build-assets/icon.png
git commit -m "chore(electron): electron-builder config and placeholder icon"
```

---

## Task 3: `electron/preload.ts` + renderer type declarations

**Files:**
- Create: `electron/preload.ts`
- Create: `frontend/src/electron-api.d.ts`

- [ ] **Step 1: Create `electron/preload.ts`**

```typescript
import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  onUpdateAvailable: (cb: (data: { version: string }) => void) =>
    ipcRenderer.on('update:available', (_event, data) => cb(data)),
  onUpdateDownloaded: (cb: (data: { version: string }) => void) =>
    ipcRenderer.on('update:downloaded', (_event, data) => cb(data)),
  onServiceStatusChanged: (cb: (data: { service: string; status: string }) => void) =>
    ipcRenderer.on('service:status-changed', (_event, data) => cb(data)),
  installUpdate: (): Promise<void> => ipcRenderer.invoke('app:install-update'),
  quit: (): Promise<void> => ipcRenderer.invoke('app:quit'),
  getVersion: (): Promise<string> => ipcRenderer.invoke('app:get-version'),
  openLogsDir: (): Promise<void> => ipcRenderer.invoke('app:open-logs-dir'),
  getServiceStatus: (): Promise<{ go: string; spring: string }> =>
    ipcRenderer.invoke('app:service-status'),
});
```

- [ ] **Step 2: Create `frontend/src/electron-api.d.ts`**

```typescript
interface ElectronAPI {
  onUpdateAvailable: (cb: (data: { version: string }) => void) => void;
  onUpdateDownloaded: (cb: (data: { version: string }) => void) => void;
  onServiceStatusChanged: (cb: (data: { service: string; status: string }) => void) => void;
  installUpdate: () => Promise<void>;
  quit: () => Promise<void>;
  getVersion: () => Promise<string>;
  openLogsDir: () => Promise<void>;
  getServiceStatus: () => Promise<{ go: string; spring: string }>;
}

declare global {
  interface Window {
    electronAPI?: ElectronAPI;
  }
}

export {};
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
tsc -p tsconfig.electron.json --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add electron/preload.ts frontend/src/electron-api.d.ts
git commit -m "feat(electron): preload contextBridge IPC surface"
```

---

## Task 4: `electron/service-manager.ts`

**Files:**
- Create: `electron/service-manager.ts`

- [ ] **Step 1: Create `electron/service-manager.ts`**

```typescript
import { spawn, ChildProcess, execSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import * as net from 'net';
import * as http from 'http';
import { app, BrowserWindow } from 'electron';

const IS_WIN = process.platform === 'win32';

export type ServiceStatus = 'starting' | 'running' | 'crashed' | 'stopped';

interface ServiceState {
  process: ChildProcess | null;
  status: ServiceStatus;
  restartCount: number;
  stableTimer: ReturnType<typeof setTimeout> | null;
}

function makeState(): ServiceState {
  return { process: null, status: 'stopped', restartCount: 0, stableTimer: null };
}

export class ServiceManager {
  private userData: string;
  private goState: ServiceState = makeState();
  private springState: ServiceState = makeState();
  private mainWindow: BrowserWindow | null = null;

  constructor(userData: string) {
    this.userData = userData;
  }

  getStatus(): { go: ServiceStatus; spring: ServiceStatus } {
    return { go: this.goState.status, spring: this.springState.status };
  }

  async start(mainWindow: BrowserWindow): Promise<void> {
    this.mainWindow = mainWindow;
    await fs.promises.mkdir(path.join(this.userData, 'logs'), { recursive: true });
    await fs.promises.mkdir(path.join(this.userData, 'downloads'), { recursive: true });

    this.spawnGo();
    await waitForPort(50051, 30_000);
    this.goState.status = 'running';

    this.spawnSpring();
    await waitForHttp('http://localhost:8080/actuator/health', 45_000);
    this.springState.status = 'running';
  }

  async stop(): Promise<void> {
    // Spring first — it depends on Go via gRPC
    const springProc = this.springState.process;
    this.springState.status = 'stopped';
    await killProc(springProc);
    this.springState.process = null;

    // Go last — its defer handles Prowlarr + FFmpeg children
    const goProc = this.goState.process;
    this.goState.status = 'stopped';
    await killProc(goProc);
    this.goState.process = null;
  }

  private spawnGo(): void {
    const bin = getResourcePath('go-server', IS_WIN ? 'server.exe' : 'server');
    if (!IS_WIN) safeChmod(bin, 0o755);

    const logStream = fs.createWriteStream(
      path.join(this.userData, 'logs', 'go-server.log'),
      { flags: 'a' }
    );

    const proc = spawn(bin, [], {
      env: {
        ...process.env,
        DATA_DIR: path.join(this.userData, 'downloads'),
        PROWLARR_DATA_DIR: path.join(this.userData, 'prowlarr-data'),
        PROWLARR_BIN_DIR: path.join(this.userData, 'prowlarr'),
        FFMPEG_BIN_DIR: path.join(this.userData, 'bin'),
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    proc.stdout?.pipe(logStream);
    proc.stderr?.pipe(logStream);
    this.goState.process = proc;
    this.goState.status = 'starting';
    this.watchProcess('go', proc, () => this.spawnGo());
  }

  private spawnSpring(): void {
    const java = getResourcePath('jre', 'bin', IS_WIN ? 'java.exe' : 'java');
    const jar = getResourcePath('spring', 'mp-spring.jar');
    if (!IS_WIN) safeChmod(java, 0o755);

    const logStream = fs.createWriteStream(
      path.join(this.userData, 'logs', 'spring.log'),
      { flags: 'a' }
    );

    const proc = spawn(java, ['-jar', jar], {
      env: {
        ...process.env,
        GRPC_TORRENT_SERVICE_ADDRESS: 'localhost:50051',
        SERVER_PORT: '8080',
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    proc.stdout?.pipe(logStream);
    proc.stderr?.pipe(logStream);
    this.springState.process = proc;
    this.springState.status = 'starting';
    this.watchProcess('spring', proc, () => this.spawnSpring());
  }

  private watchProcess(
    name: 'go' | 'spring',
    proc: ChildProcess,
    respawn: () => void
  ): void {
    const state = name === 'go' ? this.goState : this.springState;

    state.stableTimer = setTimeout(() => {
      state.restartCount = 0;
    }, 60_000);

    proc.on('exit', (_code, _signal) => {
      if (state.stableTimer) clearTimeout(state.stableTimer);
      if (state.status === 'stopped') return; // intentional shutdown

      state.status = 'crashed';
      this.mainWindow?.webContents.send('service:status-changed', {
        service: name,
        status: 'crashed',
      });

      if (state.restartCount < 3) {
        state.restartCount++;
        setTimeout(respawn, 2_000);
      }
      // After 3 failures, status stays 'crashed'.
      // main.ts watches for this and shows an error dialog.
    });
  }
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function getResourcePath(...parts: string[]): string {
  // Only ever called in packaged mode — dev mode skips service spawning entirely.
  return path.join(process.resourcesPath, ...parts);
}

function safeChmod(filePath: string, mode: number): void {
  try { fs.chmodSync(filePath, mode); } catch { /* binary may not exist in dev */ }
}

function waitForPort(port: number, timeoutMs: number): Promise<void> {
  return new Promise((resolve, reject) => {
    const deadline = Date.now() + timeoutMs;
    function attempt() {
      const socket = new net.Socket();
      socket.setTimeout(500);
      socket.connect(port, '127.0.0.1', () => { socket.destroy(); resolve(); });
      socket.on('error', () => { socket.destroy(); retry(); });
      socket.on('timeout', () => { socket.destroy(); retry(); });
    }
    function retry() {
      if (Date.now() < deadline) setTimeout(attempt, 500);
      else reject(new Error(`Port ${port} not ready after ${timeoutMs}ms`));
    }
    attempt();
  });
}

function waitForHttp(url: string, timeoutMs: number): Promise<void> {
  return new Promise((resolve, reject) => {
    const deadline = Date.now() + timeoutMs;
    function attempt() {
      http.get(url, (res) => {
        res.resume();
        if (res.statusCode === 200) resolve();
        else retry();
      }).on('error', retry);
    }
    function retry() {
      if (Date.now() < deadline) setTimeout(attempt, 500);
      else reject(new Error(`${url} not ready after ${timeoutMs}ms`));
    }
    attempt();
  });
}

async function killProc(proc: ChildProcess | null): Promise<void> {
  if (!proc || proc.exitCode !== null) return;
  return new Promise((resolve) => {
    proc.once('exit', resolve);
    if (IS_WIN) {
      try { execSync(`taskkill /F /T /PID ${proc.pid}`); } catch { resolve(); }
    } else {
      proc.kill('SIGTERM');
      setTimeout(() => {
        if (proc.exitCode === null) proc.kill('SIGKILL');
      }, 8_000);
    }
  });
}
```

- [ ] **Step 2: Type-check**

```bash
tsc -p tsconfig.electron.json --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add electron/service-manager.ts
git commit -m "feat(electron): ServiceManager — spawn, health-poll, crash recovery"
```

---

## Task 5: `electron/updater.ts`

**Files:**
- Create: `electron/updater.ts`

- [ ] **Step 1: Create `electron/updater.ts`**

```typescript
import { BrowserWindow } from 'electron';
import { autoUpdater } from 'electron-updater';

export function initUpdater(mainWindow: BrowserWindow): void {
  autoUpdater.autoDownload = true;
  autoUpdater.autoInstallOnAppQuit = true;

  autoUpdater.on('update-available', (info: { version: string }) => {
    mainWindow.webContents.send('update:available', { version: info.version });
  });

  autoUpdater.on('update-downloaded', (info: { version: string }) => {
    mainWindow.webContents.send('update:downloaded', { version: info.version });
  });

  autoUpdater.on('error', (err: Error) => {
    console.error('[updater] error:', err.message);
  });

  // Delay first check — let services fully start first
  setTimeout(() => autoUpdater.checkForUpdates(), 10_000);
  setInterval(() => autoUpdater.checkForUpdates(), 4 * 60 * 60 * 1000);
}
```

- [ ] **Step 2: Type-check**

```bash
tsc -p tsconfig.electron.json --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add electron/updater.ts
git commit -m "feat(electron): auto-updater via electron-updater + GitHub Releases"
```

---

## Task 6: `electron/main.ts`

**Files:**
- Create: `electron/main.ts`

- [ ] **Step 1: Create `electron/main.ts`**

```typescript
import { app, BrowserWindow, ipcMain, dialog, shell } from 'electron';
import * as path from 'path';
import { ServiceManager } from './service-manager';
import { initUpdater } from './updater';
import { autoUpdater } from 'electron-updater';

const isDev = !app.isPackaged;
let mainWindow: BrowserWindow | null = null;
let serviceManager: ServiceManager | null = null;
let isQuitting = false;

async function createWindow(): Promise<void> {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    show: false,
    title: 'MagnetPlay',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  // CSP: allow API calls to Spring Boot (:8080) and HLS server (:8091), block everything else
  mainWindow.webContents.session.webRequest.onHeadersReceived((details, callback) => {
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        'Content-Security-Policy': [
          "default-src 'self' file:; " +
          "connect-src 'self' http://localhost:8080 http://localhost:8091 ws://localhost:8080; " +
          "media-src 'self' http://localhost:8080 http://localhost:8091 blob:; " +
          "script-src 'self'; " +
          "style-src 'self' 'unsafe-inline';"
        ],
      },
    });
  });

  registerIpcHandlers();

  if (isDev) {
    // Dev: services started manually — just open the Vite dev server
    mainWindow.loadURL('http://localhost:5173');
    mainWindow.webContents.openDevTools();
    mainWindow.show();
  } else {
    serviceManager = new ServiceManager(app.getPath('userData'));
    try {
      await serviceManager.start(mainWindow);
    } catch (err) {
      dialog.showErrorBox(
        'MagnetPlay — Startup Failed',
        `Services could not start.\n\n${err}\n\nLogs: ${path.join(app.getPath('userData'), 'logs')}`
      );
      app.quit();
      return;
    }
    mainWindow.loadFile(
      path.join(__dirname, '../../frontend/dist/index.html')
    );
    mainWindow.show();
    initUpdater(mainWindow);
  }

  mainWindow.on('closed', () => { mainWindow = null; });
}

function registerIpcHandlers(): void {
  ipcMain.handle('app:quit', () => app.quit());

  ipcMain.handle('app:get-version', () => app.getVersion());

  ipcMain.handle('app:open-logs-dir', () =>
    shell.openPath(path.join(app.getPath('userData'), 'logs'))
  );

  ipcMain.handle('app:service-status', () =>
    serviceManager?.getStatus() ?? { go: 'unknown', spring: 'unknown' }
  );

  ipcMain.handle('app:install-update', () =>
    autoUpdater.quitAndInstall(false, true)
  );
}

app.whenReady().then(createWindow);

app.on('before-quit', (event) => {
  if (!isQuitting && serviceManager) {
    event.preventDefault();
    isQuitting = true;
    serviceManager.stop().finally(() => app.quit());
  }
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});
```

- [ ] **Step 2: Type-check**

```bash
tsc -p tsconfig.electron.json --noEmit
```

Expected: no errors.

- [ ] **Step 3: Compile**

```bash
npm run build:electron
```

Expected: `electron/dist/main.js`, `electron/dist/preload.js`, `electron/dist/service-manager.js`, `electron/dist/updater.js` all created.

- [ ] **Step 4: Commit**

```bash
git add electron/main.ts
git commit -m "feat(electron): main process — BrowserWindow, IPC handlers, lifecycle"
```

---

## Task 7: Go server — env-driven paths

**Files:**
- Modify: `backend/go-server/config/config.go`
- Modify: `backend/go-server/main.go`
- Modify: `backend/go-server/internal/grpc/server.go`

- [ ] **Step 1: Update `backend/go-server/config/config.go`**

Replace the entire file:

```go
package config

import (
	"os"
	"time"
)

type ProwlarrConfig struct {
	DataDir      string
	BinDir       string
	Port         int
	SeedIndexers bool
}

type Config struct {
	GRPCPort        int
	DataDir         string
	MetadataTimeout time.Duration
	Prowlarr        ProwlarrConfig
	FFmpegPath      string
	FFprobePath     string
	FFmpegBinDir    string
	HLSPort         int
}

func Default() Config {
	return Config{
		GRPCPort:        50051,
		DataDir:         "./downloads",
		MetadataTimeout: 60 * time.Second,
		Prowlarr: ProwlarrConfig{
			DataDir:      "./prowlarr-data",
			BinDir:       "./prowlarr",
			Port:         9696,
			SeedIndexers: true,
		},
		FFmpegBinDir: "./bin",
		HLSPort:      8091,
	}
}

// FromEnv returns Default() with path overrides from environment variables.
// Electron sets these to subdirectories of app.getPath('userData').
func FromEnv() Config {
	cfg := Default()
	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("PROWLARR_DATA_DIR"); v != "" {
		cfg.Prowlarr.DataDir = v
	}
	if v := os.Getenv("PROWLARR_BIN_DIR"); v != "" {
		cfg.Prowlarr.BinDir = v
	}
	if v := os.Getenv("FFMPEG_BIN_DIR"); v != "" {
		cfg.FFmpegBinDir = v
	}
	return cfg
}
```

- [ ] **Step 2: Update `backend/go-server/main.go`** — change `config.Default()` to `config.FromEnv()`

```go
package main

import (
	"io"
	"log"
	"os"

	"gopkg.in/lumberjack.v2"

	"server/config"
	server "server/internal/grpc"
)

func main() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("cannot create logs dir: %v", err)
	}
	roller := &lumberjack.Logger{
		Filename:   "logs/go-server.log",
		MaxSize:    20,
		MaxBackups: 7,
		MaxAge:     14,
		Compress:   true,
	}
	log.SetOutput(io.MultiWriter(os.Stdout, roller))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("[startup] logging to stdout + logs/go-server.log")

	server.StartServer(config.FromEnv())
}
```

- [ ] **Step 3: Update `backend/go-server/internal/grpc/server.go`** — replace the hardcoded `"./bin"` with `cfg.FFmpegBinDir`

Find this block (lines 38–48 of current file):

```go
	// Ensure FFmpeg binaries — fatal if unavailable
	ffmpegBinDir := "./bin"
	ffmpegPath, err := ffmpeg.EnsureFFmpeg(cfg.FFmpegPath, ffmpegBinDir)
```

Replace with:

```go
	// Ensure FFmpeg binaries — fatal if unavailable
	ffmpegBinDir := cfg.FFmpegBinDir
	ffmpegPath, err := ffmpeg.EnsureFFmpeg(cfg.FFmpegPath, ffmpegBinDir)
```

- [ ] **Step 4: Verify Go compiles and tests pass**

```bash
cd backend/go-server
go build ./...
go vet ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add backend/go-server/config/config.go backend/go-server/main.go backend/go-server/internal/grpc/server.go
git commit -m "feat(go): env-driven data paths for Electron userData isolation"
```

---

## Task 8: Frontend — Vite base path + UpdateBanner + App.tsx wiring

**Files:**
- Modify: `frontend/vite.config.ts`
- Create: `frontend/src/components/UpdateBanner.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Update `frontend/vite.config.ts`** — add `base: './'`

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  base: './',
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/subtitle': {
        target: 'http://localhost:8091',
        changeOrigin: true,
      },
      '/remux': {
        target: 'http://localhost:8091',
        changeOrigin: true,
      },
    },
  },
});
```

- [ ] **Step 2: Create `frontend/src/components/UpdateBanner.tsx`**

```tsx
interface Props {
  status: 'available' | 'downloaded' | null;
  version: string;
  onInstall: () => void;
  onDismiss: () => void;
}

export default function UpdateBanner({ status, version, onInstall, onDismiss }: Props) {
  if (!status) return null;

  const bg = status === 'downloaded' ? '#1a6b2a' : '#1a3d6b';

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, zIndex: 9999,
      background: bg, color: '#fff',
      padding: '8px 16px', display: 'flex', alignItems: 'center', gap: 12,
    }}>
      {status === 'available' && (
        <span>Update v{version} available — downloading in background…</span>
      )}
      {status === 'downloaded' && (
        <>
          <span>v{version} ready to install</span>
          <button
            onClick={onInstall}
            style={{ padding: '4px 12px', cursor: 'pointer', borderRadius: 4 }}
          >
            Restart now
          </button>
        </>
      )}
      <button
        onClick={onDismiss}
        style={{
          marginLeft: 'auto', background: 'none', border: 'none',
          color: '#fff', cursor: 'pointer', fontSize: 20, lineHeight: 1,
        }}
        aria-label="Dismiss"
      >
        ×
      </button>
    </div>
  );
}
```

- [ ] **Step 3: Add update state + IPC wiring to `frontend/src/App.tsx`**

Add these imports at the top of `frontend/src/App.tsx` (after existing imports):

```typescript
import UpdateBanner from './components/UpdateBanner';
```

Add these state declarations inside the `App` component (after existing `useState` calls):

```typescript
  const [updateStatus, setUpdateStatus] = useState<'available' | 'downloaded' | null>(null);
  const [updateVersion, setUpdateVersion] = useState('');
```

Add this `useEffect` inside the `App` component (after existing effects):

```typescript
  useEffect(() => {
    if (!window.electronAPI) return;
    window.electronAPI.onUpdateAvailable(({ version }) => {
      setUpdateVersion(version);
      setUpdateStatus('available');
    });
    window.electronAPI.onUpdateDownloaded(({ version }) => {
      setUpdateVersion(version);
      setUpdateStatus('downloaded');
    });
  }, []);
```

Add `<UpdateBanner>` as the first child inside the JSX return (before any existing elements in the return statement):

```tsx
      <UpdateBanner
        status={updateStatus}
        version={updateVersion}
        onInstall={() => window.electronAPI?.installUpdate()}
        onDismiss={() => setUpdateStatus(null)}
      />
```

- [ ] **Step 4: Verify frontend TypeScript compiles**

```bash
cd frontend
npm run build
```

Expected: `frontend/dist/` produced with no TypeScript errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/vite.config.ts frontend/src/components/UpdateBanner.tsx frontend/src/App.tsx
git commit -m "feat(frontend): file:// base path, UpdateBanner, Electron IPC wiring"
```

---

## Task 9: `.github/workflows/build.yml`

**Files:**
- Create: `.github/workflows/build.yml`

- [ ] **Step 1: Create `.github/workflows/build.yml`**

```yaml
name: Build & Draft Release

on:
  push:
    branches: [master]

jobs:
  build-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Cross-compile Go sidecar
        run: |
          cd backend/go-server
          mkdir -p ../../resources/go-server/win-x64
          mkdir -p ../../resources/go-server/linux-x64
          GOOS=windows GOARCH=amd64 go build -o ../../resources/go-server/win-x64/server.exe .
          GOOS=linux   GOARCH=amd64 go build -o ../../resources/go-server/linux-x64/server .
      - uses: actions/upload-artifact@v4
        with:
          name: go-binaries
          path: resources/go-server/

  build-spring:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-java@v4
        with:
          java-version: '17'
          distribution: 'temurin'
      - name: Build Spring Boot fat JAR
        run: |
          cd backend/mp-spring
          ./mvnw clean package -DskipTests
          mkdir -p ../../resources/spring
          cp target/mp-spring-*.jar ../../resources/spring/mp-spring.jar
      - uses: actions/upload-artifact@v4
        with:
          name: spring-jar
          path: resources/spring/

  build-jre-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-java@v4
        with:
          java-version: '17'
          distribution: 'temurin'
      - name: Build jlink JRE for Linux
        run: |
          mkdir -p resources/jre/linux-x64
          jlink \
            --add-modules java.base,java.logging,java.net.http,java.xml,\
          java.naming,java.management,java.instrument,java.security.jgss,\
          jdk.crypto.ec,jdk.unsupported \
            --no-header-files --no-man-pages --compress=2 \
            --output resources/jre/linux-x64
      - uses: actions/upload-artifact@v4
        with:
          name: jre-linux
          path: resources/jre/

  build-jre-win:
    runs-on: windows-latest
    steps:
      - uses: actions/setup-java@v4
        with:
          java-version: '17'
          distribution: 'temurin'
      - name: Build jlink JRE for Windows
        shell: bash
        run: |
          mkdir -p resources/jre/win-x64
          jlink \
            --add-modules java.base,java.logging,java.net.http,java.xml,\
          java.naming,java.management,java.instrument,java.security.jgss,\
          jdk.crypto.ec,jdk.unsupported \
            --no-header-files --no-man-pages --compress=2 \
            --output resources/jre/win-x64
      - uses: actions/upload-artifact@v4
        with:
          name: jre-win
          path: resources/jre/

  build-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Build React frontend
        run: |
          cd frontend
          npm ci
          npm run build
      - uses: actions/upload-artifact@v4
        with:
          name: frontend-dist
          path: frontend/dist/

  package-linux:
    needs: [build-go, build-spring, build-jre-linux, build-frontend]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - uses: actions/download-artifact@v4
        with:
          name: go-binaries
          path: resources/go-server/
      - uses: actions/download-artifact@v4
        with:
          name: spring-jar
          path: resources/spring/
      - uses: actions/download-artifact@v4
        with:
          name: jre-linux
          path: resources/jre/
      - uses: actions/download-artifact@v4
        with:
          name: frontend-dist
          path: frontend/dist/
      - name: Normalize platform dir for electron-builder
        run: |
          mv resources/go-server/linux-x64 resources/go-server/current
          mv resources/jre/linux-x64 resources/jre/current
      - name: Build Linux packages
        run: |
          npm ci
          npm run build:electron
          npx electron-builder --linux AppImage deb rpm --publish never
      - uses: actions/upload-artifact@v4
        with:
          name: linux-packages
          path: dist-electron/

  package-win:
    needs: [build-go, build-spring, build-jre-win, build-frontend]
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - uses: actions/download-artifact@v4
        with:
          name: go-binaries
          path: resources/go-server/
      - uses: actions/download-artifact@v4
        with:
          name: spring-jar
          path: resources/spring/
      - uses: actions/download-artifact@v4
        with:
          name: jre-win
          path: resources/jre/
      - uses: actions/download-artifact@v4
        with:
          name: frontend-dist
          path: frontend/dist/
      - name: Normalize platform dir for electron-builder
        shell: bash
        run: |
          mv resources/go-server/win-x64 resources/go-server/current
          mv resources/jre/win-x64 resources/jre/current
      - name: Build Windows installer
        run: |
          npm ci
          npm run build:electron
          npx electron-builder --win nsis --publish never
      - uses: actions/upload-artifact@v4
        with:
          name: win-packages
          path: dist-electron/

  draft-release:
    needs: [package-linux, package-win]
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          token: ${{ secrets.GH_TOKEN }}
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - uses: actions/download-artifact@v4
        with:
          name: linux-packages
          path: dist-electron/
      - uses: actions/download-artifact@v4
        with:
          name: win-packages
          path: dist-electron/
      - name: Bump patch version
        id: bump
        run: |
          npm version patch --no-git-tag-version
          echo "version=$(node -p "require('./package.json').version")" >> $GITHUB_OUTPUT
      - name: Commit version bump
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add package.json
          git commit -m "chore: bump version to v${{ steps.bump.outputs.version }}"
          git push
      - name: Create draft release
        run: |
          gh release create "v${{ steps.bump.outputs.version }}" \
            --title "MagnetPlay v${{ steps.bump.outputs.version }}" \
            --generate-notes \
            --draft \
            dist-electron/*.exe \
            dist-electron/*.AppImage \
            dist-electron/*.deb \
            dist-electron/*.rpm \
            dist-electron/*.yml
        env:
          GH_TOKEN: ${{ secrets.GH_TOKEN }}
```

- [ ] **Step 2: Add `GH_TOKEN` secret to the GitHub repository**

Go to: GitHub repo → Settings → Secrets and variables → Actions → New repository secret

Name: `GH_TOKEN`
Value: a Personal Access Token with `contents: write` scope (or use the built-in `GITHUB_TOKEN` if the repo permissions allow it).

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/build.yml
git commit -m "ci: build all platforms and draft GitHub Release on master push"
```

---

## Task 10: `.github/workflows/release.yml`

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create `.github/workflows/release.yml`**

```yaml
name: Publish Release

on:
  push:
    tags:
      - 'v*'

jobs:
  promote:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - name: Promote draft release to published
        run: |
          gh release edit "${{ github.ref_name }}" \
            --draft=false \
            --repo "${{ github.repository }}"
        env:
          GH_TOKEN: ${{ secrets.GH_TOKEN }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: publish release when v* tag is pushed"
```

---

## Task 11: Integration smoke test

All code is now in place. Verify the full stack works.

- [ ] **Step 1: Verify Go builds with env override**

```bash
cd backend/go-server
DATA_DIR=/tmp/mp-test go run main.go
```

Expected: server starts, logs show `[startup]` line, listens on `:50051`. Press Ctrl+C to stop.

- [ ] **Step 2: Verify frontend builds correctly for file:// loading**

```bash
cd frontend
npm run build
```

Then open `frontend/dist/index.html` directly in a browser (drag-drop to address bar as `file://`). Expected: React app loads without blank screen or console errors about missing assets.

- [ ] **Step 3: Run Electron in dev mode**

In 3 separate terminals, start the services manually:
```bash
# Terminal 1
cd backend/go-server && go run main.go

# Terminal 2
cd backend/mp-spring && ./mvnw spring-boot:run

# Terminal 3
cd frontend && npm run dev
```

Then in a 4th terminal:
```bash
npm run electron:dev
```

Expected:
- Electron window opens loading `http://localhost:5173`
- DevTools open automatically
- App functions normally (add magnet link, play video)
- No IPC errors in DevTools console

- [ ] **Step 4: Verify TypeScript across all layers**

```bash
# Electron layer
tsc -p tsconfig.electron.json --noEmit

# Frontend layer
cd frontend && npx tsc --noEmit
```

Expected: no errors in either.

- [ ] **Step 5: Push to master and verify CI**

```bash
git push origin feature/torrent-management
# Open a PR and merge to master, or push directly to master if permitted
```

Expected in GitHub Actions:
- `build-go`, `build-spring`, `build-jre-linux`, `build-jre-win`, `build-frontend` run in parallel
- `package-linux` and `package-win` run after their dependencies
- `draft-release` creates a draft release in GitHub Releases with all artifacts attached

- [ ] **Step 6: Verify draft release exists**

Go to GitHub → Releases. Expected: a draft release `MagnetPlay v0.1.x` with these artifacts:
- `MagnetPlay-Setup-0.1.x.exe`
- `MagnetPlay-0.1.x-x86_64.AppImage`
- `MagnetPlay-0.1.x-amd64.deb`
- `MagnetPlay-0.1.x-x86_64.rpm`
- `latest.yml`, `latest-linux.yml` (electron-updater manifests)

- [ ] **Step 7: Test release promotion**

```bash
git tag v0.1.0
git push origin v0.1.0
```

Expected: `release.yml` runs, the draft release is promoted to published.

- [ ] **Step 8: Final commit if any loose changes**

```bash
git status
# Stage any untracked files found above
git commit -m "chore: electron desktop app — complete implementation"
```

---

## Notes

**Dev workflow** (unchanged from before): Run Go, Spring Boot, and Vite manually in 3 terminals. Run `npm run electron:dev` in a 4th. ServiceManager skips spawning services when `app.isPackaged === false`.

**First-run downloads**: Prowlarr (~80MB) and FFmpeg (~60MB) are still downloaded by the Go server on first use of search/HLS features. They are saved to `app.getPath('userData')/prowlarr` and `app.getPath('userData')/bin` respectively via the `PROWLARR_BIN_DIR` and `FFMPEG_BIN_DIR` env vars.

**Installer sizes** (approximate):
- Windows NSIS: ~280MB
- Linux AppImage: ~260MB
- Linux deb/rpm: ~255MB

**Auto-update on Linux deb/rpm**: electron-updater does not support silent deb/rpm updates. The UpdateBanner shows a download link instead of a "Restart now" button for those users.
