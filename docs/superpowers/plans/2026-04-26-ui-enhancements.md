# UI Enhancements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add MKV loading feedback, show file name instead of fileId, add Netflix-style video overlay with title and controls, fix seek progress time on hover, and restyle the player bar as a full video overlay.

**Architecture:** All overlay content (title, badges, CC/stats buttons) moves inside `VideoPlayer` via React portal into `playerEl` — the same pattern already used for skip overlays. video.js's `vjs-user-active`/`vjs-user-inactive` CSS classes drive overlay visibility with no extra state. The separate `.player-bar` above the video is removed entirely.

**Tech Stack:** React 18, TypeScript, video.js, CSS custom properties

---

## File Map

| File | Change |
|------|--------|
| `frontend/src/types/index.ts` | Add `fileName` to `ActivePlayer` |
| `frontend/src/App.tsx` | `streamLoading` state, `handlePlay` update, file select fix, remove player-bar, pass new props |
| `frontend/src/components/VideoPlayer.tsx` | New props, top overlay portal, seek hover fix |
| `frontend/src/App.css` | `.nf-top-overlay`, `.stream-loading-card`, control bar gradient |

---

### Task 1: Add `fileName` to `ActivePlayer` type

**Files:**
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Add `fileName` field to `ActivePlayer`**

In `frontend/src/types/index.ts`, replace the `ActivePlayer` interface (lines 28–36):

```ts
export interface ActivePlayer {
  infoHash: string;
  fileId: string;
  fileName: string;
  mimeType: string;
  isMkv: boolean;
  streamUrl: string; // remux base URL for MKV; empty for non-MKV
  durationSec: number;
  audioTracks: AudioTrack[];
}
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit
```

Expected: type errors on `setActive` call sites in `App.tsx` (missing `fileName`) — these are fixed in Task 3.

---

### Task 2: Fix file select to sync `fileName` in draft

**Files:**
- Modify: `frontend/src/App.tsx`

**Context:** `handleChange('fileId')` only updates `fileId` in `draft`. When user switches files, `draft.fileName` goes stale. Fix the select's `onChange` to also update `fileName`.

- [ ] **Step 1: Replace file select `onChange` in App.tsx**

Find the `<select id="fileSelect" ...>` block (around line 395). Replace:

```tsx
onChange={handleChange('fileId')}
```

with:

```tsx
onChange={(e) => {
  const selected = torrentFiles.find(f => f.id === e.target.value);
  setDraft(d => ({ ...d, fileId: e.target.value, fileName: selected?.name ?? d.fileName }));
}}
```

---

### Task 3: Add `streamLoading` state and update `handlePlay`

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Add `streamLoading` state**

After the existing state declarations (around line 43), add:

```tsx
const [streamLoading, setStreamLoading] = useState(false);
```

- [ ] **Step 2: Replace `handlePlay` with updated version**

Replace the entire `handlePlay` function (lines 161–202):

```tsx
const handlePlay = async () => {
  if (!draft.infoHash.trim() || !draft.fileId.trim()) return;
  setError(null);
  setMoovStatus('idle');
  setSubtitleTracks([]);
  setConfigCollapsed(true);

  const isMkv = draft.fileName.toLowerCase().endsWith('.mkv');

  if (isMkv) {
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
```

- [ ] **Step 3: Disable Load Stream button during loading**

Find the Load Stream button (around line 426). Replace:

```tsx
<button className="btn btn-primary" onClick={handlePlay}>▶ Load Stream</button>
```

with:

```tsx
<button className="btn btn-primary" onClick={handlePlay} disabled={streamLoading}>
  {streamLoading
    ? <><span className="badge-spin" style={{ display: 'inline-block' }}>↻</span>&nbsp;Starting…</>
    : '▶ Load Stream'}
</button>
```

- [ ] **Step 4: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors (both `setActive` calls now include `fileName`).

---

### Task 4: Add MKV loading UI in App render

**Files:**
- Modify: `frontend/src/App.tsx`

