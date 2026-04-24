# MKV HLS Streaming Design

**Date:** 2026-04-24  
**Branch:** feature/torrent-management  
**Status:** Approved — ready for implementation planning

---

## Problem

MagnetPlay currently streams only MP4-compatible containers. MKV files (`video/x-matroska`) are rejected by browsers. Browsers also need `Accept-Ranges` + known file size for seeking, which breaks for on-the-fly transcoded output. fMP4 pipe was considered and rejected: browser can't byte-range seek into unknown-size pipe output.

---

## Solution

Transcode MKV → HLS (HTTP Live Streaming) in the Go sidecar via FFmpeg. VHS (videojs-http-streaming) is already in the project and handles HLS natively. Segments served directly from Go's lightweight HTTP server on port 8091. Spring Boot remains thin — one new trigger endpoint.

---

## Architecture

```
Browser (VHS)
  │  1. POST /v1/torrent/hls/{hash}/start?fileId=X&t=0
  │     ← { manifestUrl: "http://localhost:8091/hls/{hash}/{fileId}/playlist.m3u8" }
  │  2. VHS fetches manifest  → Go HTTP :8091
  │  3. VHS fetches segments  → Go HTTP :8091
  ▼
Spring Boot  ← one new endpoint pair (start/stop trigger only)
  │  gRPC: StartHLS / StopHLS
  ▼
Go sidecar
  ├─ gRPC  :50051  (adds StartHLS + StopHLS RPCs)
  └─ HTTP  :8091   (new — serves manifest + .ts segments with CORS)
       └─ internal/ffmpeg/binary.go     EnsureFFmpeg() download + path resolution
       └─ internal/hls/transcoder.go    FFmpeg job manager (per fileId)
       └─ internal/hls/server.go        HTTP handler (manifest + segments)
```

MP4 path is **completely unchanged**.

---

## Proto Changes

```proto
// New RPCs
rpc StartHLS (HLSRequest) returns (HLSResponse);
rpc StopHLS  (HLSRequest) returns (HLSResponse);

message HLSRequest {
  string info_hash     = 1;
  string file_id       = 2;
  double seek_time_sec = 3;  // 0 = start of file
}

message HLSResponse {
  string manifest_url = 1;  // e.g. "http://localhost:8091/hls/{hash}/{fileId}/playlist.m3u8"
  bool   success      = 2;
}

// Extend FileInfoResponse (fields 6–7, MKV only)
message FileInfoResponse {
  string file_name   = 1;
  string file_path   = 2;
  int64  total_size  = 3;
  string mime_type   = 4;
  bool   is_ready    = 5;
  double duration_sec       = 6;  // 0 for MP4; populated by ffprobe for MKV
  repeated AudioTrack audio_tracks = 7;  // empty for MP4
}

message AudioTrack {
  int32  index    = 1;  // stream index within the MKV
  string language = 2;  // ISO 639-2 code, e.g. "eng"
  string codec    = 3;  // e.g. "aac", "dts", "ac3"
  string title    = 4;  // track name from MKV metadata
}
```

`StreamRequest` is **not changed** — byte-range path for MP4 untouched.

---

## Data Flow

### MKV Initial Load

