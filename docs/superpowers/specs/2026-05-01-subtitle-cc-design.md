# Subtitle & CC Enhancement — Design Spec
**Date:** 2026-05-01  
**Branch:** feature/torrent-management  
**Status:** Approved

---

## Problem

Two subtitle sources exist but are not surfaced to the user:

1. MKV files contain embedded subtitle streams detected by FFprobe — never returned via proto or shown in CC panel.
2. Torrent packages often include external subtitle files (.srt, .vtt, .ass) — filtered out entirely by `mapper.go`.

Manual upload via CC panel works but requires the user to source subtitles externally.

---

## Goals

1. Surface MKV embedded subtitle tracks in CC panel (toggle on/off per track).
2. Allow subtitle files bundled in a torrent to auto-populate CC panel on play.
3. Keep manual subtitle upload working unchanged.
4. Support .srt, .vtt, .ass/.ssa server-side; skip bitmap codecs (PGS, DVB).
5. Work for both MP4 streaming and MKV remux paths.

---

## Non-Goals

- Subtitle rendering quality tuning (font, position, styling).
- Bitmap subtitle support (PGS/DVB — no viable text conversion without OCR).
- Subtitle search or external provider integration.

---

## Architecture

```
[Play pressed]
  │
  ├─ MKV path ──► POST /v1/torrent/remux/{hash}/start
  │                 ← subtitleTracks[] (embedded, from FFprobe)
  │                 For each non-bitmap track:
  │                   GET :8091/subtitle/embedded/{hash}/{b64fileId}/{streamIndex}
  │                   ← WebVTT text → blob URL → subtitleTracks state
  │
  ├─ Any format ──► Torrent subtitle files detected from file list
  │                   GET :8091/subtitle/file/{hash}/{b64subtitleFileId}
  │                   ← WebVTT text → blob URL → subtitleTracks state
  │
  └─ Manual upload ──► unchanged (SRT→VTT client-side, blob URL)

[CC panel] ──► same SubtitlePanel, same toggle UX, no changes
[VideoPlayer] ──► same addRemoteTextTrack() call, no changes
```

---

## Proto Changes (`backend/proto/torrent.proto`)

### New message
```proto
message SubtitleTrackInfo {
  int32  index    = 1;  // stream index within the container
  string language = 2;
  string codec    = 3;  // "ass", "subrip", "hdmv_pgs_subtitle", etc.
  string title    = 4;
}
```

### Updated messages
```proto
// FileInfoResponse: add field 8
repeated SubtitleTrackInfo subtitle_tracks = 8;

// HLSResponse: add field 5
repeated SubtitleTrackInfo subtitle_tracks = 5;

// FileType enum (new)
enum FileType { VIDEO = 0; SUBTITLE = 1; }

// FileInfo: add field 4
FileType file_type = 4;
```

---

## Go Backend (`backend/go-server/`)

### `internal/torrent/mapper.go`
- Add subtitle extension allowlist: `.srt`, `.vtt`, `.ass`, `.ssa`, `.sub`
- Matching files get `FileType: pb.FileType_SUBTITLE`, added to repo with `PiecePriorityNone`
- Returned in `TorrentResponse.Files` alongside video files

### `internal/torrent/service.go`
- `GetFileInfo`: populate `resp.SubtitleTracks` from `probe.Subtitles`, skip bitmap codecs
- `StartRemux`: populate `resp.SubtitleTracks` from `probe.Subtitles`, skip bitmap codecs
- Bitmap codecs to skip: `hdmv_pgs_subtitle`, `dvb_subtitle`, `dvb_teletext`, `pgssub`

### `internal/hls/subtitle_handler.go` (new file)

**Endpoint 1 — embedded subtitle extraction:**
```
GET /subtitle/embedded/{infoHash}/{b64fileId}/{streamIndex}
```
- Acquires anacrolix reader for the video file (same as rawfile)
- Spawns FFmpeg: `ffmpeg -i pipe:0 -map 0:s:{streamIndex} -f webvtt pipe:1`
- Timeout: 30s
- Response: `Content-Type: text/vtt`
- CORS headers: same as remux handler

