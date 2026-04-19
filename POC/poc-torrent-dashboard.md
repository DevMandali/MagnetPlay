# POC: Torrent Status Widget + Management Dashboard

**Date:** 2026-04-19  
**Branch:** feature/torrent-management  
**Author:** Purnay Barge

---

## Overview

Two independent feature groups:

| Feature | Scope | Estimated Tokens |
|---------|-------|-----------------|
| **F1** — Player Status Widget | Proto + Go + Spring + React component | ~20k–25k |
| **F2i** — Torrent Management Page | Proto + Go + Spring + React page | ~35k–45k |
| **F2ii** — Session Tracking | Spring SessionManager + React tables | ~25k–35k |
| **Total** | All features end-to-end | **~80k–105k** |

---

## F1: Torrent Status Hoverable Button (Player Bar)

### What it does
Hoverable button on the video player control bar (same pattern as the subtitle panel button).  
On hover/click — shows a floating panel with:
- Selected file total size
- Downloaded bytes + percentage
- Current download speed (bytes/s, rolling 5s window)

### Feasibility
**HIGH** — `anacrolix/torrent` exposes exactly what's needed:
- `file.BytesCompleted()` → downloaded bytes for that file
- `file.Length()` → total file size  
- `torrent.Stats().BytesReadUsefulData` → global throughput (use delta over time for speed)
- Speed: track bytes-completed delta at 1s intervals in Go, store last 5 readings for rolling average

### Tech Approach

**Proto — new RPC + messages:**
```proto
message TorrentFileStats {
  string file_id           = 1;
  int64  total_size        = 2;
  int64  downloaded_bytes  = 3;
  double download_speed_bps = 4;   // rolling 5-second average
  double completion_pct    = 5;
}

message GetTorrentStatsRequest {
  string info_hash = 1;
  string file_id   = 2;
}

message GetTorrentStatsResponse {
  TorrentFileStats stats = 1;
}

// added to TorrentService:
rpc GetTorrentStats(GetTorrentStatsRequest) returns (GetTorrentStatsResponse);
```

**Go — `service.go` additions:**
- `GetTorrentStats()` method reads file.BytesCompleted() and the speed tracker
- `SpeedTracker` struct: ring buffer of (timestamp, bytesCompleted) samples, compute delta/interval

**Spring — new endpoint:**
```
GET /v1/torrent/stats/{infoHash}?fileId=abc123:0
```
Returns: `{ fileId, totalSize, downloadedBytes, completionPct, speedBps }`  
Polling from frontend: every 2 seconds via `setInterval`.

**Frontend — `StatusPanel.tsx`:**
- Same architectural pattern as `SubtitlePanel.tsx`
- Button in player controls bar (icon: activity/wifi)
- Floating panel appears on click (toggle, same as subtitle panel)
- Shows three stats in a clean grid; auto-refreshes every 2s via interval
- Stops polling when panel closed or player unmounted

### Risks
- Speed accuracy: anacrolix Stats() is global per torrent, not per file. Workaround: track per-file BytesCompleted() delta in Go with a goroutine.
- Download speed = 0 when torrent is fully seeded locally. Show "Complete" instead.

---

## F2i: Torrent Management Page

### What it does
New `/torrents` page (React route or wizard step) showing a management table for all torrents the Go client knows about:
- Name, status (Active/Paused/Stopped), total size, downloaded, speed, progress bar
- **Pause** button: stops downloading, keeps metadata
- **Delete** button: drops torrent and optionally deletes files from disk

### Feasibility
**HIGH** for List + Delete. **MEDIUM** for Pause.

**Pause nuance:** `anacrolix/torrent` has no first-class "pause" API.  
Workaround: call `file.SetPriority(torrent.PiecePriorityNone)` on all files → stops piece scheduling = effective pause.  
Resume: call `file.Download()` on each file again.  
Repository needs a `paused` flag per torrent to track state.

**Delete:** `torrent.Drop()` removes from client. Files on disk: `os.RemoveAll(dataDir + infoHash)` if `deleteFiles=true`.

### Proto additions
```proto
enum TorrentState {
  TORRENT_ACTIVE  = 0;
  TORRENT_PAUSED  = 1;
  TORRENT_STOPPED = 2;
}

message TorrentListItem {
  string      torrent_id         = 1;
  string      name               = 2;
  TorrentStatus status           = 3;   // existing enum
  TorrentState  state            = 4;   // new: active/paused/stopped
  int64       total_size         = 5;
  int64       downloaded_bytes   = 6;
  double      completion_pct     = 7;
  double      download_speed_bps = 8;
  repeated FileInfo files        = 9;
}

message ListTorrentsRequest  {}
message ListTorrentsResponse { repeated TorrentListItem torrents = 1; }

message PauseTorrentRequest  { string info_hash = 1; }
message PauseTorrentResponse { bool success = 1; string message = 2; }

message ResumeTorrentRequest  { string info_hash = 1; }
message ResumeTorrentResponse { bool success = 1; string message = 2; }

message DeleteTorrentRequest  { string info_hash = 1; bool delete_files = 2; }
message DeleteTorrentResponse { bool success = 1; string message = 2; }

// Added to TorrentService:
rpc ListTorrents(ListTorrentsRequest)   returns (ListTorrentsResponse);
rpc PauseTorrent(PauseTorrentRequest)   returns (PauseTorrentResponse);
rpc ResumeTorrent(ResumeTorrentRequest) returns (ResumeTorrentResponse);
rpc DeleteTorrent(DeleteTorrentRequest) returns (DeleteTorrentResponse);
```