```
1. POST /add
   mapper.go: .mkv passes video/ prefix → fileId registered (no change)

2. GET /stream/{hash}?fileId=X  (frontend probes)
   → GetFileInfo gRPC
   → Go: filepath.Ext == ".mkv" → ffprobe on disk path (cached on TorrentInfo after first call)
   → FileInfoResponse{ mime="video/mp4", duration_sec=7243.0,
                       audio_tracks=[{0,"eng","aac","English"},{1,"jpn","dts","Japanese"}] }

3. Frontend: file name ends with .mkv (from AddTorrent FileInfo.name) → MKV mode
   duration_sec > 0 → show seekbar; duration_sec == 0 (ffprobe failed) → play without seekbar
   POST /v1/torrent/hls/{hash}/start?fileId=X&t=0
   → grpc.StartHLS{info_hash, file_id, seek_time_sec=0}
   → Go:
       a. mkdir ./downloads/<hash>/hls/<fileId>/
       b. open torrent reader, seek to byte 0
       c. spawn FFmpeg (see FFmpeg args below)
       d. poll every 500ms until playlist.m3u8 has ≥ 2 segments (~8s buffered)
       e. return HLSResponse{ manifest_url, success=true }
   → Spring: return { manifestUrl } to frontend

4. Frontend: player.src({ src: manifestUrl, type: "application/x-mpegURL" })
   VHS fetches manifest → reads #EXT-X-TARGETDURATION + segment count → full seekbar rendered
   VHS fetches seg000.ts, seg001.ts ... → playback starts
```

### MKV Seek (user clicks T=600s)

```
1. VHS fires "seeking" event
   → frontend debounces 300ms
   → POST /v1/torrent/hls/{hash}/start?fileId=X&t=600

2. Go HLS transcoder (per-fileId mutex):
   a. kill existing FFmpeg
   b. delete seg*.ts from segment dir
   c. spawn new FFmpeg with -ss 600
   d. wait ≥ 1 segment ready
   e. return same manifest_url (file overwritten in place)

3. Frontend: player.src({ src: manifestUrl, type: "application/x-mpegURL" })
   VHS re-fetches manifest → seg000.ts now contains T=600
   ~1-2s rebuffer, then resumes
```

### Audio Track Switch

```
User selects track 1 ("Japanese") from audio switcher:
  player.audioTracks()[1].enabled = true
  VHS handles in-stream — no new HTTP request
  All tracks already muxed into every .ts segment via -map 0:a
```

### Cleanup

```
POST /v1/torrent/hls/{hash}/stop?fileId=X
→ grpc.StopHLS → kill FFmpeg → rm -rf ./downloads/<hash>/hls/<fileId>/
```

---

## FFmpeg Command

```bash
# Base (seek_time_sec == 0)
ffmpeg -i pipe:0 \
  -map 0:v:0 -map 0:a \
  -c:v copy -c:a copy \
  -f hls \
  -hls_time 4 \
  -hls_list_size 0 \
  -hls_flags delete_segments \
  -hls_segment_filename ./downloads/<hash>/hls/<fileId>/seg%03d.ts \
  ./downloads/<hash>/hls/<fileId>/playlist.m3u8

# With seek
ffmpeg -ss 600.000 -i pipe:0 \
  -map 0:v:0 -map 0:a \
  -c:v copy -c:a copy \
  ...same...

# Audio fallback (triggered if stderr contains codec error)
ffmpeg [...] -c:v copy -c:a aac [...]
```

**Audio strategy:** try `-c:a copy` (all tracks). Monitor FFmpeg stderr for lifetime of process. If stderr contains `"Invalid data found"` or `"codec not currently supported in container"`, kill and restart with `-c:a aac`. One retry only — if AAC also fails, return `success=false`.

---

## Config Changes (`config.go`)

```go
type Config struct {
    // existing fields unchanged
    FFmpegPath string  // optional override; empty = auto-download to bin/ffmpeg[.exe]
    HLSPort    int     // default 8091
    HLSDir     string  // default ./downloads — segments written to <HLSDir>/<hash>/hls/<fileId>/
}
```

`FFmpegPath` override is the Electron integration point — Electron build sets this to the bundled binary path.

---

## New Files

