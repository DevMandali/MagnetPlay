import React, { useState, useEffect } from 'react';
import { TorrentListItem, StreamingSession } from '../types';

interface TorrentsPageProps {
  apiBase?: string;
  onClose: () => void;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function formatSpeed(bps: number) {
  return bps === 0 ? '—' : `${formatBytes(bps)}/s`;
}

function stateBadge(state: TorrentListItem['state']) {
  const colors: Record<string, string> = {
    TORRENT_ACTIVE: '#4ade80', TORRENT_PAUSED: '#facc15', TORRENT_STOPPED: '#f87171',
  };
  const labels: Record<string, string> = {
    TORRENT_ACTIVE: 'Active', TORRENT_PAUSED: 'Paused', TORRENT_STOPPED: 'Stopped',
  };
  return (
    <span style={{
      display: 'inline-block', padding: '2px 8px', borderRadius: 12,
      background: colors[state] || '#888', color: '#000', fontSize: 11, fontWeight: 700,
    }}>
      {labels[state] || state}
    </span>
  );
}

export function TorrentsPage({ apiBase = 'http://localhost:8080', onClose }: TorrentsPageProps) {
  const [torrents, setTorrents] = useState<TorrentListItem[]>([]);
  const [sessions, setSessions] = useState<StreamingSession[]>([]);
  const [tab, setTab] = useState<'torrents' | 'sessions'>('torrents');
  const [loading, setLoading] = useState(true);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [actioning, setActioning] = useState<Set<string>>(new Set());

  const fetchData = async () => {
    try {
      const [t, s] = await Promise.all([
        fetch(`${apiBase}/v1/torrent/list`).then(r => r.json()),
        fetch(`${apiBase}/v1/torrent/sessions`).then(r => r.json()),
      ]);
      setTorrents(t.torrents ?? []);
      setSessions(s.sessions ?? []);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  };

  useEffect(() => {
    fetchData();
    const id = setInterval(fetchData, 3000);
    return () => clearInterval(id);
  }, []);

  const withAction = async (id: string, fn: () => Promise<Response>) => {
    if (actioning.has(id)) return;
    setActioning(prev => new Set(prev).add(id));
    try {
      const res = await fn();
      if (res.ok) await fetchData();
    } catch { /* ignore network errors */ }
    finally { setActioning(prev => { const n = new Set(prev); n.delete(id); return n; }); }
  };

  const pause = (id: string) =>
    withAction(id, () => fetch(`${apiBase}/v1/torrent/pause/${id}`, { method: 'POST' }));

  const resume = (id: string) =>
    withAction(id, () => fetch(`${apiBase}/v1/torrent/resume/${id}`, { method: 'POST' }));

  const deleteTorrent = async (id: string, deleteFiles: boolean) => {
    setDeleteConfirm(null);
    await withAction(id, () => fetch(`${apiBase}/v1/torrent/${id}?deleteFiles=${deleteFiles}`, { method: 'DELETE' }));
  };

  const panelStyle: React.CSSProperties = {
    position: 'fixed', inset: 0, background: 'rgba(10,10,10,0.97)',
    color: '#fff', zIndex: 10000, display: 'flex', flexDirection: 'column',
    fontFamily: 'system-ui, sans-serif',
  };
  const th: React.CSSProperties = {
    padding: '10px 12px', textAlign: 'left', fontSize: 12,
    color: '#888', fontWeight: 600, borderBottom: '1px solid #333',
  };
  const td: React.CSSProperties = {
    padding: '10px 12px', fontSize: 13, borderBottom: '1px solid #222',
  };

  return (
    <div style={panelStyle}>
      <div style={{ display: 'flex', alignItems: 'center', padding: '16px 24px', borderBottom: '1px solid #333' }}>
        <h2 style={{ margin: 0, fontSize: 18 }}>Torrent Dashboard</h2>
        <div style={{ marginLeft: 24, display: 'flex', gap: 8 }}>
          {(['torrents', 'sessions'] as const).map(t => (
            <button key={t} onClick={() => setTab(t)} style={{
              background: tab === t ? '#4ade80' : '#222', color: tab === t ? '#000' : '#fff',
              border: 'none', borderRadius: 6, padding: '6px 16px',
              cursor: 'pointer', fontSize: 13, fontWeight: 600,
            }}>
              {t === 'torrents' ? 'Torrents' : 'Sessions'}
            </button>
          ))}
        </div>
        <button onClick={onClose} style={{
          marginLeft: 'auto', background: 'transparent', border: 'none',
          color: '#aaa', fontSize: 22, cursor: 'pointer',
        }}>&#x2715;</button>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
        {loading ? (
          <div style={{ color: '#aaa' }}>Loading&hellip;</div>
        ) : tab === 'torrents' ? (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                {['Name', 'Status', 'Progress', 'Downloaded', 'Total', 'Speed', 'Actions'].map(h => (
                  <th key={h} style={th}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {torrents.length === 0 ? (
                <tr><td colSpan={7} style={{ ...td, color: '#555', textAlign: 'center' }}>No torrents</td></tr>
              ) : torrents.map(t => (
                <tr key={t.torrentId}>
                  <td style={td}><span title={t.torrentId}>{t.name}</span></td>
                  <td style={td}>{stateBadge(t.state)}</td>
                  <td style={{ ...td, minWidth: 120 }}>
                    <div style={{ background: '#333', borderRadius: 4, height: 6 }}>
                      <div style={{
                        width: `${Math.min(t.completionPct, 100)}%`, height: '100%',
                        background: '#4ade80', borderRadius: 4,
                      }} />
                    </div>
                    <span style={{ fontSize: 11, color: '#aaa' }}>{t.completionPct.toFixed(1)}%</span>
                  </td>
                  <td style={td}>{formatBytes(t.downloadedBytes)}</td>
                  <td style={td}>{formatBytes(t.totalSize)}</td>
                  <td style={td}>{formatSpeed(t.downloadSpeedBps)}</td>
                  <td style={td}>
                    <div style={{ display: 'flex', gap: 6 }}>
                      {t.state === 'TORRENT_ACTIVE' ? (
                        <button onClick={() => pause(t.torrentId)} disabled={actioning.has(t.torrentId)} style={{
                          background: actioning.has(t.torrentId) ? '#555' : '#facc15',
                          color: '#000', border: 'none',
                          borderRadius: 4, padding: '3px 10px',
                          cursor: actioning.has(t.torrentId) ? 'not-allowed' : 'pointer', fontSize: 12,
                        }}>{actioning.has(t.torrentId) ? '…' : 'Pause'}</button>
                      ) : (
                        <button onClick={() => resume(t.torrentId)} disabled={actioning.has(t.torrentId)} style={{
                          background: actioning.has(t.torrentId) ? '#555' : '#4ade80',
                          color: '#000', border: 'none',
                          borderRadius: 4, padding: '3px 10px',
                          cursor: actioning.has(t.torrentId) ? 'not-allowed' : 'pointer', fontSize: 12,
                        }}>{actioning.has(t.torrentId) ? '…' : 'Resume'}</button>
                      )}
                      <button onClick={() => setDeleteConfirm(t.torrentId)} disabled={actioning.has(t.torrentId)} style={{
                        background: actioning.has(t.torrentId) ? '#555' : '#f87171',
                        color: '#fff', border: 'none',
                        borderRadius: 4, padding: '3px 10px',
                        cursor: actioning.has(t.torrentId) ? 'not-allowed' : 'pointer', fontSize: 12,
                      }}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <>
            <h3 style={{ color: '#4ade80', margin: '0 0 12px' }}>Active Sessions ({sessions.filter(s => s.status === 'ACTIVE').length})</h3>
            <SessionTable sessions={sessions.filter(s => s.status === 'ACTIVE')} td={td} th={th} />
            <h3 style={{ color: '#aaa', margin: '24px 0 12px' }}>Closed Sessions</h3>
            <SessionTable sessions={sessions.filter(s => s.status !== 'ACTIVE')} td={td} th={th} />
          </>
        )}
      </div>

      {deleteConfirm && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10001,
        }}>
          <div style={{ background: '#1a1a1a', borderRadius: 12, padding: 32, maxWidth: 360 }}>
            <h3 style={{ margin: '0 0 8px' }}>Delete Torrent?</h3>
            <p style={{ color: '#aaa', fontSize: 14 }}>Also delete downloaded files from disk?</p>
            <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
              <button onClick={() => deleteTorrent(deleteConfirm, false)} style={{
                background: '#f87171', color: '#fff', border: 'none',
                borderRadius: 6, padding: '8px 16px', cursor: 'pointer',
              }}>Delete (keep files)</button>
              <button onClick={() => deleteTorrent(deleteConfirm, true)} style={{
                background: '#dc2626', color: '#fff', border: 'none',
                borderRadius: 6, padding: '8px 16px', cursor: 'pointer',
              }}>Delete + files</button>
              <button onClick={() => setDeleteConfirm(null)} style={{
                background: '#333', color: '#fff', border: 'none',
                borderRadius: 6, padding: '8px 16px', cursor: 'pointer',
              }}>Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function SessionTable({ sessions, td, th }: {
  sessions: StreamingSession[];
  td: React.CSSProperties;
  th: React.CSSProperties;
}) {
  if (sessions.length === 0) {
    return <div style={{ color: '#555', fontSize: 13, marginBottom: 8 }}>None</div>;
  }
  return (
    <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: 16 }}>
      <thead>
        <tr>
          {['Session ID', 'File', 'Client IP', 'Start Byte', 'Bytes Served', 'Start', 'End', 'Status'].map(h => (
            <th key={h} style={th}>{h}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {sessions.map(s => (
          <tr key={s.sessionId}>
            <td style={td}><span style={{ fontFamily: 'monospace', fontSize: 11 }}>{s.sessionId.slice(0, 8)}&hellip;</span></td>
            <td style={td}>{s.fileId}</td>
            <td style={td}>{s.clientIp}</td>
            <td style={td}>{s.startByte.toLocaleString()}</td>
            <td style={td}>{s.bytesServed.toLocaleString()} B</td>
            <td style={td}>{new Date(s.startTime).toLocaleTimeString()}</td>
            <td style={td}>{s.endTime ? new Date(s.endTime).toLocaleTimeString() : '—'}</td>
            <td style={td}><span style={{ color: s.status === 'ACTIVE' ? '#4ade80' : '#888', fontSize: 12 }}>{s.status}</span></td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
