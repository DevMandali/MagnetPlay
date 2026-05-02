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
