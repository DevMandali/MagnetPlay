export interface TorrentFile {
  id: string;
  name: string;
  sizeLabel: string;
}

export interface ActivePlayer {
  infoHash: string;
  fileId: string;
  mimeType: string;
}

export interface SubtitleTrack {
  id: string;
  label: string;
  srclang: string;
  format: 'SRT' | 'VTT';
  blobUrl: string;
  active: boolean;
}

export type MoovStatus = 'idle' | 'checking' | 'front' | 'end' | 'unknown' | 'n/a';
export type FetchState = 'idle' | 'loading' | 'error';

export interface TorrentFileStats {
  fileId: string;
  totalSize: number;
  downloadedBytes: number;
  completionPct: number;
  downloadSpeedBps: number;
}
