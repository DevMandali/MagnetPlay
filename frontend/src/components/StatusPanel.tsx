import { useState, useEffect, useRef } from 'react';
import { TorrentFileStats } from '../types';

interface StatusPanelProps {
  infoHash: string;
  fileId: string;
}

function fmtBytes(b: number): string {
  if (b <= 0) return '0 B';
  if (b < 1024) return `${b} B`;
  if (b < 1024 ** 2) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1024 ** 3) return `${(b / 1024 ** 2).toFixed(1)} MB`;
  return `${(b / 1024 ** 3).toFixed(2)} GB`;
}

function fmtSpeed(bps: number): string {
  if (bps <= 0) return '—';
  return `${fmtBytes(bps)}/s`;
}

export default function StatusPanel({ infoHash, fileId }: StatusPanelProps) {
  const [stats, setStats] = useState<TorrentFileStats | null>(null);
  const [loading, setLoading] = useState(true);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    let cancelled = false;

    const poll = async () => {
      try {
        const res = await fetch(
          `/v1/torrent/stats/${infoHash}?fileId=${encodeURIComponent(fileId)}`
        );
        if (res.ok && !cancelled) {
          setStats(await res.json());
          setLoading(false);
        } else if (!cancelled) {
          // HTTP error — still mark as not loading so UI doesn't freeze
          setLoading(false);
        }
      } catch {
        // network error — retain last known stats
        setLoading(false);
      }
    };

    poll();
    intervalRef.current = setInterval(poll, 2000);

    return () => {
      cancelled = true;
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [infoHash, fileId]);

  const pct = stats ? Math.min(Math.max(stats.completionPct, 0), 100) : 0;
  const complete = pct >= 99.9;

  return (
    <div className="status-panel">
      <div className="status-panel-inner">
        <div className="status-panel-header">
          <span className="status-panel-title">Download Status</span>
          <span className="status-panel-title" style={{ opacity: 0.4 }}>
            {loading ? 'loading…' : 'live · 2s'}
          </span>
        </div>

        {loading || !stats ? (
          <span className="status-stat-key">Fetching stats…</span>
        ) : (
          <>
            <div className="status-stat-grid">
              <span className="status-stat-key">Total</span>
              <span className="status-stat-val">{fmtBytes(stats.totalSize)}</span>

              <span className="status-stat-key">Downloaded</span>
              <span className={`status-stat-val${complete ? ' ok' : ''}`}>
                {fmtBytes(stats.downloadedBytes)}
              </span>

              <span className="status-stat-key">Progress</span>
              <span className={`status-stat-val${complete ? ' ok' : ' accent'}`}>
                {pct.toFixed(1)}%
              </span>

              <span className="status-stat-key">Speed</span>
              <span className="status-stat-val">
                {complete ? '— seeding' : fmtSpeed(stats.downloadSpeedBps)}
              </span>

              <span className="status-stat-key">Seeders</span>
              <span className={`status-stat-val${(stats.seeders ?? 0) > 0 ? ' ok' : ''}`}>
                {stats.seeders ?? 0}
              </span>

              <span className="status-stat-key">Peers</span>
              <span className="status-stat-val">{stats.peers ?? 0}</span>

              <span className="status-stat-key">Trackers</span>
              <span className="status-stat-val">{stats.trackers ?? 0}</span>
            </div>

            <div className="status-progress-bar">
              <div
                className={`status-progress-fill${complete ? ' complete' : ''}`}
                style={{ width: `${pct}%` }}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
}