**Endpoint 2 — torrent subtitle file serving:**
```
GET /subtitle/file/{infoHash}/{b64fileId}
```
- Acquires anacrolix reader for the subtitle file
- `.vtt` files: pipe bytes directly (no FFmpeg)
- All other formats: FFmpeg `ffmpeg -i pipe:0 -f webvtt pipe:1`
- Timeout: 30s
- Response: `Content-Type: text/vtt`

Both endpoints registered in `grpc/server.go` `StartServer()`.  
`SubtitleHandler` needs access to the `Repository` to resolve fileId → anacrolix file reader.

---

## Spring Backend (`backend/mp-spring/`)

Thin pass-through — no new REST endpoints. Subtitle files served directly from Go HLS server (port 8091), same as rawfile and remux.

### New DTO
```java
// SubtitleTrackInfo.java
public record SubtitleTrackInfo(int index, String language, String codec, String title) {}
```

### `TorrentGrpcClient`
- Map proto `SubtitleTrackInfo` → Java DTO in `getFileInfo()` and `startRemux()` responses
- Map proto `FileType` → string `"VIDEO"` / `"SUBTITLE"` in `FileInfo` mapping

### `TorrentService` + `TorrentController`
- Pass `subtitleTracks` list through to JSON response
- `FileInfo` response includes `fileType` string field

---

## Frontend (`frontend/src/`)

### `types/index.ts`
```typescript
export interface SubtitleTrackInfo {
  index: number;
  language: string;
  codec: string;
  title: string;
}

// Update RemuxStartResponse
subtitleTracks: SubtitleTrackInfo[];

// Update ActivePlayer
subtitleTracks: SubtitleTrackInfo[];

// Update TorrentFile
fileType: 'VIDEO' | 'SUBTITLE';
```

### `App.tsx` — `handleFetchFiles`
- Separate files into `videoFiles` (fileType === 'VIDEO') and `subtitleFiles` (fileType === 'SUBTITLE')
- File select dropdown shows only `videoFiles`
- `subtitleFiles` stored in separate state: `torrentSubtitleFiles`

### `App.tsx` — `handlePlay`
```
1. Clear subtitleTracks state
2. For MKV:
   - POST /remux/start → data.subtitleTracks (embedded, non-bitmap only)
   - For each embedded track:
       fetch GET :8091/subtitle/embedded/{hash}/{b64fileId}/{index}
       → blob URL → push to subtitleTracks (inactive, label = "[EMB] {title|language}")
3. For ALL formats:
   - For each torrentSubtitleFile in same torrent:
       fetch GET :8091/subtitle/file/{hash}/{b64subtitleFileId}
       → blob URL → push to subtitleTracks (inactive, label = filename without extension)
4. Manual upload: unchanged
```

### `SubtitlePanel` — no changes
### `VideoPlayer` — no changes

---

## Subtitle Label Conventions

| Source | Label format |
|--------|-------------|
| MKV embedded | `[EMB] {title}` or `[EMB] {language}` if no title |
| Torrent file | filename without extension (e.g. `Movie.en`) |
| Manual upload | user-entered label (unchanged) |

---

## Error Handling

- FFmpeg fails on embedded extraction → skip that track silently, log warning
- FFmpeg fails on torrent file → skip that file silently, log warning
- Partial torrent (pieces not yet downloaded) → anacrolix reader blocks until available; 30s timeout guards against stall
- Bitmap codec slips through → FFmpeg will error → caught by above rule

---

## Codec Skip List (Go)

```go
var bitmapSubtitleCodecs = map[string]bool{
    "hdmv_pgs_subtitle": true,
    "dvb_subtitle":      true,
    "dvb_teletext":      true,
    "pgssub":            true,
    "xsub":              true,
}
```

---

## File Serving Flow Summary

```
Torrent file .srt/.vtt/.ass  →  /subtitle/file/...  →  Go FFmpeg  →  WebVTT text  →  blob URL  →  <track>
MKV embedded sub stream      →  /subtitle/embedded/...  →  Go FFmpeg  →  WebVTT text  →  blob URL  →  <track>
MP4 byte stream              →  /v1/torrent/stream/...  →  gRPC StreamFile  →  raw bytes  →  <video>
MKV remux stream             →  /remux/...  →  Go FFmpeg  →  fMP4 bytes  →  <video>
```

Video and subtitle streams are fully independent HTTP connections.