**Spring endpoints:**
```
GET    /v1/torrent/list
POST   /v1/torrent/pause/{infoHash}
POST   /v1/torrent/resume/{infoHash}
DELETE /v1/torrent/{infoHash}?deleteFiles=false
```

**Frontend — `TorrentsPage.tsx`:**
- New route/tab accessible from main UI (gear icon or top nav)
- Table columns: Name | Status | Progress Bar | Speed | Size | Actions
- Polling: every 3s auto-refresh via `setInterval`
- Confirm dialog before delete
- Pause becomes Resume when already paused

### Risks
- anacrolix pause workaround may not halt peer connections immediately (peers may still push data for buffered pieces). Acceptable for MVP.
- `TorrentInfo` in `repository.go` needs `Paused bool` field + mutex-guarded mutators.

---

## F2ii: Session Tracking (Spring Side)

### What it does
Track every streaming HTTP connection (seek = new connection with new byte range):
- Session ID (UUID), Info Hash, File ID, Client IP, User Agent
- Start time, end time (or "active"), bytes served, seek count
- Stored in memory: active map + ring buffer of last 100 closed sessions

New management table on the dashboard page — two tabs: **Active Sessions** | **Closed Sessions**

### Feasibility
**HIGH** — Spring WebFlux `Flux` lifecycle gives exact hooks:
- `doOnSubscribe` → session created
- `doOnNext` → bytes served counter
- `doOnCancel` / `doOnComplete` / `doOnError` → session closed

Seek tracking: each seek is a new HTTP range request = new Flux subscription = new session record linked by `(infoHash + fileId + clientIP)` to a logical "viewer session". Track seek count by grouping by viewer key.

### Architecture
```
TorrentController.stream()
    → SessionManager.openSession()        → generates UUID, stores in activeMap
    → Flux pipeline with doOn* callbacks
        doOnNext(chunk)   → session.addBytes(chunk.length)
        doOnCancel/Complete → SessionManager.closeSession()
    → returns Flux to client

GET /v1/sessions         → SessionManager.listAll()
GET /v1/sessions/active  → SessionManager.listActive()
```

**`SessionManager.java`:**
```java
@Component
public class SessionManager {
    private final ConcurrentHashMap<String, StreamingSession> active = new ConcurrentHashMap<>();
    private final Deque<StreamingSession> closed = new ArrayDeque<>(); // capped at 100
    
    public StreamingSession openSession(String infoHash, String fileId, String clientIp, String userAgent, long startByte) { ... }
    public void closeSession(String sessionId, CloseReason reason) { ... }
    public List<StreamingSession> listActive() { ... }
    public List<StreamingSession> listAll() { ... }
}
```

**`StreamingSession.java` (record):**
```java
public record StreamingSession(
    String sessionId,
    String infoHash,
    String fileId,
    String clientIp,
    String userAgent,
    Instant startTime,
    Instant endTime,       // null if active
    long startByte,
    AtomicLong bytesServed,
    SessionStatus status   // ACTIVE, COMPLETED, CANCELLED, ERROR
) {}
```

**Spring endpoints:**
```
GET /v1/sessions         → all sessions (active + last 100 closed)
GET /v1/sessions/active  → active only
```

**Frontend — `SessionsPanel.tsx`:**
- Two tables side by side or tabs: Active | Closed
- Active: SessionID (short), File, Client IP, Duration, Bytes Served, Start Byte
- Closed: same + End Time + Close Reason
- Auto-refresh every 5s

### Risks
- Client IP behind reverse proxy: extract from `X-Forwarded-For` header, not `RemoteAddr`
- Memory: ring buffer for closed sessions prevents unbounded growth
- Multiple range requests from same browser = multiple sessions (this is correct; VideoJS does this for buffering)

---

## Recommended Phasing

| Phase | Features | Estimated Tokens | Can ship independently? |
|-------|----------|-----------------|------------------------|
| **Phase 1** | F1 (Status Widget) | ~22k | YES — player UX improvement |
| **Phase 2** | F2i (Management Page) | ~40k | YES — ops/debug utility |
| **Phase 3** | F2ii (Session Tracking) | ~30k | YES — add to Phase 2 page |

---

## Dependencies Map

```
F1 needs:  new proto RPC + Go SpeedTracker + Spring stats endpoint + React StatusPanel
F2i needs: new proto RPCs + Go pause/resume/delete + Spring list/pause/delete endpoints + React TorrentsPage
F2ii needs: Spring SessionManager + TorrentController changes + React SessionsPanel (no proto changes)
```

F2ii has no Go-side changes — fully Spring + Frontend. Fastest to ship after F2i page exists.
