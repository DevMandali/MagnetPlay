# Torrent State Management — Design Spec
Date: 2026-04-27

## Overview

Four coordinated changes to improve how MagnetPlay manages torrent lifetime, logging, and cleanup:

1. **Persistent anacrolix torrents** — "delete" only removes from dashboard session; anacrolix client keeps seeding
2. **FFmpeg/FFprobe logging** — stderr errors + structured file metadata into go-server logs
3. **Shutdown-only dataDir cleanup** — files and bolt.db deleted only after `client.Close()`
4. **Remove "delete with files" option** — simplify delete to session-removal only

---

## Change 1: Torrent Persistence (session-only delete)

### Invariant
The anacrolix `*torrent.Torrent` handle is NEVER dropped except at server shutdown. "Delete" from the dashboard removes only the in-memory repository entry (Go) and clears speed trackers. The torrent continues seeding in the background.

### `service.go — DeleteTorrent`
- Remove `info.Torrent().Drop()` call
- Remove entire `deleteFiles` conditional block (and `QueueDelete` call)
- Keep: `s.repo.Remove(infoHash)` and speed tracker cleanup

### `repository.go — GetOrAdd`
- Remove `t.Drop()` on the duplicate-guard at line 90–91
- Rationale: after a dashboard delete, user may re-add the same magnet. anacrolix's `AddMagnet` returns the existing torrent handle (not dropped). `GotInfo` fires immediately (metadata already resolved). We re-register in the map. The duplicate guard was only for concurrent callers — in normal re-add flow, only one goroutine runs, so no double-drop risk.
- Keep: the double-check lock pattern itself (still needed for true concurrent add race)

### Applies equally to MP4 and MKV
Both streaming paths (direct gRPC stream for MP4, FFmpeg remux for MKV) look up the torrent via `repo.GetTorrent` / `repo.GetFile`. Since the anacrolix handle is never dropped, a re-add after dashboard delete restores those lookups instantly.

---

## Change 2: FFmpeg/FFprobe Logging

### FFprobe — `service.go:runFFprobe`
- Add `SubtitleTrack` type: `{ Index int, Language, Codec, Title string }`
- Add `Subtitles []SubtitleTrack` to `probeResult`
- Change `cmd.Output()` → separate `cmd.Stdout` pipe + `cmd.Stderr = &stderrBuf`
- After exec: log full stderr buffer at `[ffprobe-stderr]` prefix (ffprobe stderr is small — codec detection only, no per-frame noise)
- After JSON parse: log one structured summary line:
  ```
  [ffprobe] file=%s duration=%.1fs video=%s audio=%s audio_tracks=%d subtitles=%d
  ```
- Extract subtitle streams from `codec_type == "subtitle"` entries in JSON

### FFmpeg Remux — `transcoder.go:ServeHTTP`
- After `cmd.Start()`, launch goroutine: read stderr line-by-line, log only lines containing keywords `error`, `Error`, `warning`, `Warning`, `invalid`, `Invalid`, `failed`, `Failed`, `No such`
- Goroutine exits when stderr closes (FFmpeg done or cancelled)
- Prefix: `[ffmpeg-stderr hash=%s fileId=%s]`

### Torrent info logging — `service.go:AddTorrent`
- After `t.Info()` resolved, log:
  ```
  [torrent] name=%s infoHash=%s files=%d totalSize=%d
  ```

---

## Change 3: Shutdown DataDir Cleanup

### Remove runtime-delete infrastructure
- Remove `pendingDelete []string` and `deleteMu sync.Mutex` from `Repository`
- Remove `QueueDelete()` method
- Simplify `Clearup()`: only Drop all tracked torrents (no pending-delete loop)

### Shutdown sequence in `server.go`
Defers run LIFO. Register in this order so cleanup runs after `client.Close()`:

