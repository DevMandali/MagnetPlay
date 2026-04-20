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

export interface TorrentListItem {
  torrentId: string;
  name: string;
  state: 'TORRENT_ACTIVE' | 'TORRENT_PAUSED' | 'TORRENT_STOPPED';
  totalSize: number;
  downloadedBytes: number;
  completionPct: number;
  downloadSpeedBps: number;
  files: Array<{ id: string; name: string; size: number }>;
}

export interface StreamingSession {
  sessionId: string;
  infoHash: string;
  fileId: string;
  clientIp: string;
  startTime: string;
  endTime: string | null;
  startByte: number;
  bytesServed: number;
  status: 'ACTIVE' | 'COMPLETED' | 'CANCELLED' | 'ERROR';
  closeReason: string | null;
}
