import { useState, useEffect, useRef, useCallback } from 'react';
import { TorrentFile, ActivePlayer, SubtitleTrack, MoovStatus, FetchState, TorrentFileStats, RemuxStartResponse } from './types';
import { srtToVtt, readFileAsText } from './lib/utils';
import SubtitlePanel from './components/SubtitlePanel';
import StatusPanel from './components/StatusPanel';
import VideoPlayer from './components/VideoPlayer';
import { TorrentsPage } from './components/TorrentsPage';
import SearchPanel from './components/SearchPanel';

const MOOV_BADGE: Record<string, { cls: string; icon: string; spin: boolean; label: string } | null> = {
  idle:     null,
  checking: { cls: 'badge-muted', icon: '↻', spin: true,  label: 'Probing moov'           },
  front:    { cls: 'badge-ok',    icon: '✓', spin: false, label: 'Fast-start ready'        },
  end:      { cls: 'badge-warn',  icon: '⚠', spin: false, label: 'moov at end — slow seek' },
  unknown:  { cls: 'badge-muted', icon: '?', spin: false, label: 'moov unknown'            },
  'n/a':    null,
};

export default function App() {
  // Wizard state
  const [step, setStep] = useState<1 | 2>(1);
  const [magnetLink, setMagnetLink] = useState('');
  const [fetchState, setFetchState] = useState<FetchState>('idle');
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [torrentFiles, setTorrentFiles] = useState<TorrentFile[]>([]);
  const [infoHash, setInfoHash] = useState('');

  // Player state
  const [draft, setDraft] = useState({ infoHash: '', fileId: '', mimeType: 'video/mp4', fileName: '' });
  const [active, setActive] = useState<ActivePlayer | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [moovStatus, setMoovStatus] = useState<MoovStatus>('idle');

  // Subtitle state
  const [showSubPanel, setShowSubPanel] = useState(false);
  const [showStatusPanel, setShowStatusPanel] = useState(false);
  const [showDashboard, setShowDashboard] = useState(false);
  const [configCollapsed, setConfigCollapsed] = useState(false);
  const [inputMode, setInputMode] = useState<'magnet' | 'search'>('magnet');
  const [peerStats, setPeerStats] = useState<Pick<TorrentFileStats, 'seeders' | 'peers' | 'trackers'> | null>(null);
  const [subFontSize, setSubFontSize] = useState(100);
  const [subtitleTracks, setSubtitleTracks] = useState<SubtitleTrack[]>([]);
  const [streamLoading, setStreamLoading] = useState(false);
  const blobUrls = useRef<Record<string, string>>({});

  // Live subtitle font-size injection
  useEffect(() => {
    const styleId = 'mp-sub-size';
    let el = document.getElementById(styleId) as HTMLStyleElement | null;
    if (!el) {
      el = document.createElement('style');
      el.id = styleId;
      document.head.appendChild(el);
    }
    el.textContent = `.video-js .vjs-text-track-cue { font-size: ${subFontSize / 100}em !important; }`;
  }, [subFontSize]);

  const handleSubtitleAdd = useCallback(async (file: File, labelOverride: string, langOverride: string) => {
    const ext = file.name.split('.').pop()?.toLowerCase();
    const isSrt = ext === 'srt';
    const isVtt = ext === 'vtt';
    if (!isSrt && !isVtt) return;

    const rawText = await readFileAsText(file);
    const vttText = isSrt ? srtToVtt(rawText) : rawText;

    if (!vttText.startsWith('WEBVTT')) console.warn('[subtitles] missing WEBVTT header');

    const blob = new Blob([vttText], { type: 'text/vtt' });
    const url = URL.createObjectURL(blob);
    const id = `sub_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`;
    const label = labelOverride || file.name.replace(/\.[^.]+$/, '');
    const srclang = langOverride || 'en';

    blobUrls.current[id] = url;
    setSubtitleTracks(prev => [
      ...prev.map(t => ({ ...t, active: false })),
      { id, label, srclang, format: isSrt ? 'SRT' : 'VTT', blobUrl: url, active: true },
    ]);
  }, []);

  const handleSubtitleToggle = useCallback((id: string) => {
    setSubtitleTracks(prev =>
      prev.map(t => ({ ...t, active: t.id === id ? !t.active : false }))
    );
  }, []);

  const handleSubtitleRemove = useCallback((id: string) => {
    setSubtitleTracks(prev => prev.filter(t => t.id !== id));
    if (blobUrls.current[id]) {
      URL.revokeObjectURL(blobUrls.current[id]);
      delete blobUrls.current[id];
    }
  }, []);

  const handleReset = () => {
    setActive(null);
    setError(null);
    setMoovStatus('idle');
    setStreamLoading(false);
    setShowSubPanel(false);
    setShowStatusPanel(false);
    Object.values(blobUrls.current).forEach(u => URL.revokeObjectURL(u));
    blobUrls.current = {};
    setSubtitleTracks([]);
  };

  const handleFullReset = () => {
    handleReset();
    setStep(1);
    setMagnetLink('');
    setFetchState('idle');
    setFetchError(null);
    setTorrentFiles([]);
    setInfoHash('');
    setDraft({ infoHash: '', fileId: '', mimeType: 'video/mp4', fileName: '' });
    setConfigCollapsed(false);
    setInputMode('magnet');
  };

  const handleFetchFiles = async () => {
    const mag = magnetLink.trim();
    if (!mag) return;
    setFetchState('loading');
    setFetchError(null);
    try {
      const res = await fetch('/v1/torrent/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ magnet: mag }),
        signal: AbortSignal.timeout(40000),
      });
      if (!res.ok) throw new Error(`Server returned ${res.status}`);
      const data = await res.json();
      const hash = (data.torrentId as string) || '';
      const rawFiles = (data.files && typeof data.files === 'object' ? data.files : {}) as Record<string, string>;
      const files: TorrentFile[] = Object.entries(rawFiles).map(([id, label]) => {
        const [namePart, sizePart] = label.split(' => ');
        return { id, name: namePart.trim(), sizeLabel: (sizePart ?? '').trim() };
      });
      if (!files.length) throw new Error('No files found in torrent');
      const firstVideo = files.find(f => /\.(mp4|mkv|avi|mov|webm|ts|m4v|flv)$/i.test(f.name)) ?? files[0];
      setInfoHash(hash);
      setTorrentFiles(files);
      setDraft(d => ({ ...d, infoHash: hash, fileId: firstVideo.id, fileName: firstVideo.name }));
      setStep(2);
      setFetchState('idle');
    } catch (err) {
      setFetchState('error');
      setFetchError(err instanceof Error ? err.message : 'Failed to fetch torrent info');
    }
  };

  const handleChange = (field: string) => (e: React.ChangeEvent<HTMLSelectElement>) =>
    setDraft(d => ({ ...d, [field]: e.target.value }));

  const handlePlay = async () => {
    if (!draft.infoHash.trim() || !draft.fileId.trim()) return;
    setError(null);
    setMoovStatus('idle');
    setSubtitleTracks([]);
    setConfigCollapsed(true);

    const isMkv = draft.fileName.toLowerCase().endsWith('.mkv');

    if (isMkv) {
      // streamLoading only covers the MKV remux start — non-MKV paths are synchronous setActive calls.
      setStreamLoading(true);
      try {
        const res = await fetch(
          `/v1/torrent/remux/${draft.infoHash}/start?fileId=${encodeURIComponent(draft.fileId)}&t=0`,
          { method: 'POST', signal: AbortSignal.timeout(15000) }
        );
        if (!res.ok) throw new Error(`Remux start failed: ${res.status}`);
        const data: RemuxStartResponse = await res.json();
        if (!data.success) throw new Error('Remux handler failed to start');
        setActive({
          infoHash: draft.infoHash,
          fileId: draft.fileId,
          fileName: draft.fileName,
          mimeType: 'video/mp4',
          isMkv: true,
          streamUrl: data.manifestUrl,
          durationSec: data.durationSec,
          audioTracks: data.audioTracks,
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to start remux stream');
      } finally {
        setStreamLoading(false);
      }
    } else {
      setActive({
        infoHash: draft.infoHash,
        fileId: draft.fileId,
        fileName: draft.fileName,
        mimeType: draft.mimeType,
        isMkv: false,
        streamUrl: '',
        durationSec: 0,
        audioTracks: [],
      });
    }
  };

  const handleError = useCallback((err: { code: number; message: string } | null) => {
    setError(err ? `[${err.code}] ${err.message}` : 'Unknown playback error');
  }, []);

  useEffect(() => {
    if (!active) { setPeerStats(null); return; }
    const poll = async () => {
      try {
        const res = await fetch(`/v1/torrent/stats/${active.infoHash}?fileId=${encodeURIComponent(active.fileId)}`);
        if (res.ok) {
          const data: TorrentFileStats = await res.json();
          setPeerStats({ seeders: data.seeders ?? 0, peers: data.peers ?? 0, trackers: data.trackers ?? 0 });
        }
      } catch { /* ignore */ }
    };
    poll();
    const id = setInterval(poll, 5000);
    return () => clearInterval(id);
  }, [active]);

  const moovBadge = MOOV_BADGE[moovStatus] ?? null;
  const streamUrlPreview = active
    ? `/v1/torrent/stream/${active.infoHash}?fileId=${encodeURIComponent(active.fileId)}`
    : '';
  const activeTrackCount = subtitleTracks.filter(t => t.active).length;

  return (
    <>
      <header className="header">
        <span className="header-logo">▶ MagnetPlay</span>
        <span className="header-title">MagnetPlay Video Player</span>
        <button
          className="btn btn-ghost"
          onClick={() => setShowDashboard(true)}
          style={{ marginLeft: 'auto', fontSize: 13 }}
          title="Open torrent dashboard"
        >
          ☰ Dashboard
        </button>
      </header>

      {showDashboard && <TorrentsPage onClose={() => setShowDashboard(false)} />}

      {/* ── Wizard card ── */}
      <div className="wizard-viewport card" style={{ position: 'relative' }}>
        {fetchState === 'loading' && (
          <div className="fetch-overlay">
            <div className="fetch-spinner" />
            <span className="fetch-label">Fetching torrent info…</span>
            <span className="fetch-magnet-preview">{magnetLink}</span>
          </div>
        )}

        <div
          className="card-header"
          style={step === 2 ? { cursor: 'pointer' } : undefined}
          onClick={step === 2 ? () => setConfigCollapsed(v => !v) : undefined}
        >
          <span className="dot dot-red" />
          <span className="dot dot-amber" />
          <span className="dot dot-green" />
          <span className="card-label">Stream Config</span>
          {step === 2 && (
            <span style={{ marginLeft: 'auto', fontSize: 10, color: 'var(--muted)', userSelect: 'none' }}>
              {configCollapsed ? '▼ expand' : '▲ collapse'}
            </span>
          )}
        </div>

        {!configCollapsed && (
          <>
          <div className="wizard-steps">
            <div className={`wizard-step-pill ${step === 1 ? 'active' : 'done'}`}>
              <div className="wizard-step-num">{step > 1 ? '✓' : '1'}</div>
              Magnet Link
            </div>
            <div className={`wizard-sep ${step > 1 ? 'done' : ''}`} />
            <div className={`wizard-step-pill ${step === 2 ? 'active' : ''}`}>
              <div className="wizard-step-num">2</div>
              Select File
            </div>
          </div>

        <div className={`wizard-track step-${step}`}>
          {/* Step 1 */}
          <div className="wizard-panel">
            {/* Tab bar */}
            <div style={{ display: 'flex', marginBottom: 12, borderBottom: '1px solid var(--border)' }}>
              {(['magnet', 'search'] as const).map(mode => (
                <button
                  key={mode}
                  onClick={() => setInputMode(mode)}
                  style={{
                    padding: '6px 16px',
                    background: 'none',
                    border: 'none',
                    borderBottom: inputMode === mode ? '2px solid var(--accent, #818cf8)' : '2px solid transparent',
                    color: inputMode === mode ? 'var(--text)' : 'var(--muted)',
                    cursor: 'pointer',
                    fontSize: 12,
                    fontFamily: 'var(--mono)',
                    marginBottom: -1,
                  }}
                >
                  {mode === 'magnet' ? '⎘ Paste Magnet' : '⌕ Search'}
                </button>
              ))}
            </div>

            {inputMode === 'search' ? (
              <div style={{ padding: '0 24px 24px' }}>
                <SearchPanel
                  onSelect={(mag) => {
                    setMagnetLink(mag);
                    setFetchState('idle');
                    setFetchError(null);
                    setInputMode('magnet');
                  }}
                />
              </div>
            ) : (
              <div className="form-grid" style={{ gridTemplateColumns: '1fr' }}>
                <div className="form-group full">
                  <label htmlFor="magnetLink">Magnet Link</label>
                  <input
                    id="magnetLink"
                    type="text"
                    placeholder="magnet:?xt=urn:btih:…"
                    value={magnetLink}
                    onChange={e => { setMagnetLink(e.target.value); setFetchState('idle'); setFetchError(null); }}
                    onKeyDown={e => e.key === 'Enter' && handleFetchFiles()}
                    style={{ fontFamily: 'var(--mono)', fontSize: 11 }}
                  />
                  {fetchState === 'error' && (
                    <div style={{
                      marginTop: 6, padding: '8px 12px',
                      background: 'rgba(230,57,70,.1)', border: '1px solid rgba(230,57,70,.3)',
                      borderRadius: 6, fontFamily: 'var(--mono)', fontSize: 10,
                      color: '#ff8080', display: 'flex', alignItems: 'center', gap: 8,
                    }}>
                      <span>⚠</span>{fetchError}
                    </div>
                  )}
                </div>
              </div>
            )}

            {inputMode === 'magnet' && (
              <div className="form-actions">
                <button
                  className="btn btn-primary"
                  onClick={handleFetchFiles}
                  disabled={!magnetLink.trim() || fetchState === 'loading'}
                >
                  ▶ Fetch Files
                </button>
                <button
                  className="btn btn-ghost"
                  onClick={async () => {
                    try {
                      const text = await navigator.clipboard.readText();
                      if (text) { setMagnetLink(text); setFetchState('idle'); setFetchError(null); }
                    } catch { /* clipboard permission denied */ }
                  }}
                  title="Paste from clipboard"
                >
                  ⎘ Paste
                </button>
                {magnetLink && (
                  <button
                    className="btn btn-ghost"
                    onClick={() => { setMagnetLink(''); setFetchState('idle'); setFetchError(null); }}
                    title="Clear"
                  >
                    ✕ Clear
                  </button>
                )}
              </div>
            )}
          </div>

          {/* Step 2 */}
          <div className="wizard-panel">
            <div className="form-grid">
              <div className="form-group full">
                <label htmlFor="fileSelect">
                  Video File
                  {torrentFiles.length > 0 && (
                    <span className="file-count-badge">{torrentFiles.length} files</span>
                  )}
                </label>
                <select
                  id="fileSelect"
                  className="file-select"
                  value={draft.fileId}
                  onChange={(e) => {
                    const selected = torrentFiles.find(f => f.id === e.target.value);
                    setDraft(d => ({ ...d, fileId: e.target.value, fileName: selected?.name ?? d.fileName }));
                  }}
                >
                  {torrentFiles.map(f => (
                    <option key={f.id} value={f.id}>
                      {f.name}{f.sizeLabel ? `  —  ${f.sizeLabel}` : ''}
                    </option>
                  ))}
                </select>
              </div>
              <div className="form-group">
                <label htmlFor="mimeType">MIME Type</label>
                <select id="mimeType" value={draft.mimeType} onChange={handleChange('mimeType')}>
                  <option value="video/mp4">video/mp4 — MP4</option>
                  <option value="video/webm">video/webm — WebM</option>
                  <option value="application/x-mpegURL">application/x-mpegURL — HLS</option>
                </select>
              </div>
              <div className="form-group" style={{ justifyContent: 'flex-end', paddingTop: 22 }}>
                <div style={{ fontFamily: 'var(--mono)', fontSize: 10, color: 'var(--muted)' }}>
                  Hash&nbsp;
                  <span style={{ color: 'var(--text)', wordBreak: 'break-all' }}>
                    {infoHash.slice(0, 16)}…
                  </span>
                </div>
              </div>
            </div>
            <div className="form-actions">
              <button className="btn btn-primary" onClick={handlePlay} disabled={streamLoading}>
                {streamLoading
                  ? <><span className="badge-spin" style={{ display: 'inline-block' }}>↻</span>&nbsp;Starting…</>
                  : '▶ Load Stream'}
              </button>
              <button className="btn-back" onClick={handleFullReset}>← Back</button>
              {active && <button className="btn btn-ghost" onClick={handleReset}>✕ Clear</button>}
            </div>
            {error && <div className="error-box"><span>⚠</span>{error}<button onClick={() => setError(null)} style={{ marginLeft: 'auto', background: 'none', border: 'none', color: '#ff8080', cursor: 'pointer', fontSize: 14, lineHeight: 1 }}>✕</button></div>}
          </div>
        </div>
          </>
        )}
      </div>

      {/* ── Player ── */}
      {streamLoading && !active ? (
        <div className="stream-loading-card">
          <div className="fetch-spinner" />
          <span className="fetch-label">Starting MKV stream…</span>
          <span className="fetch-magnet-preview">{draft.fileName}</span>
        </div>
      ) : active ? (
        <div className="player-card">
          {showStatusPanel && active && (
            <StatusPanel infoHash={active.infoHash} fileId={active.fileId} />
          )}

          {showSubPanel && (
            <SubtitlePanel
              tracks={subtitleTracks}
              onAdd={handleSubtitleAdd}
              onToggle={handleSubtitleToggle}
              onRemove={handleSubtitleRemove}
              fontSize={subFontSize}
              onFontSizeChange={setSubFontSize}
            />
          )}

          <div className="kbd-hints">
            {([
              ['Double-tap ←', 'Rewind 10s'],
              ['Double-tap →', 'Forward 10s'],
              ['Space / K', 'Play/Pause'],
              ['F', 'Fullscreen'],
              ['M', 'Mute'],
            ] as [string, string][]).map(([k, v]) => (
              <span key={k} className="kbd-hint">
                <span className="kbd">{k}</span>
                <span className="kbd-label">{v}</span>
              </span>
            ))}
          </div>

          <VideoPlayer
            key={`${active.infoHash}::${active.fileId}`}
            infoHash={active.infoHash}
            fileId={active.fileId}
            fileName={active.fileName}
            mimeType={active.mimeType}
            isMkv={active.isMkv}
            streamUrl={active.streamUrl}
            durationSec={active.durationSec}
            audioTracks={active.audioTracks}
            subtitleTracks={subtitleTracks}
            moovBadge={moovBadge}
            peerStats={peerStats}
            showStatusPanel={showStatusPanel}
            onToggleStatusPanel={() => setShowStatusPanel(v => !v)}
            showSubPanel={showSubPanel}
            onToggleSubPanel={() => setShowSubPanel(v => !v)}
            subtitleTrackCount={subtitleTracks.length}
            onError={handleError}
            onMoovStatus={setMoovStatus}
          />

          <div className="meta-row">
            <div className="meta-item">
              <span className="meta-key">Info Hash</span>
              <span className="meta-val">{active.infoHash}</span>
            </div>
            <div className="meta-item">
              <span className="meta-key">File</span>
              <span className="meta-val">{active.fileName || active.fileId}</span>
            </div>
            <div className="meta-item" style={{ maxWidth: 340 }}>
              <span className="meta-key">Stream URL</span>
              <span className="meta-val">{streamUrlPreview}</span>
            </div>
            {subtitleTracks.length > 0 && (
              <div className="meta-item">
                <span className="meta-key">Subtitle Tracks</span>
                <span className="meta-val">{subtitleTracks.length} loaded · {activeTrackCount} active</span>
              </div>
            )}
          </div>
        </div>
      ) : step === 2 ? (
        <div className="empty-state">
          <div className="empty-icon">◈</div>
          <div className="empty-label">Select a file above and press Load Stream</div>
        </div>
      ) : (
        <div className="empty-state">
          <div className="empty-icon">◈</div>
          <div className="empty-label">Enter a magnet link to begin</div>
        </div>
      )}
    </>
  );
}