**Context:** When `streamLoading && !active`, show a loading card instead of the empty state. The existing `.fetch-spinner`, `.fetch-label`, `.fetch-magnet-preview` classes already provide the right visual style.

- [ ] **Step 1: Update the player-area conditional render**

Find the block starting with `{/* ── Player ── */}` (around line 437). The current structure is:

```tsx
{active ? (
  <div className="player-card">
    ...
  </div>
) : step === 2 ? (
  <div className="empty-state">...</div>
) : (
  <div className="empty-state">...</div>
)}
```

Wrap the outermost condition with the loading check:

```tsx
{streamLoading && !active ? (
  <div className="stream-loading-card">
    <div className="fetch-spinner" />
    <span className="fetch-label">Starting MKV stream…</span>
    <span className="fetch-magnet-preview">{draft.fileName}</span>
  </div>
) : active ? (
  <div className="player-card">
    {/* ... keep all existing player-card contents unchanged for now ... */}
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
```

---

### Task 5: Fix seek progress time on hover (MKV)

**Files:**
- Modify: `frontend/src/components/VideoPlayer.tsx`

**Context:** `SeekBar.prototype.handleMouseMove` patch runs for ALL calls including hover (`mouseDown=false`). On hover, it calls `player.currentTime(target)` which resets the currentTime display to 0 (browser currentTime=0 after MKV seek, offset not re-applied yet). Fix: only apply seek logic when `mouseDown===true`.

- [ ] **Step 1: Add `mouseDown` guard to SeekBar patch**

In `VideoPlayer.tsx`, find `SeekBarComp.prototype.handleMouseMove` (around line 263). Replace the entire assignment:

```ts
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
```

---

### Task 6: Add new props and top overlay portal to VideoPlayer

**Files:**
- Modify: `frontend/src/components/VideoPlayer.tsx`

- [ ] **Step 1: Replace Props interface**

Replace the existing `interface Props` (lines 9–20):

```ts
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
```

- [ ] **Step 2: Update function signature destructuring**

Replace line 22:

```ts
export default function VideoPlayer({ infoHash, fileId, mimeType, isMkv, streamUrl, durationSec, audioTracks, subtitleTracks, onError, onMoovStatus }: Props) {
```

with:

```ts
export default function VideoPlayer({ infoHash, fileId, fileName, mimeType, isMkv, streamUrl, durationSec, audioTracks, subtitleTracks, moovBadge, peerStats, showStatusPanel, onToggleStatusPanel, showSubPanel, onToggleSubPanel, subtitleTrackCount, onError, onMoovStatus }: Props) {
```

- [ ] **Step 3: Add `MIME_LABELS` constant at module level in VideoPlayer.tsx**

After the import lines (before the `interface Props` declaration), add:

```ts
const MIME_LABELS: Record<string, string> = {
  'video/mp4': 'MP4',
  'video/webm': 'WebM',
  'application/x-mpegURL': 'HLS',
};
```

- [ ] **Step 4: Replace the return JSX block**

Replace the entire `return (...)` block (starting around line 409):

```tsx
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
```


- [ ] **Step 5: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit
```

Expected: errors on `<VideoPlayer>` usage in App.tsx (missing new props) — fixed in Task 7.

---

### Task 7: Remove player-bar from App, wire new VideoPlayer props

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Remove the `.player-bar` block**

Inside the `active ? (...)` branch of the player conditional, find and delete the entire `.player-bar` div (lines 440–483 approximately):

```tsx
{/* DELETE THIS ENTIRE BLOCK */}
<div className="player-bar">
  <span className="live-dot" />
  <span className="player-bar-title">{active.fileId || active.infoHash}</span>
  <div className="player-bar-badges">
    {moovBadge && (...)}
    <span className="badge badge-mime">...</span>
    {peerStats !== null && (...)}
    <button className="stat-toggle-btn..." ...>...</button>
    <button className="cc-toggle-btn..." ...>CC...</button>
  </div>
