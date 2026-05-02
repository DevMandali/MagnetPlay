import { useEffect, useRef, useState, useCallback } from 'react';
import { createPortal } from 'react-dom';
import videojs from 'video.js';
import 'video.js/dist/video-js.css';
import { SubtitleTrack, MoovStatus, AudioTrack } from '../types';
import { detectMoovPosition, API_BASE } from '../lib/utils';
import NetflixSkipOverlay from './NetflixSkipOverlay';

const MIME_LABELS: Record<string, string> = {
  'video/mp4': 'MP4',
  'video/webm': 'WebM',
  'application/x-mpegURL': 'HLS',
};

interface Props {
  infoHash: string;
  fileId: string;
  fileName: string;
  mimeType: string;
  isMkv: boolean;
  streamUrl: string;
  durationSec: number;
  audioTracks: AudioTrack[];
  subtitleTracks: SubtitleTrack[];
  moovBadge: { cls: string; icon: string; spin: boolean; label: string } | null;
  peerStats: { seeders: number; peers: number; trackers: number } | null;
  showStatusPanel: boolean;
  onToggleStatusPanel: () => void;
  showSubPanel: boolean;
  onToggleSubPanel: () => void;
  subtitleTrackCount: number;
  onError?: (err: { code: number; message: string } | null) => void;
  onMoovStatus?: (status: MoovStatus) => void;
}

