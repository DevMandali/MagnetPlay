# MagnetPlay — End-to-End Flow

> Peer-to-peer video streaming: paste a magnet link → watch immediately.  
> Three layers connected by HTTP REST and gRPC server-side streaming.

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     BROWSER (frontend)                       │
│          React 18 + Video.js 8  ·  index.html (CDN)        │
└────────────────────────┬────────────────────────────────────┘
                         │  HTTP REST  (port 8080)
                         │  GET  POST  Range requests
┌────────────────────────▼────────────────────────────────────┐
│              SPRING BOOT  (mp-spring · Java 17)              │
│              WebFlux — reactive, non-blocking                │
│                                                              │
│   TorrentController  →  TorrentService  →  gRPC client      │
└────────────────────────┬────────────────────────────────────┘
                         │  gRPC server-side streaming
                         │  (port 50051)
┌────────────────────────▼────────────────────────────────────┐
│              GO SIDECAR  (go-server)                         │
│              anacrolix/torrent BitTorrent engine             │
│                                                              │
│   TorrentService  →  Repository  →  torrent client          │
└────────────────────────┬────────────────────────────────────┘
                         │  BitTorrent protocol
                         │  (DHT + Trackers)
┌────────────────────────▼────────────────────────────────────┐
│                      PEER NETWORK                            │
└─────────────────────────────────────────────────────────────┘
```

---

## Flow 1 — Add Torrent

User pastes a magnet link into the UI.

```
User  ──────────────────────────────────────────────────────►
      pastes "magnet:?xt=urn:btih:..."

Frontend  ──────────────────────────────────────────────────►
          POST /v1/torrent/add
          { "magnet": "magnet:?xt=urn:btih:..." }

Spring  ────────────────────────────────────────────────────►
        TorrentController.addTorrent()
          └─ TorrentService.addTorrentToSession()
               └─ gRPC TorrentRequest { magnetUrl }

Go  ────────────────────────────────────────────────────────►
    service.AddTorrent()
      ├─ repo.GetOrAdd()          → deduplicate by infoHash
      ├─ anacrolix.AddMagnet()    → joins DHT / trackers
      ├─ t.GotInfo()  ⏳          → waits up to 180 s for metadata
      └─ toFileInfoList()         → video files only
                                    fileId = "{infoHash}:{fileIndex}"

Go  ◄───────────────────────────────────────────────────────
    TorrentResponse {
        torrentId, name,
        files: [{ fileId, fileName, size, mimeType }]
    }

Spring  ◄───────────────────────────────────────────────────
        HTTP 201 Created  →  TorrentAddResponse (JSON)

Frontend  ◄─────────────────────────────────────────────────
          renders file list → user selects a video
```

---

## Flow 2 — Stream Video (initial play)

User clicks Play on a file.

```
User  ──────────────────────────────────────────────────────►
      clicks Play

Video.js  ──────────────────────────────────────────────────►
          GET /v1/torrent/stream/{infoHash}?fileId={id}
          Range: bytes=0-

Spring  ────────────────────────────────────────────────────►
        TorrentController.stream()
          ├─ service.getFileInfo()          gRPC FileInfoRequest
          │    └─ returns: size, mimeType
          └─ service.streamFile()           gRPC StreamRequest
               └─ maps FileChunk → DataBuffer (Flux)
                  .onBackpressureBuffer(32) ← max 8 MB buffer

Go  ────────────────────────────────────────────────────────►
    service.StreamFile()
      ├─ repo.GetFile()
      ├─ f.NewReader()
      │    ├─ reader.SetResponsive()   → fetch pieces reader needs NOW
      │    └─ reader.SetReadahead(20MB)→ pre-buffer ~5–10 s ahead
      ├─ prioritize()                  → see piece strategy below
      └─ loop: io.ReadFull()
               └─ Send FileChunk { data[256 KB], offset, isLast }

Go  ◄───────────────────────────────────────────────────────
    stream of FileChunk messages (256 KB each, server-side gRPC)

Spring  ◄───────────────────────────────────────────────────
        HTTP 206 Partial Content
          Content-Type:   video/mp4
          Accept-Ranges:  bytes
          Content-Range:  bytes 0–N/Total
          Content-Length: N
          Body: reactive chunked byte stream

Video.js  ◄─────────────────────────────────────────────────
          decodes + renders video in browser
```

---

## Flow 3 — Seek

User drags the seekbar to a new position (e.g. 2 min 30 s).

```
Video.js
  ├─ reads moov atom sample table (already buffered)
  ├─ converts timestamp → byte offset (e.g. 45 000 000)
  └─ GET /stream/{infoHash}?fileId=...
     Range: bytes=45000000-
          │
          ▼
Spring  → new StreamRequest { startByte=45_000_000, endByte=fileSize }
          │
          ▼
Go      → reader.Seek(45_000_000, io.SeekStart)
        → re-prioritize pieces around new offset
        → SetResponsive() ensures anacrolix fetches those pieces first
        → streams from new offset
          │
          ▼
Browser → playback resumes at seeked position
          (previous stream cancelled — Spring logs "stream cancelled")
```

---

## Key Data Shapes

### File ID

```
"{infoHash}:{fileIndex}"

Example:  "abc123def456:2"
```

### gRPC `StreamRequest`

| Field | Type | Notes |
|---|---|---|
| `torrentId` | string | infoHash |
| `fileId` | string | `{infoHash}:{fileIndex}` |
| `startByte` | int64 | inclusive |
| `endByte` | int64 | **exclusive** in gRPC; HTTP Range is inclusive → Spring sends `endByte + 1` |

### gRPC `FileChunk`

| Field | Type | Notes |
|---|---|---|
| `data` | bytes | 256 KB per chunk |
| `offset` | int64 | byte position in file |
| `isLast` | bool | signals end of stream |

---

## Piece Priority Strategy

When streaming starts (or after a seek), Go prioritizes torrent pieces in three tiers:

```
Torrent pieces across the file:

 ┌───────────────────────────────────────────────────────────┐
 │ [0][1][2][3][4] · · · · · · · · · [N-5][N-4][N-3][N-2][N]│
 └───────────────────────────────────────────────────────────┘
   ↑___________↑                       ↑____________________↑
   PriorityNow                          PriorityNow
   moov atom                            end metadata / seek table
   (MP4 must read this                  (needed for accurate
    before it can play)                  seeking anywhere in file)

                ↓ requested byte range ↓
         [startPiece · · · · · · endPiece]
          ↑_______________________________↑
          PriorityNext
          current playback window
```

| Tier | Pieces | Reason |
|---|---|---|
| `PriorityNow` | First 5 | moov atom — MP4 cannot play without this |
| `PriorityNow` | Last 6 | end metadata — required for accurate seeks |
| `PriorityNext` | Requested range | current playback window |

> **Why moov matters:** MP4 files store the sample index (`moov` atom) at the start or end.  
> Without it the player cannot map timestamps to byte offsets — playback stalls.

---

## Backpressure & Flow Control

```
Go sidecar          Spring Boot              Browser / Video.js
────────────        ───────────────          ──────────────────
produces chunks     onBackpressureBuffer(32) consumes chunks
256 KB each    →    buffers up to 8 MB   →   at its own pace
(fast — P2P)        if browser is slow       (variable — network)
```

Spring's `.onBackpressureBuffer(32)` absorbs bursts from the Go sidecar without
dropping data or applying back-pressure all the way to the torrent reader.
