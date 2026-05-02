import { app, BrowserWindow, ipcMain, dialog, shell, Tray, Menu, nativeImage } from 'electron';
import * as path from 'path';
import { ServiceManager } from './service-manager';
import { initUpdater } from './updater';
import { autoUpdater } from 'electron-updater';

const isDev = !app.isPackaged;
let mainWindow: BrowserWindow | null = null;
let splashWindow: BrowserWindow | null = null;
let tray: Tray | null = null;
let serviceManager: ServiceManager | null = null;
let isQuitting = false;

// ── Splash ────────────────────────────────────────────────────────────────────

function createSplash(): void {
  splashWindow = new BrowserWindow({
    width: 340,
    height: 220,
    frame: false,
    transparent: true,
    alwaysOnTop: true,
    resizable: false,
    center: true,
    skipTaskbar: true,
    webPreferences: { nodeIntegration: false, contextIsolation: true },
  });
  splashWindow.loadFile(path.join(__dirname, '../splash.html'));
  splashWindow.on('closed', () => { splashWindow = null; });
}

function closeSplash(): void {
  if (splashWindow && !splashWindow.isDestroyed()) {
    splashWindow.close();
  }
}

// ── Tray ─────────────────────────────────────────────────────────────────────

function createTray(): void {
  const iconPath = path.join(__dirname, '../../build-assets/icon.png');
  const icon = nativeImage.createFromPath(iconPath).resize({ width: 16, height: 16 });
  tray = new Tray(icon);
  tray.setToolTip('MagnetPlay');
  updateTrayMenu();

  tray.on('click', () => {
    if (!mainWindow) return;
    if (mainWindow.isVisible()) {
      mainWindow.focus();
    } else {
      mainWindow.show();
    }
  });
}

function updateTrayMenu(): void {
  if (!tray) return;
  const menu = Menu.buildFromTemplate([
    {
      label: 'Open MagnetPlay',
      click: () => { mainWindow?.show(); mainWindow?.focus(); },
    },
    { type: 'separator' },
    {
      label: 'Open Logs Folder',
      click: () => shell.openPath(path.join(app.getPath('userData'), 'logs')),
    },
    { type: 'separator' },
    {
      label: 'Quit',
      click: () => { isQuitting = true; app.quit(); },
    },
  ]);
  tray.setContextMenu(menu);
}

// ── Main window ───────────────────────────────────────────────────────────────

async function createWindow(): Promise<void> {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    show: false,
    title: 'MagnetPlay',
    autoHideMenuBar: true,   // hidden by default; Alt toggles visibility
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  // CSP: allow API calls to Spring Boot (:8080) and HLS server (:8091)
  mainWindow.webContents.session.webRequest.onHeadersReceived((details, callback) => {
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        'Content-Security-Policy': [
          isDev ?
            "default-src 'self' file:; " +
            "script-src 'self' 'unsafe-inline'; " +
            "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
            "font-src 'self' https://fonts.gstatic.com data:; " +
            "img-src 'self' data: blob:;"
          :
            "default-src 'self' file:; " +
            "connect-src 'self' http://localhost:8080 http://localhost:8091 ws://localhost:8080 blob:; " +
            "media-src 'self' http://localhost:8080 http://localhost:8091 blob:; " +
            "script-src 'self'; " +
            "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
            "font-src 'self' https://fonts.gstatic.com data:; " +
            "img-src 'self' data: blob:;",
        ],
      },
    });
  });

  // Hide to tray on close instead of quitting
  mainWindow.on('close', (event) => {
    if (!isQuitting) {
      event.preventDefault();
      mainWindow?.hide();
    }
  });

  mainWindow.on('closed', () => { mainWindow = null; });

  registerIpcHandlers();

  if (isDev) {
    mainWindow.loadURL('http://localhost:5173');
    mainWindow.webContents.openDevTools();
    mainWindow.show();
  } else {
    createSplash();
    createTray();

    serviceManager = new ServiceManager(app.getPath('userData'));
    try {
      await serviceManager.start(mainWindow);
    } catch (err) {
      closeSplash();
      dialog.showErrorBox(
        'MagnetPlay — Startup Failed',
        `Services could not start.\n\n${err}\n\nLogs: ${path.join(app.getPath('userData'), 'logs')}`
      );
      isQuitting = true;
      app.quit();
      return;
    }

    mainWindow.loadFile(path.join(__dirname, '../../frontend/dist/index.html'));
    mainWindow.once('ready-to-show', () => {
      closeSplash();
      mainWindow?.show();
      initUpdater(mainWindow!);
    });
  }
}

function registerIpcHandlers(): void {
  ipcMain.handle('app:quit', () => { isQuitting = true; app.quit(); });

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

const gotSingleInstanceLock = app.requestSingleInstanceLock();

if (!gotSingleInstanceLock) {
  app.quit();
} else {
  app.on('second-instance', () => {
    if (mainWindow) {
      if (!mainWindow.isVisible()) mainWindow.show();
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.focus();
    }
  });

  app.whenReady().then(createWindow);

  app.on('before-quit', (event) => {
    if (!isQuitting && serviceManager) {
      event.preventDefault();
      isQuitting = true;
      serviceManager.stop().finally(() => app.quit());
    }
  });

  // Keep app alive in tray when all windows closed
  app.on('window-all-closed', () => {
    if (isDev && process.platform !== 'darwin') app.quit();
    // In prod: stay alive in tray
  });
}
