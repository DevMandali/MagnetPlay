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