| File | Responsibility |
|---|---|
| `internal/ffmpeg/binary.go` | `EnsureFFmpeg()`: config path → PATH check → auto-download from evansmp/ffmpeg-builds GitHub releases. Also `EnsureFFprobe()`. Same retry/backoff pattern as `downloader.go`. |
| `internal/hls/transcoder.go` | `HLSTranscoder` struct. `Start(infoHash, fileId, seekSec)` → spawns FFmpeg, monitors stderr, returns when manifest ready. `Stop(fileId)` → kill + cleanup. Per-fileId `sync.Mutex`. |
| `internal/hls/server.go` | `net/http` mux on `:HLSPort`. Routes: `GET /hls/{hash}/{fileId}/playlist.m3u8` and `GET /hls/{hash}/{fileId}/seg{N}.ts`. CORS: `Access-Control-Allow-Origin: *`. 404 if file missing. |

---

## Modified Files

### Go sidecar

**`config/config.go`** — add `FFmpegPath`, `HLSPort`, `HLSDir` fields with defaults.

**`internal/grpc/server.go`**
- Call `ffmpeg.EnsureFFmpeg()` at startup (fatal if fails)
- Instantiate `HLSTranscoder`, start HTTP server on `HLSPort`
- Wire transcoder into `TorrentService`

**`internal/torrent/service.go`**
- `GetFileInfo`: if `.mkv` extension → run ffprobe on `<config.DataDir>/<t.Name()>/<f.Path()>`, populate `duration_sec` + `audio_tracks`, cache result on `TorrentInfo` (subsequent calls return cached value)
- Add `StartHLS(ctx, req)` and `StopHLS(ctx, req)` gRPC handlers that delegate to `HLSTranscoder`
- `StreamFile` unchanged

### Spring Boot

**`TorrentGrpcClient.java`** — add `startHLS(HLSRequest)` and `stopHLS(HLSRequest)` stub calls.

**`TorrentService.java`** — add `startHLS(infoHash, fileId, seekSec)` and `stopHLS(infoHash, fileId)`.

**`TorrentController.java`**
```java
@PostMapping("/hls/{infoHash}/start")
Mono<ResponseEntity<HLSStartResponse>> startHLS(
    @PathVariable String infoHash,
    @RequestParam String fileId,
    @RequestParam(defaultValue = "0") double t)

@DeleteMapping("/hls/{infoHash}/stop")
Mono<ResponseEntity<String>> stopHLS(
    @PathVariable String infoHash,
    @RequestParam String fileId)
```

New DTO: `HLSStartResponse(String manifestUrl)`.

### Frontend

**`types/index.ts`** — add `HLSStartResponse`, `AudioTrack`, extend `TorrentFileStats` or `FileInfoData` with `durationSec`, `audioTracks`.

**`VideoPlayer.tsx`**
- After `GetFileInfo`: if `fileName.endsWith('.mkv')` → MKV mode → call `POST /hls/start?t=0` → set VHS source
- `durationSec > 0` → show seekbar; `durationSec == 0` → hide seekbar, play without seek
- On `seeked` event (debounced 300ms): call `POST /hls/start?t=currentTime` → `player.src(...)` again
- On unmount / file change: call `DELETE /hls/stop`
- Render `<AudioTrackMenu>` only when `audioTracks.length > 1`

**`videoSetup.ts`** — register `AudioTrackMenuButton` custom VHS control bar button.

---

## Error Handling

| Error | Go | Spring | Frontend |
|---|---|---|---|
| FFmpeg/ffprobe binary missing | fatal exit at startup | N/A | N/A |
| ffprobe timeout (>10s) | return `duration_sec=0`, empty tracks | pass through | hide seekbar, play progressive |
| Audio copy codec error | kill, restart with `-c:a aac` | transparent | none |
| Audio AAC also fails | `success=false` | 503 | error toast |
| FFmpeg crash mid-stream | log, segments preserved on disk | N/A | VHS error event → retry toast |
| StartHLS timeout (>30s, no segments) | `success=false` | 503 | auto-retry after 5s |
| Disk full / mkdir fails | gRPC `codes.Internal` | 500 | error toast |
| Port 8091 conflict | fatal at startup | N/A | N/A |
| Client disconnects mid-segment | HTTP handler closes, no FFmpeg impact | N/A | N/A |

