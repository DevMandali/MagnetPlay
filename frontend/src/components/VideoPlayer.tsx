import { useEffect, useRef, useState, useCallback } from 'react';
import { createPortal } from 'react-dom';
import videojs from 'video.js';
import 'video.js/dist/video-js.css';
import { SubtitleTrack, MoovStatus } from '../types';
import { detectMoovPosition } from '../lib/utils';
import NetflixSkipOverlay from './NetflixSkipOverlay';

interface Props {
  infoHash: string;
  fileId: string;
  mimeType: string;
  subtitleTracks: SubtitleTrack[];
  onError?: (err: { code: number; message: string } | null) => void;
  onMoovStatus?: (status: MoovStatus) => void;
}

export default function VideoPlayer({ infoHash, fileId, mimeType, subtitleTracks, onError, onMoovStatus }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const playerRef = useRef<ReturnType<typeof videojs> | null>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const trackEls = useRef<Map<string, any>>(new Map());

  const [skipState, setSkipState] = useState<{
    left: { key: number; seconds: number } | null;
    right: { key: number; seconds: number } | null;
  }>({ left: null, right: null });

  // Tracks player.el() so overlays can be portaled inside it (required for fullscreen visibility)
  const [playerEl, setPlayerEl] = useState<Element | null>(null);

  // Double-click detection state
  const clickRef = useRef<{ count: number; side: 'left' | 'right' | null; timer: ReturnType<typeof setTimeout> | null }>({
    count: 0,
    side: null,
    timer: null,
  });

  const streamUrl = `/v1/torrent/stream/${infoHash}?fileId=${encodeURIComponent(fileId)}`;

  // Detect moov atom position
  useEffect(() => {
    if (!onMoovStatus) return;
    if (mimeType !== 'video/mp4') { onMoovStatus('n/a'); return; }
    onMoovStatus('checking');
    detectMoovPosition(streamUrl).then(onMoovStatus);
  }, [streamUrl, mimeType, onMoovStatus]);

  // Sync subtitle tracks to videojs
  useEffect(() => {
    const player = playerRef.current;
    if (!player) return;

    subtitleTracks.forEach(t => {
      if (!trackEls.current.has(t.id)) {
        const vjsEl = player.addRemoteTextTrack({
          kind: 'subtitles',
          label: t.label,
          srclang: t.srclang,
          src: t.blobUrl,
        }, false);
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
  }, [subtitleTracks]);

  // Unified skip: seek + show animation (accumulates seconds on consecutive presses)
  const triggerSkip = useCallback((secs: number) => {
    const p = playerRef.current;
    if (!p) return;
    const now = p.currentTime() ?? 0;
    const dur = isFinite(p.duration() ?? Infinity) ? (p.duration() ?? Infinity) : Infinity;
    p.currentTime(Math.max(0, Math.min(dur, now + secs)));
    const side: 'left' | 'right' = secs > 0 ? 'right' : 'left';
    const absSecs = Math.abs(secs);
    setSkipState(prev => {
      const existing = prev[side];
      return { ...prev, [side]: { key: existing ? existing.key + 1 : 1, seconds: existing ? existing.seconds + absSecs : absSecs } };
    });
  }, []);

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (document.activeElement as HTMLElement)?.tagName ?? '';
      if (['INPUT', 'SELECT', 'TEXTAREA'].includes(tag)) return;
      const p = playerRef.current;
      if (!p) return;
      switch (e.key) {
        case 'ArrowLeft':  e.preventDefault(); triggerSkip(-10); break;
        case 'ArrowRight': e.preventDefault(); triggerSkip(+10); break;
        case ' ':
        case 'k':          e.preventDefault(); p.paused() ? p.play() : p.pause(); break;
        case 'f':          e.preventDefault(); p.isFullscreen() ? p.exitFullscreen() : p.requestFullscreen(); break;
        case 'm':          e.preventDefault(); p.muted(!p.muted()); break;
        case 'ArrowUp':    e.preventDefault(); p.volume(Math.min(1, (p.volume() ?? 0) + 0.1)); break;
        case 'ArrowDown':  e.preventDefault(); p.volume(Math.max(0, (p.volume() ?? 0) - 0.1)); break;
        case '0': case 'Home': e.preventDefault(); p.currentTime(0); break;
        case '1': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.1); break;
        case '2': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.2); break;
        case '3': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.3); break;
        case '4': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.4); break;
        case '5': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.5); break;
        case '6': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.6); break;
        case '7': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.7); break;
        case '8': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.8); break;
        case '9': e.preventDefault(); p.currentTime((p.duration() ?? 0) * 0.9); break;
        case 'End': e.preventDefault(); p.currentTime(p.duration() ?? 0); break;
        default: break;
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [triggerSkip]);

  // Double-click to skip: ignore clicks on control bar, require 2 clicks within 280ms
  const handleOverlayClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    const target = e.target as HTMLElement;
    if (target.closest('.vjs-control-bar') || target.closest('.vjs-big-play-button')) return;
    if (!playerRef.current) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const side: 'left' | 'right' = e.clientX < rect.left + rect.width / 2 ? 'left' : 'right';
    const secs = side === 'right' ? +10 : -10;
    const cr = clickRef.current;

    if (cr.side !== side) {
      cr.count = 0;
      cr.side = side;
      if (cr.timer) clearTimeout(cr.timer);
    }
    cr.count += 1;
    if (cr.timer) clearTimeout(cr.timer);
    cr.timer = setTimeout(() => { cr.count = 0; cr.side = null; }, 280);

    if (cr.count >= 2) {
      cr.count = 0;
      if (cr.timer) clearTimeout(cr.timer);
      triggerSkip(secs);
    }
  }, [triggerSkip]);

  // Init player
  useEffect(() => {
    if (!containerRef.current || playerRef.current) return;

    const videoEl = document.createElement('video');
    videoEl.className = 'video-js vjs-big-play-centered';
    containerRef.current.appendChild(videoEl);

    const player = videojs(videoEl, {
      controls: true,
      fluid: true,
      responsive: true,
      preload: 'metadata',
      playbackRates: [0.5, 0.75, 1, 1.25, 1.5, 2],
      sources: [{ src: streamUrl, type: mimeType }],
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

    // mp-skip: triggered by control-bar buttons; data is on event object directly
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    player.on('mp-skip', (e: any) => {
      const dir: string = e.direction;
      const sec: number = e.seconds ?? 10;
      if (dir === 'left')  setSkipState(s => ({ ...s, left:  { key: (s.left?.key  ?? 0) + 1, seconds: (s.left?.seconds  ?? 0) + sec } }));
      if (dir === 'right') setSkipState(s => ({ ...s, right: { key: (s.right?.key ?? 0) + 1, seconds: (s.right?.seconds ?? 0) + sec } }));
    });

    return () => {
      player.dispose();
      playerRef.current = null;
      trackEls.current.clear();
      setPlayerEl(null);
    };
  }, []);  // eslint-disable-line react-hooks/exhaustive-deps

  // Update src when active changes
  useEffect(() => {
    const player = playerRef.current;
    if (!player) return;
    player.src([{ src: streamUrl, type: mimeType }]);
  }, [streamUrl, mimeType]);

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
      {/* Portaled into player.el() so overlays appear inside the fullscreen element */}
      {playerEl && skipState.left && createPortal(
        <NetflixSkipOverlay
          side="left"
          seconds={skipState.left.seconds}
          animKey={skipState.left.key}
          onHidden={() => setSkipState(s => ({ ...s, left: null }))}
        />,
        playerEl
      )}
      {playerEl && skipState.right && createPortal(
        <NetflixSkipOverlay
          side="right"
          seconds={skipState.right.seconds}
          animKey={skipState.right.key}
          onHidden={() => setSkipState(s => ({ ...s, right: null }))}
        />,
        playerEl
      )}
    </div>
  );
}