```go
// 1st defer registered = runs LAST (after client close)
defer repo.CleanupDataDir(cfg.DataDir)

// 2nd defer registered = runs 2nd-to-last
defer func() {
    if err := client.Close(); err != nil {
        log.Printf("Error closing torrent client: %v", err)
    }
}()
```

### `repository.go — CleanupDataDir(dataDir string)`
New method (replaces `Clearup` data-deletion role):
- Read all entries in `dataDir`
- For each entry: `os.RemoveAll(filepath.Join(dataDir, entry.Name()))`
- This removes torrent content subdirs AND `torrent.bolt.db` in one pass
- Keep `dataDir` folder itself (option a from design discussion)
- Log each removal; log errors but do not fatal

### `Clearup()` becomes shutdown-torrent-drop only
Called from `<-stop` handler before gRPC GracefulStop. Drops all anacrolix torrent handles so OS releases file locks before `CleanupDataDir` runs.

### Shutdown order (full sequence)
```
<-stop signal received
remuxHandler.StopAll()       — cancel FFmpeg processes
repo.Clearup()               — Drop() all torrents (releases file handles)
grpcServer.GracefulStop()    — drain in-flight RPCs
[defers unwind]:
  client.Close()             — close anacrolix client + flush bolt.db
  repo.CleanupDataDir(...)   — delete all contents of ./downloads/
```

---

## Change 4: Remove "Delete with Files" Option

### Proto (`backend/proto/torrent.proto`)
- Remove `bool delete_files = 2` from `DeleteTorrentRequest`
- Regenerate: Go (`make -f MakeFile proto`) and Java (`mvn compile`)

### Go (`service.go`)
- `DeleteTorrent`: remove `req.GetDeleteFiles()` branch entirely (already covered by Change 1)

### Spring (`TorrentController.java`)
- `DELETE /{infoHash}`: remove `@RequestParam(defaultValue = "false") boolean deleteFiles` param
- Call `service.deleteTorrent(infoHash)` (no deleteFiles arg)

### Spring (`TorrentService.java` + `TorrentGrpcClient.java`)
- Remove `deleteFiles` parameter from `deleteTorrent` method signatures
- Build `DeleteTorrentRequest` without `setDeleteFiles`

### Frontend (`TorrentsPage.tsx`)
- Replace confirm dialog (two buttons: keep/delete-files) with single confirmation:
  - Button: **"Remove from dashboard"** — calls `DELETE /{id}` (no `deleteFiles` param)
  - Button: **Cancel**
- Update `deleteTorrent(id, deleteFiles)` → `deleteTorrent(id)` — no deleteFiles arg

---

## Error Handling

- `CleanupDataDir`: log errors, never fatal — server is already shutting down
- FFmpeg stderr goroutine: exits silently when pipe closes (normal EOF)
- Re-add after delete: if `AddMagnet` fails (network issue), return error to client — same as first-add behavior

## Files Changed

| File | Change |
|------|--------|
| `backend/proto/torrent.proto` | Remove `delete_files` from `DeleteTorrentRequest` |
| `backend/go-server/internal/torrent/service.go` | DeleteTorrent simplified; runFFprobe adds stderr+subtitles; AddTorrent logs torrent info |
| `backend/go-server/internal/torrent/repository.go` | Remove pendingDelete; simplify Clearup; add CleanupDataDir; fix GetOrAdd duplicate-drop |
| `backend/go-server/internal/grpc/server.go` | Reorder defers; add CleanupDataDir defer |
| `backend/go-server/internal/hls/transcoder.go` | Add FFmpeg stderr goroutine |
| `backend/go-server/proto/` | Regenerated Go proto |
| `backend/mp-spring/.../TorrentController.java` | Remove deleteFiles param |
| `backend/mp-spring/.../TorrentService.java` | Remove deleteFiles param |
| `backend/mp-spring/.../TorrentGrpcClient.java` | Remove deleteFiles from request build |
| `frontend/src/components/TorrentsPage.tsx` | Simplify delete confirm dialog |
