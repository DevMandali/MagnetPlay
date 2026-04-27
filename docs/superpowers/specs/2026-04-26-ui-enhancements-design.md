# UI Enhancements Design — 2026-04-26

## Scope

5 UI/UX improvements to MagnetPlay frontend:
1. MKV stream loading state
2. File title in player (not fileId)
3. Video name overlay on player
4. Fix seek progress time display on hover
5. Netflix-style player overlay redesign

---

## Task 1 — MKV Stream Loading State

**Problem:** Clicking "Load Stream" on an MKV file fires a remux API call that can take several seconds with no feedback.

**Solution:**
- Add `streamLoading: boolean` state to `App.tsx`
- `handlePlay()`: set `true` before remux fetch, `false` in `finally` (both success and error paths)
- While `streamLoading && !active`: render a loading card in the player area — spinner + "Starting MKV stream…" + file name preview. Same visual style as existing `fetch-overlay`.
- `▶ Load Stream` button: disabled + shows spinner icon during load

**Files:** `frontend/src/App.tsx`, `frontend/src/App.css`

---

## Task 2 — File Title in Player

**Problem:** Player bar title shows `fileId` (e.g. `abc123:0`) instead of the human-readable file name.

**Solution:**
- Add `fileName: string` to `ActivePlayer` type in `frontend/src/types/index.ts`
- `handlePlay()` in `App.tsx`: set `fileName: draft.fileName` in both MKV and non-MKV branches
- `draft.fileName` is already populated correctly during `handleFetchFiles()` from the `namePart` of the label split

**Files:** `frontend/src/types/index.ts`, `frontend/src/App.tsx`

---

## Task 3 — Video Name Overlay

**Problem:** No visual indicator of which file is playing on the video surface.

**Solution:**
- `VideoPlayer` receives `fileName: string` prop
- Portal a top overlay div into `playerEl` (same pattern as `NetflixSkipOverlay`)
- Overlay shows file name top-left, auto-visible with user activity
- Visibility controlled by CSS tied to `.vjs-user-active` / `.vjs-user-inactive` classes (syncs with control bar)
- On mount: video.js `userActive(true)` is already called — name shows for the standard inactivity timeout (~3s), then fades

**Files:** `frontend/src/components/VideoPlayer.tsx`, `frontend/src/App.css`

---

## Task 4 — Fix Seek Progress Time on Hover

**Problem:** `SeekBar.prototype.handleMouseMove` patch in `VideoPlayer.tsx` applies seek logic for ALL calls including hover (mouseDown=false). This calls `player.currentTime(target)` on hover, resetting the current time display to 0 (browser currentTime = 0 after MKV seek, offset not re-applied yet).

**Root cause:** Missing `mouseDown` guard in the patch.

**Fix:**
```ts
SeekBarComp.prototype.handleMouseMove = function(this: any, event: any, mouseDown = false) {
  const dur: number = this.player_?._mpDuration;
  if (mouseDown && dur > 0) {
    const distance: number = this.calculateDistance(event);
    const target = Math.max(0, Math.min(dur, distance * dur));
    this.player_._mpPendingSeek = target;
    this.player_.currentTime(target);
    return;
  }
  return origMove.call(this, event, mouseDown);
};
```

Hover (mouseDown=false) passes through to original video.js — tooltip updates from duration × mouse position correctly. `getPercent` patch is unaffected and still drives correct fill position.

**Files:** `frontend/src/components/VideoPlayer.tsx`

---

## Task 5 — Netflix-Style Player Overlay Redesign

**Architecture decision:** Option 1 (CSS-driven). video.js manages user activity state via `.vjs-user-active`/`.vjs-user-inactive` CSS classes. All overlay content syncs to these classes — no extra React state or event listeners needed.

### VideoPlayer new props

```ts
fileName: string
moovBadge: { cls: string; icon: string; spin: boolean; label: string } | null
peerStats: { seeders: number; peers: number; trackers: number } | null
showStatusPanel: boolean
onToggleStatusPanel: () => void
showSubPanel: boolean
onToggleSubPanel: () => void
subtitleTrackCount: number
```

### Top overlay (portal into playerEl)

- `position: absolute; top: 0; left: 0; right: 0; z-index: 5; pointer-events: none`
- Background: `linear-gradient(to bottom, rgba(0,0,0,0.75), transparent)`
- Left: live-dot + `fileName` (truncated, monospace)
- Right: moov badge + MIME badge + seeders badge + stats toggle button + CC toggle button (pointer-events: auto on buttons)
- CSS transition on opacity tied to `.vjs-user-active` parent class

### Control bar restyle

- Background: `linear-gradient(to top, rgba(0,0,0,0.85), transparent)` — replaces flat `rgba(10,10,15,.92)`
- Bottom padding increased slightly for breathing room
- All existing buttons, icons, behavior unchanged

### App.tsx changes

- Remove `.player-bar` div and all child elements (`.live-dot`, `.player-bar-title`, `.player-bar-badges`)
- Remove `.player-bar-badges` buttons (stats toggle, CC toggle) — now in VideoPlayer overlay
- Pass new props to `<VideoPlayer>`: `fileName`, `moovBadge`, `peerStats`, `showStatusPanel`, `onToggleStatusPanel`, `showSubPanel`, `onToggleSubPanel`, `subtitleTrackCount`
- `StatusPanel` and `SubtitlePanel` remain above `<VideoPlayer>` in `.player-card` DOM order — triggered by buttons now in overlay
- `kbd-hints` remains below video, no change

### CSS changes

- Remove `.player-bar`, `.player-bar-title`, `.player-bar-badges`, `.live-dot` rules (retired)
- Add `.nf-top-overlay` rules: absolute position, gradient, flex layout, opacity transition
- Update `.vjs-control-bar` background to gradient

**Files:** `frontend/src/App.tsx`, `frontend/src/components/VideoPlayer.tsx`, `frontend/src/App.css`

---

## Summary of file changes

| File | Tasks |
|------|-------|
| `frontend/src/types/index.ts` | 2 |
| `frontend/src/App.tsx` | 1, 2, 5 |
| `frontend/src/App.css` | 1, 3, 5 |
| `frontend/src/components/VideoPlayer.tsx` | 3, 4, 5 |