</div>
```

- [ ] **Step 2: Pass new props to VideoPlayer**

Replace the existing `<VideoPlayer ... />` usage (around line 515) with:

```tsx
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
```

- [ ] **Step 3: Verify TypeScript compiles with no errors**

```bash
cd frontend && npx tsc --noEmit
```

Expected: clean — no errors.

---

### Task 8: CSS — overlay, loading card, control bar

**Files:**
- Modify: `frontend/src/App.css`

- [ ] **Step 1: Add `.stream-loading-card` rule**

After the `.fetch-magnet-preview` rule (around line 274), add:

```css
/* ── Stream loading card ────────────────────── */
.stream-loading-card { width: 100%; max-width: 860px; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 18px; padding: 60px 20px; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); }
```

- [ ] **Step 2: Add `.nf-top-overlay` rules**

After the `.video-player-wrapper` rule (around line 177), add:

```css
/* ── Netflix top overlay ────────────────────── */
.nf-top-overlay { position: absolute; top: 0; left: 0; right: 0; z-index: 5; padding: 16px 20px 48px; background: linear-gradient(to bottom, rgba(0,0,0,0.78), transparent); display: flex; align-items: flex-start; justify-content: space-between; pointer-events: none; opacity: 1; transition: opacity 0.35s ease; }
.nf-top-left { display: flex; align-items: center; gap: 8px; min-width: 0; flex: 1; }
.nf-top-right { display: flex; align-items: center; gap: 6px; flex-shrink: 0; pointer-events: auto; }
.nf-live-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--accent); flex-shrink: 0; animation: blink 1.4s ease-in-out infinite; }
.nf-title { font-family: var(--mono); font-size: 11px; color: var(--text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 400px; }
.vjs-user-inactive .nf-top-overlay { opacity: 0; pointer-events: none; }
```

- [ ] **Step 3: Update `.vjs-control-bar` background to gradient**

Find the `.vjs-control-bar` rule (around line 189):

```css
.vjs-control-bar { background: rgba(10,10,15,.92) !important; z-index: 2; }
```

Replace with:

```css
.vjs-control-bar { background: linear-gradient(to top, rgba(0,0,0,0.92), transparent) !important; z-index: 2; padding-bottom: 8px !important; }
```

- [ ] **Step 4: Remove retired CSS rules**

Delete these rules (now unused — elements removed from DOM):
- `.player-bar { ... }`
- `.live-dot { ... }` and `@keyframes blink` — **keep `blink` keyframe** (reused by `.nf-live-dot`) but delete `.live-dot` selector
- `.player-bar-title { ... }`
- `.player-bar-badges { ... }`

---

### Task 9: Run dev server and verify all 5 features

- [ ] **Step 1: Start dev server**

```bash
cd frontend && npm run dev
```

Open `http://localhost:5173` in browser.

- [ ] **Step 2: Verify Task 1 — MKV loading**

1. Paste an MKV magnet link, fetch files, select an `.mkv` file
2. Click ▶ Load Stream
3. Expected: button shows `↻ Starting…` (disabled), loading card appears with spinner + "Starting MKV stream…" + file name
4. After remux starts: loading card disappears, player appears

- [ ] **Step 3: Verify Task 2 + 3 — File title**

1. With any file playing, look at top-left of the video
2. Expected: live dot + file name (e.g. `Movie.Name.2024.1080p.mkv`), NOT the fileId (`abc123:0`)
3. Switch to a different file in the dropdown, click Load Stream — expected: new file name shows

- [ ] **Step 4: Verify Task 4 — Seek hover fix (MKV only)**

1. With an MKV stream playing, seek to ~30s
2. After seek, hover mouse over the progress bar without clicking
3. Expected: bottom-left current time display stays at ~30s, does NOT flicker to 0

- [ ] **Step 5: Verify Task 5 — Netflix overlay**

1. Move mouse over video: overlay (title + badges + CC/stats buttons) fades in at top, control bar appears at bottom with gradient
2. Stop moving: both fade out after ~3s
3. Click stats button in top overlay: StatusPanel slides in above video
4. Click CC button in top overlay: SubtitlePanel slides in
5. No separate player-bar visible above the video

- [ ] **Step 6: Final TypeScript check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: clean.
