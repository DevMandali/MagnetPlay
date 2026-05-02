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
    await waitForHttp('http://localhost:8080/actuator/health', 120_000);
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
    safeChmod(bin, 0o755);

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
      windowsHide: true,
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
    safeChmod(java, 0o755);

    const logStream = fs.createWriteStream(
      path.join(this.userData, 'logs', 'spring.log'),
      { flags: 'a' }
    );

    const proc = spawn(java, ['-jar', jar], {
      env: {
        ...process.env,
        GRPC_TORRENT_SERVICE_ADDRESS: 'localhost:50051',
        SERVER_PORT: '8080',
        PROWLARR_URL: 'http://localhost:9696',
        PROWLARR_CONFIG_XML: path.join(this.userData, 'prowlarr-data', 'config.xml'),
      },
      stdio: ['ignore', 'pipe', 'pipe'],
      windowsHide: true,
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
      // After 3 failures status stays 'crashed' — main.ts shows error dialog
    });
  }
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function getResourcePath(...parts: string[]): string {
  // Only ever called in packaged mode — dev mode skips service spawning entirely.
  return path.join(process.resourcesPath, ...parts);
}

function safeChmod(filePath: string, mode: number): void {
  if (IS_WIN) return;
  try { fs.chmodSync(filePath, mode); } catch { /* ignore in dev/missing */ }
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