export default function VideoPlayer({ infoHash, fileId, fileName, mimeType, isMkv, streamUrl, durationSec, audioTracks: _audioTracks, subtitleTracks, moovBadge, peerStats, showStatusPanel, onToggleStatusPanel, showSubPanel, onToggleSubPanel, subtitleTrackCount, onError, onMoovStatus }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const playerRef = useRef<ReturnType<typeof videojs> | null>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const trackEls = useRef<Map<string, any>>(new Map());

  const [skipState, setSkipState] = useState<{
    left: { key: number; seconds: number } | null;
    right: { key: number; seconds: number } | null;
  }>({ left: null, right: null });

  const [playerEl, setPlayerEl] = useState<Element | null>(null);

  const clickRef = useRef<{ count: number; side: 'left' | 'right' | null; timer: ReturnType<typeof setTimeout> | null }>({
    count: 0,
    side: null,
    timer: null,
  });

  const seekDebounceRef       = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reapplySubtitlesRef  = useRef<(() => void) | null>(null);
  const isMkvRef         = useRef(isMkv);
  const streamUrlRef     = useRef(streamUrl);
  const durationSecRef   = useRef(durationSec);

  // True while we are changing src — suppresses the spurious seeking event that
  // Video.js fires when it resets currentTime to 0 on every src change.
  const isSrcChangingRef = useRef(false);

  // Stream start offset after each seek on MKV sources.
  // After player.src([...&t=X]) the browser's currentTime resets to 0 but the
  // FFmpeg pipe actually starts from X. Add this to any raw currentTime() read
  // to recover the true absolute video position.
  const seekOffsetRef = useRef<number>(0);

  useEffect(() => { isMkvRef.current = isMkv; }, [isMkv]);
  useEffect(() => { streamUrlRef.current = streamUrl; }, [streamUrl]);
  useEffect(() => { durationSecRef.current = durationSec; }, [durationSec]);

  const streamUrlForNonMkv = `${API_BASE}/v1/torrent/stream/${infoHash}?fileId=${encodeURIComponent(fileId)}`;

  // ─── Moov detection ─────────────────────────────────────────────────────────
  useEffect(() => {
    if (!onMoovStatus) return;
    if (mimeType !== 'video/mp4' || isMkv) { onMoovStatus('n/a'); return; }
    onMoovStatus('checking');
    detectMoovPosition(streamUrlForNonMkv).then(onMoovStatus);
  }, [streamUrlForNonMkv, mimeType, isMkv, onMoovStatus]);

  // ─── Subtitle sync ──────────────────────────────────────────────────────────
  useEffect(() => {
    const player = playerRef.current;
    if (!player) return;

    const apply = () => {
      subtitleTracks.forEach(t => {
        if (!trackEls.current.has(t.id)) {
          const vjsEl = player.addRemoteTextTrack({ kind: 'subtitles', label: t.label, srclang: t.srclang, src: t.blobUrl }, false);
          trackEls.current.set(t.id, vjsEl);
        }
      });
      trackEls.current.forEach((track, id) => {
        if (!subtitleTracks.find(t => t.id === id)) {
          try { player.removeRemoteTextTrack(track); } catch (_) { /* ignore */ }
          trackEls.current.delete(id);
        }
      });
      subtitleTracks.forEach(t => {
        const el = trackEls.current.get(t.id);
        if (el?.track) el.track.mode = t.active ? 'showing' : 'hidden';
      });
    };

    reapplySubtitlesRef.current = apply;
    apply();
  }, [subtitleTracks]);

  // ─── Helpers ────────────────────────────────────────────────────────────────

  // True absolute video position = raw currentTime + stream-start offset.
  const getAbsoluteTime = useCallback(() => {
    const p = playerRef.current;
    if (!p) return 0;
    return (p.currentTime() ?? 0) + (isMkvRef.current ? seekOffsetRef.current : 0);
  }, []);

  // Issue a seek on an MKV player.
  // Writes _mpPendingSeek onto the player object SYNCHRONOUSLY before calling
  // player.currentTime() so the 'seeking' event handler always reads the right
  // target. We cannot rely on tech.setCurrentTime patching because that override
  // is silently lost whenever player.src() causes videojs to recreate the tech.
  const mkvSeekTo = useCallback((absoluteTarget: number) => {
    const p = playerRef.current;
    if (!p) return;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (p as any)._mpPendingSeek = absoluteTarget;
    p.currentTime(absoluteTarget);
  }, []);

  // ─── Skip ───────────────────────────────────────────────────────────────────
  const triggerSkip = useCallback((secs: number) => {
    const p = playerRef.current;
    if (!p) return;
    const absoluteNow = getAbsoluteTime();
    const dur = isMkvRef.current && durationSecRef.current > 0
      ? durationSecRef.current
      : (isFinite(p.duration() ?? Infinity) ? (p.duration() ?? Infinity) : Infinity);
    const target = Math.max(0, Math.min(dur, absoluteNow + secs));
    if (isMkvRef.current) { mkvSeekTo(target); } else { p.currentTime(target); }
    const side: 'left' | 'right' = secs > 0 ? 'right' : 'left';
    const absSecs = Math.abs(secs);
    setSkipState(prev => {
      const existing = prev[side];
      return { ...prev, [side]: { key: existing ? existing.key + 1 : 1, seconds: existing ? existing.seconds + absSecs : absSecs } };
    });
  }, [getAbsoluteTime, mkvSeekTo]);

  // ─── Keyboard shortcuts ─────────────────────────────────────────────────────
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (document.activeElement as HTMLElement)?.tagName ?? '';
      if (['INPUT', 'SELECT', 'TEXTAREA'].includes(tag)) return;
      const p = playerRef.current;
      if (!p) return;

      // All percentage-key seeks must also go through mkvSeekTo so the
      // correct target reaches the 'seeking' handler.
      const seekPct = (pct: number) => {
        const dur = isMkvRef.current && durationSecRef.current > 0
          ? durationSecRef.current
          : (p.duration() ?? 0);
        const target = dur * pct;
        if (isMkvRef.current) { mkvSeekTo(target); } else { p.currentTime(target); }
      };

      switch (e.key) {
        case 'ArrowLeft':  e.preventDefault(); triggerSkip(-10); break;
        case 'ArrowRight': e.preventDefault(); triggerSkip(+10); break;
        case ' ':
        case 'k':          e.preventDefault(); p.paused() ? p.play() : p.pause(); break;
        case 'f':          e.preventDefault(); p.isFullscreen() ? p.exitFullscreen() : p.requestFullscreen(); break;
        case 'm':          e.preventDefault(); p.muted(!p.muted()); break;
        case 'ArrowUp':    e.preventDefault(); p.volume(Math.min(1, (p.volume() ?? 0) + 0.1)); break;
        case 'ArrowDown':  e.preventDefault(); p.volume(Math.max(0, (p.volume() ?? 0) - 0.1)); break;
        case '0': case 'Home': e.preventDefault(); seekPct(0); break;
        case '1': e.preventDefault(); seekPct(0.1); break;
        case '2': e.preventDefault(); seekPct(0.2); break;
        case '3': e.preventDefault(); seekPct(0.3); break;
        case '4': e.preventDefault(); seekPct(0.4); break;
        case '5': e.preventDefault(); seekPct(0.5); break;
        case '6': e.preventDefault(); seekPct(0.6); break;
        case '7': e.preventDefault(); seekPct(0.7); break;
        case '8': e.preventDefault(); seekPct(0.8); break;
        case '9': e.preventDefault(); seekPct(0.9); break;
        case 'End': e.preventDefault(); seekPct(1); break;
        default: break;
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [triggerSkip, mkvSeekTo]);

  // ─── Double-tap overlay ─────────────────────────────────────────────────────
  const handleOverlayClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const target = e.target as HTMLElement;
    if (target.closest('.vjs-control-bar') || target.closest('.vjs-big-play-button')) return;
    if (!playerRef.current) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const side: 'left' | 'right' = e.clientX < rect.left + rect.width / 2 ? 'left' : 'right';
    const secs = side === 'right' ? +10 : -10;
    const cr = clickRef.current;

    if (cr.side !== side) { cr.count = 0; cr.side = side; if (cr.timer) clearTimeout(cr.timer); }
    cr.count += 1;
    if (cr.timer) clearTimeout(cr.timer);
    cr.timer = setTimeout(() => { cr.count = 0; cr.side = null; }, 280);

    if (cr.count >= 2) {
      cr.count = 0;
      if (cr.timer) clearTimeout(cr.timer);
      triggerSkip(secs);
    }
  }, [triggerSkip]);

  // ─── Player init ────────────────────────────────────────────────────────────
  useEffect(() => {
    if (!containerRef.current || playerRef.current) return;

    const videoEl = document.createElement('video');
    videoEl.className = 'video-js vjs-big-play-centered';
    containerRef.current.appendChild(videoEl);

    const initialSrc = isMkv
      ? [{ src: streamUrl + '&t=0', type: 'video/mp4' }]
      : [{ src: streamUrlForNonMkv, type: mimeType }];

    const player = videojs(videoEl, {
      controls: true,
      fluid: true,
      responsive: true,
      preload: 'metadata',
      playbackRates: [0.5, 0.75, 1, 1.25, 1.5, 2],
      sources: initialSrc,
      controlBar: {
        children: [
          'playToggle',
          { name: 'SkipBackButton', seconds: 10 },
          { name: 'SkipForwardButton', seconds: 10 },
          'volumePanel',
          'currentTimeDisplay',
          'timeDivider',
          'durationDisplay',
          'progressControl',
          'remainingTimeDisplay',
          'playbackRateMenuButton',
          { name: 'StatsButton', infoHash, fileId },
          'fullscreenToggle',
        ],
      },
    });

    playerRef.current = player;
    setPlayerEl(player.el());

    player.on('error', () => {
      const err = player.error();
      onError?.(err ? { code: err.code, message: err.message } : null);
    });


    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    player.on('mp-skip', (e: any) => {
      const dir: string = e.direction;
      const sec: number = e.seconds ?? 10;
      if (dir === 'left')  setSkipState(s => ({ ...s, left:  { key: (s.left?.key  ?? 0) + 1, seconds: (s.left?.seconds  ?? 0) + sec } }));
      if (dir === 'right') setSkipState(s => ({ ...s, right: { key: (s.right?.key ?? 0) + 1, seconds: (s.right?.seconds ?? 0) + sec } }));
    });

    if (isMkv) {
      // ── SeekBar patches ─────────────────────────────────────────────────────
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const SeekBarComp = (videojs as any).getComponent('SeekBar');
      if (SeekBarComp && !SeekBarComp.prototype._mpSeekPatched) {
        const origMove    = SeekBarComp.prototype.handleMouseMove;
        const origPercent = SeekBarComp.prototype.getPercent;

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        SeekBarComp.prototype.handleMouseMove = function(this: any, event: any, mouseDown = false) {
          const dur: number = this.player_?._mpDuration;
          if (mouseDown && dur > 0) {
            const distance: number = this.calculateDistance(event);
            const target = Math.max(0, Math.min(dur, distance * dur));
            // Write _mpPendingSeek BEFORE currentTime() so the 'seeking' handler
            // reads the correct absolute target (not the browser-snapped 0).
            this.player_._mpPendingSeek = target;
            this.player_.currentTime(target);
            return;
          }
          return origMove.call(this, event, mouseDown);
        };

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        SeekBarComp.prototype.getPercent = function(this: any) {
          const dur: number = this.player_?._mpDuration;
          if (dur > 0) {
            const offset: number = this.player_?._mpSeekOffset ?? 0;
            const ct: number = (this.player_.currentTime() ?? 0) + offset;
            return Math.min(1, Math.max(0, ct / dur));
          }
          return origPercent.call(this);
        };

        SeekBarComp.prototype._mpSeekPatched = true;
      }

      // ── Time display patches ─────────────────────────────────────────────────
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const CurrentTimeDisplayComp = (videojs as any).getComponent('CurrentTimeDisplay');
      if (CurrentTimeDisplayComp && !CurrentTimeDisplayComp.prototype._mpCtdPatched) {
        const origCtd = CurrentTimeDisplayComp.prototype.updateContent;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        CurrentTimeDisplayComp.prototype.updateContent = function(this: any, event: any) {
          const offset: number = this.player_?._mpSeekOffset ?? 0;
          const dur: number    = this.player_?._mpDuration ?? 0;
          if (offset > 0 && dur > 0) {
            // video.js 8 uses updateTextNode_(time) — no duration arg, no updateFormattedTime_
            const rawTime: number = this.player_.scrubbing()
              ? (this.player_.getCache().currentTime ?? 0)
              : (this.player_.currentTime() ?? 0);
            if (typeof this.updateTextNode_ === 'function') this.updateTextNode_(rawTime + offset);
            return;
          }
          origCtd.call(this, event);
        };
        CurrentTimeDisplayComp.prototype._mpCtdPatched = true;
      }

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const RemainingTimeDisplayComp = (videojs as any).getComponent('RemainingTimeDisplay');
      if (RemainingTimeDisplayComp && !RemainingTimeDisplayComp.prototype._mpRtdPatched) {
        const origRtd = RemainingTimeDisplayComp.prototype.updateContent;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        RemainingTimeDisplayComp.prototype.updateContent = function(this: any, event: any) {
          const offset: number = this.player_?._mpSeekOffset ?? 0;
          const dur: number    = this.player_?._mpDuration ?? 0;
          if (offset > 0 && dur > 0) {
            const rawTime: number = this.player_.scrubbing()
              ? (this.player_.getCache().currentTime ?? 0)
              : (this.player_.currentTime() ?? 0);
            if (typeof this.updateTextNode_ === 'function') this.updateTextNode_(Math.max(0, dur - (rawTime + offset)));
            return;
          }
          origRtd.call(this, event);
        };
        RemainingTimeDisplayComp.prototype._mpRtdPatched = true;
      }

      // ── VOD-mode duration ────────────────────────────────────────────────────
      const applyVodMode = () => {
        const dur = durationSecRef.current;
        if (dur <= 0) return;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (player as any)._mpDuration = dur;
        player.duration(dur);
      };

      player.on('loadedmetadata', applyVodMode);
      player.on('canplay',        applyVodMode);
      player.on('durationchange', () => {
        if (isMkvRef.current && durationSecRef.current > 0) player.duration(durationSecRef.current);
      });

      // ── Seek handler ─────────────────────────────────────────────────────────
      // _mpPendingSeek is written synchronously before every player.currentTime()
      // call at each seek call-site, so it is always set when this event fires.
      player.on('seeking', () => {
        if (!isMkvRef.current || isSrcChangingRef.current) return;

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const seekTarget: number = (player as any)._mpPendingSeek ?? 0;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (player as any)._mpPendingSeek = null;

        if (seekDebounceRef.current) clearTimeout(seekDebounceRef.current);
        seekDebounceRef.current = setTimeout(() => {
          const url = streamUrlRef.current;
          if (!url) return;

          seekOffsetRef.current = seekTarget;
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          (player as any)._mpSeekOffset = seekTarget;

          let released = false;
          const releaseSrcLock = () => {
            if (!released) { released = true; isSrcChangingRef.current = false; }
          };

          isSrcChangingRef.current = true;
          player.src([{ src: `${url}&t=${seekTarget.toFixed(3)}`, type: 'video/mp4' }]);
          player.play();
          player.one('canplay', () => {
            releaseSrcLock();
            // player.src() auto-removes manualCleanup=false tracks; re-add them now
            trackEls.current.clear();
            reapplySubtitlesRef.current?.();
          });
          setTimeout(releaseSrcLock, 1500); // safety fallback
        }, 300);
      });
    }

    return () => {
      if (seekDebounceRef.current) clearTimeout(seekDebounceRef.current);
      player.dispose();
      playerRef.current = null;
      trackEls.current.clear();
      setPlayerEl(null);
    };
  }, []);  // eslint-disable-line react-hooks/exhaustive-deps

  // ─── Src update on file switch (not a user seek) ────────────────────────────
  useEffect(() => {
    const player = playerRef.current;
    if (!player) return;
    if (isMkv) {
      seekOffsetRef.current = 0;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (player as any)._mpSeekOffset = 0;

      let released = false;
      const releaseSrcLock = () => {
        if (!released) { released = true; isSrcChangingRef.current = false; }
      };

      isSrcChangingRef.current = true;
      player.src([{ src: streamUrl + '&t=0', type: 'video/mp4' }]);
      if (durationSec > 0) player.duration(durationSec);
      player.one('canplay', () => {
        releaseSrcLock();
        // player.src() auto-removes manualCleanup=false tracks; re-add them now
        trackEls.current.clear();
        reapplySubtitlesRef.current?.();
      });
      setTimeout(releaseSrcLock, 1500);
    } else {
      player.src([{ src: streamUrlForNonMkv, type: mimeType }]);
    }
  }, [streamUrlForNonMkv, mimeType, isMkv, streamUrl, durationSec]);

  return (
    <div
      className="video-player-wrapper"
      onClick={handleOverlayClick}
      onMouseMove={() => playerRef.current?.userActive(true)}
      onMouseLeave={() => playerRef.current?.userActive(false)}
      onContextMenu={(e) => e.preventDefault()}
      onDragStart={(e) => e.preventDefault()}
    >
      <div ref={containerRef} />
      {playerEl && createPortal(
        <div className="nf-top-overlay">
          <div className="nf-top-left">
            <span className="nf-live-dot" />
            <span className="nf-title">{fileName}</span>
          </div>
          <div className="nf-top-right">
            {moovBadge && (
              <span className={`badge ${moovBadge.cls}`} title={moovBadge.label}>
                <span className={moovBadge.spin ? 'badge-spin' : ''}>{moovBadge.icon}</span>
                &nbsp;{moovBadge.label}
              </span>
            )}
            <span className="badge badge-mime">{MIME_LABELS[mimeType] ?? mimeType}</span>
            {peerStats !== null && (
              <span
                className={`badge ${peerStats.seeders > 0 ? 'badge-ok' : 'badge-muted'}`}
                title={`${peerStats.peers} active peers · ${peerStats.trackers} trackers`}
              >
                ⇅ {peerStats.seeders}
              </span>
            )}
            <button
              className={`stat-toggle-btn${showStatusPanel ? ' active' : ''}`}
              onClick={onToggleStatusPanel}
              title="Download status"
            >
              <svg viewBox="0 0 24 24" width="13" height="13" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" style={{ display: 'block' }}>
                <circle cx="12" cy="12" r="9" stroke="currentColor" strokeWidth="1.8" fill="none"/>
                <ellipse cx="12" cy="12" rx="3.5" ry="9" stroke="currentColor" strokeWidth="1.5" fill="none"/>
                <line x1="3" y1="12" x2="21" y2="12" stroke="currentColor" strokeWidth="1.5"/>
                <line x1="5" y1="7.5" x2="19" y2="7.5" stroke="currentColor" strokeWidth="1.3"/>
                <line x1="5" y1="16.5" x2="19" y2="16.5" stroke="currentColor" strokeWidth="1.3"/>
              </svg>
            </button>
            <button
              className={`cc-toggle-btn${showSubPanel ? ' active' : ''}`}
              onClick={onToggleSubPanel}
              title="Subtitles / Captions"
            >
              CC
              {subtitleTrackCount > 0 && (
                <span className="cc-track-count">{subtitleTrackCount}</span>
              )}
            </button>
          </div>
        </div>,
        playerEl
      )}
      {playerEl && skipState.left && createPortal(
        <NetflixSkipOverlay side="left" seconds={skipState.left.seconds} animKey={skipState.left.key} onHidden={() => setSkipState(s => ({ ...s, left: null }))} />,
        playerEl
      )}
      {playerEl && skipState.right && createPortal(
        <NetflixSkipOverlay side="right" seconds={skipState.right.seconds} animKey={skipState.right.key} onHidden={() => setSkipState(s => ({ ...s, right: null }))} />,
        playerEl
      )}
    </div>
  );
}