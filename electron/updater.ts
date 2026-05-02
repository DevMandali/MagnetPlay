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