---

## Testing

### Go unit tests (no FFmpeg required)

`internal/hls/transcoder_test.go`
- `TestBuildFFmpegArgs_SeekZero` — no `-ss` flag when `seekSec=0`
- `TestBuildFFmpegArgs_SeekNonZero` — `-ss 600` present, before `-i`
- `TestBuildFFmpegArgs_AudioFallback` — `-c:a aac` in fallback variant
- `TestHLSTranscoder_ConcurrentStart_SameFileId` — second call kills first, mutex correct

`internal/hls/server_test.go`
- `TestSegmentHandler_NotFound` — 404 when segment missing
- `TestManifestHandler_CORS` — `Access-Control-Allow-Origin: *` present

`internal/ffmpeg/binary_test.go`
- `TestConfigPathOverride` — `FFmpegPath` set → no download attempted

### Go integration tests (skip if FFmpeg absent)

`internal/hls/transcoder_integration_test.go`
```go
func TestHLS_TranscodesRealFile(t *testing.T) {
    if _, err := exec.LookPath("ffmpeg"); err != nil { t.Skip("ffmpeg not in PATH") }
    // fixture: testdata/sample_10s.mkv (H.264+AAC, ~500KB)
    // assert: playlist.m3u8 exists, ≥1 seg*.ts, manifest has #EXT-X-TARGETDURATION
}
func TestHLS_AudioFallback_TriggersOnDTS(t *testing.T) {
    // fixture: testdata/sample_10s_dts.mkv
    // assert: second FFmpeg invocation used -c:a aac
}
```

Test fixtures committed to `backend/go-server/internal/hls/testdata/`.

### Spring unit tests

`TorrentHLSControllerTest.java` (follows `TorrentStatsControllerTest` pattern)
- `startHLS_returns200_withManifestUrl`
- `startHLS_returns503_whenSuccessFalse`
- `stopHLS_returns200`

### Frontend

Manual verification only (no existing frontend unit test infrastructure):
- MP4: existing player unchanged, no regression
- MKV: VHS source set, seekbar visible, audio switcher appears when >1 track
- Seek: rebuffer occurs, resumes at correct timestamp
- Cleanup: `StopHLS` fires on unmount

---

## Token Estimate for Implementation

| Layer | Files touched | Estimated input tokens | Estimated output tokens |
|---|---|---|---|
| Proto + regen | `torrent.proto` + both generated files | ~2,000 | ~1,500 |
| Go: ffmpeg binary | `internal/ffmpeg/binary.go` (new) | ~1,500 | ~1,200 |
| Go: HLS transcoder | `internal/hls/transcoder.go` (new) | ~2,000 | ~2,000 |
| Go: HLS HTTP server | `internal/hls/server.go` (new) | ~1,500 | ~800 |
| Go: service.go | ffprobe + StartHLS/StopHLS handlers | ~3,000 | ~1,000 |
| Go: server.go + config | wiring + config fields | ~2,000 | ~500 |
| Go: unit tests | 3 test files | ~2,500 | ~1,500 |
| Spring: gRPC client | `TorrentGrpcClient.java` | ~2,000 | ~600 |
| Spring: service + controller | 2 files + 1 new DTO | ~3,000 | ~800 |
| Spring: controller test | `TorrentHLSControllerTest.java` | ~2,000 | ~700 |
| Frontend: VideoPlayer | MKV detection + HLS source + seek handler | ~3,500 | ~1,200 |
| Frontend: videoSetup + types | audio button + new types | ~2,000 | ~600 |
| **Total** | **~15 files** | **~27,000** | **~12,400** |

---

## Out of Scope

- Subtitle track support in HLS (separate feature)
- Adaptive bitrate (multiple quality renditions)
- HLS encryption
- Session tracking for HLS segments (tracked at start/stop level only)
- Distributed deployment (single-machine assumption throughout)
