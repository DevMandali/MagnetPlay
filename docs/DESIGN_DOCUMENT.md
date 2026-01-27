# MagnetPlay - Complete Design Document

**Version:** 1.0  
**Date:** January 27, 2026  
**Author:** System Architecture Team

---

## Executive Summary

This document presents the complete architectural design for a Stremio-like peer-to-peer video streaming system that enables immediate playback of torrent content without requiring complete file downloads.

### Core Features
- 🎬 **Immediate Playback**: Start streaming within 10 seconds
- ⏩ **Smart Seeking**: Forward/backward seeks with intelligent piece prioritization
- 💾 **Storage Efficient**: LRU cache with 10GB limit
- 🔄 **Progressive Download**: Stream while downloading
- 📊 **Real-time Metrics**: Live download statistics and peer information
- 🛡️ **Resilient**: Handles network failures and slow torrents gracefully

### Technology Stack
| Layer | Technology | Purpose |
|-------|-----------|---------|
| **Frontend** | React 18 + Video.js | User interface and video player |
| **Backend** | Spring Boot 3.2 (Java 17) | HTTP streaming and API |
| **Sidecar** | Go 1.21 + libtorrent | Torrent management |
| **Communication** | gRPC | Backend ↔ Sidecar IPC |
| **Deployment** | Docker Compose | Service orchestration |
| **Monitoring** | Prometheus + Grafana | Metrics and dashboards |

---

## Table of Contents

1. [System Architecture](#1-system-architecture)
2. [Component Design](#2-component-design)
3. [Data Flow & Protocols](#3-data-flow--protocols)
4. [Streaming Strategy](#4-streaming-strategy)
5. [Torrent Management](#5-torrent-management)
6. [Performance & Optimization](#6-performance--optimization)
7. [Error Handling](#7-error-handling)
8. [Security](#8-security)
9. [Monitoring](#9-monitoring)
10. [Deployment](#10-deployment)
11. [API Specifications](#11-api-specifications)
12. [Testing Strategy](#12-testing-strategy)
13. [Future Enhancements](#13-future-enhancements)

---

## 1. System Architecture

### 1.1 High-Level Architecture Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                    Browser Client (User)                      │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │          React SPA (localhost:3000)                     │  │
│  │  ┌──────────┐  ┌──────────────┐  ┌────────────────┐   │  │
│  │  │ Torrent  │  │   Video.js   │  │   Metrics      │   │  │
│  │  │ Input    │  │   Player     │  │   Dashboard    │   │  │
│  │  └──────────┘  └──────────────┘  └────────────────┘   │  │
│  └────────────────────────────────────────────────────────┘  │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         │ HTTP REST + WebSocket
                         │
┌────────────────────────▼─────────────────────────────────────┐
│            Spring Boot Backend (localhost:8080)               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  REST Controllers                                       │  │
│  │  - TorrentController                                    │  │
│  │  - StreamingController (HTTP 206 + Chunked)            │  │
│  │  - MetricsController                                    │  │
│  ├────────────────────────────────────────────────────────┤  │
│  │  Services                                               │  │
│  │  - TorrentService (gRPC client)                         │  │
│  │  - StreamingService (byte-range logic)                  │  │
│  │  - HealthMonitorService (sidecar watchdog)              │  │
│  │  - WebSocketBroadcaster (real-time events)             │  │
│  └────────────────────────────────────────────────────────┘  │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         │ gRPC (localhost:50051)
                         │
┌────────────────────────▼─────────────────────────────────────┐
│               Go Sidecar (localhost:50051)                    │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  gRPC Server                                            │  │
│  │  - AddTorrent, ReadData, PrioritizePieces              │  │
│  ├────────────────────────────────────────────────────────┤  │
│  │  Torrent Engine                                         │  │
│  │  - libtorrent Session                                   │  │
│  │  - Piece Prioritizer (Sequential + On-Demand)          │  │
│  │  - LRU Cache Manager (10GB)                            │  │
│  ├────────────────────────────────────────────────────────┤  │
│  │  Metrics Collector (Prometheus)                         │  │
│  └────────────────────────────────────────────────────────┘  │
└────────────────────────┬─────────────────────────────────────┘
                         │
                         │ BitTorrent Protocol
                         │
┌────────────────────────▼─────────────────────────────────────┐
│                     Peer Network                              │
│  ┌───────┐  ┌───────┐  ┌───────┐  ┌───────┐  ┌─────────┐   │
│  │ Peer1 │  │ Peer2 │  │ Peer3 │  │ PeerN │  │ Tracker │   │
│  └───────┘  └───────┘  └───────┘  └───────┘  └─────────┘   │
└───────────────────────────────────────────────────────────────┘
```

### 1.2 Design Principles

1. **Separation of Concerns**
   - Frontend: User interaction and video rendering
   - Backend: HTTP streaming and orchestration
   - Sidecar: Torrent protocol and piece management

2. **Resilience**
   - Circuit breakers for gRPC calls
   - Retry policies with exponential backoff
   - Graceful degradation for slow torrents

3. **Performance**
   - LRU caching with 85% hit rate target
   - Sequential piece prioritization for streaming
   - 10MB buffer ahead during playback

4. **Debuggability**
   - Structured logging (JSON format)
   - Real-time metrics (Prometheus)
   - Comprehensive health checks

5. **Storage Efficiency**
   - 10GB cache limit with automatic eviction
   - Piece-level granularity
   - Session persistence across restarts

### 1.3 Communication Patterns

| Pattern | Usage | Latency Target |
|---------|-------|----------------|
| **Synchronous gRPC** | Add torrent, get status | < 50ms |
| **Streaming gRPC** | Read data, stream events | N/A (persistent) |
| **HTTP REST** | Frontend API calls | < 100ms |
| **WebSocket** | Real-time updates | < 50ms |
| **HTTP 206** | Video byte-range requests | < 200ms (first byte) |

---

## 2. Component Design

### 2.1 Frontend (React SPA)

#### Component Architecture

```
App
├── TorrentInputPanel
│   ├── MagnetLinkInput (text field + validation)
│   └── TorrentFileUpload (drag-drop)
├── FileSelector
│   └── FileList (multi-file torrent selection)
├── VideoPlayer (Video.js wrapper)
│   ├── PlaybackControls (play, pause, volume)
│   ├── SeekBar (with piece availability overlay)
│   ├── BufferingIndicator (shows download progress)
│   └── SettingsMenu (speed, quality [future])
├── MetricsDashboard
│   ├── DownloadSpeedChart (line graph)
│   ├── PeerCountBadge (live count)
│   ├── ProgressBar (% downloaded)
│   └── CacheUsageBar (X / 10GB)
└── ErrorBoundary (global error handling)
```

#### State Management

```typescript
interface AppState {
  activeTorrent: {
    id: string;
    name: string;
    status: 'downloading_metadata' | 'ready' | 'downloading' | 'error';
    files: FileInfo[];
    selectedFileIndex: number | null;
  } | null;
  
  playbackState: {
    isPlaying: boolean;
    currentTime: number;
    duration: number;
    buffered: TimeRanges;
    seeking: boolean;
  };
  
  metrics: {
    downloadRate: number; // bytes/sec
    uploadRate: number;
    peers: number;
    progress: number; // 0.0 - 1.0
    cacheSize: number; // bytes
    cachHitRatio: number; // 0.0 - 1.0
  };
  
  errors: ErrorMessage[];
}
```

#### Video.js Integration

**Why Video.js?**
1. **Robust Byte-Range Support**: Handles HTTP 206 seamlessly
2. **Plugin Ecosystem**: 100+ plugins for features
3. **Mobile-Friendly**: Touch controls, responsive
4. **Format Agnostic**: MP4, MKV, WebM, AVI support
5. **Event-Rich API**: 50+ events for control

**Configuration:**
```javascript
const videoJsOptions = {
  controls: true,
  autoplay: false,
  preload: 'metadata',
  fluid: true,
  responsive: true,
  
  html5: {
    vhs: {
      overrideNative: true
    }
  },
  
  // Custom plugin for piece-aware buffering
  plugins: {
    pieceBuffering: {
      torrentId: 'abc123',
      fileIndex: 0,
      onSeek: async (targetTime) => {
        const byteOffset = calculateByteOffset(targetTime);
        await api.prioritizeRange(torrentId, fileIndex, byteOffset, 10485760);
      }
    }
  }
}
```

### 2.2 Backend (Spring Boot)

#### Layer Architecture

```
┌─────────────────────────────────────┐
│      REST Controllers               │
│  @RestController                    │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Service Layer                  │
│  @Service                           │
│  - Business logic                   │
│  - gRPC client calls                │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Integration Layer              │
│  - GrpcClientService                │
│  - DockerClientService              │
│  - MetricsCollector                 │
└─────────────────────────────────────┘
```

#### Core Classes

**StreamingController:**
```java
@RestController
@RequestMapping("/api/stream")
@RequiredArgsConstructor
public class StreamingController {
    private final StreamingService streamingService;
    
    @GetMapping("/{torrentId}/{fileIndex}")
    public ResponseEntity<StreamingResponseBody> stream(
        @PathVariable String torrentId,
        @PathVariable int fileIndex,
        @RequestHeader(value = "Range", required = false) String rangeHeader
    ) {
        HttpRange range = parseRangeHeader(rangeHeader);
        
        // Decision: HTTP 206 vs Chunked
        if (streamingService.allPiecesAvailable(torrentId, range)) {
            return streamingService.streamRange(torrentId, fileIndex, range);
        } else {
            return streamingService.streamChunked(torrentId, fileIndex, range.getStart());
        }
    }
}
```

**StreamingService:**
```java
@Service
@RequiredArgsConstructor
public class StreamingService {
    private final GrpcClient grpcClient;
    private final CacheService cacheService;
    
    public ResponseEntity<StreamingResponseBody> streamRange(...) {
        return ResponseEntity.status(HttpStatus.PARTIAL_CONTENT)
            .header("Content-Type", mimeType)
            .header("Content-Range", formatRange(range, totalSize))
            .header("Content-Length", String.valueOf(range.getLength()))
            .header("Accept-Ranges", "bytes")
            .body(outputStream -> {
                // Read data from sidecar via gRPC
                Iterator<ReadDataResponse> responses = 
                    grpcClient.readData(torrentId, fileIndex, range);
                    
                while (responses.hasNext()) {
                    byte[] chunk = responses.next().getData();
                    outputStream.write(chunk);
                    outputStream.flush();
                }
            });
    }
}
```

**HealthMonitorService:**
```java
@Service
public class HealthMonitorService {
    @Scheduled(fixedDelayString = "${grpc.sidecar.health-check.interval}")
    public void checkSidecarHealth() {
        try {
            HealthCheckResponse response = healthStub.check(HealthCheckRequest.newBuilder().build());
            
            if (!"SERVING".equals(response.getStatus().name())) {
                handleUnhealthySidecar();
            }
        } catch (StatusRuntimeException e) {
            log.error("Sidecar health check failed", e);
            attemptRestart();
        }
    }
    
    private void attemptRestart() {
        if (restartAttempts < maxRestartAttempts) {
            dockerClient.restartContainer("torrent-sidecar");
            restartAttempts++;
        }
    }
}
```

### 2.3 Sidecar (Go + libtorrent)

#### Package Structure

```
sidecar/
├── cmd/
│   └── main.go                    # Entry point
├── internal/
│   ├── torrent/
│   │   ├── engine.go              # libtorrent session management
│   │   ├── prioritizer.go         # Piece prioritization
│   │   └── metadata.go            # Video metadata extraction
│   ├── grpc/
│   │   ├── server.go              # gRPC server
│   │   └── handlers.go            # RPC implementations
│   ├── cache/
│   │   ├── lru.go                 # LRU cache
│   │   └── eviction.go            # Eviction policies
│   └── metrics/
│       └── collector.go           # Prometheus metrics
└── proto/
    └── torrent.proto              # gRPC definitions
```

#### Torrent Engine

```go
type TorrentEngine struct {
    session      *libtorrent.Session
    torrents     map[string]*TorrentHandle
    cache        *LRUCache
    prioritizer  *PiecePrioritizer
    mu           sync.RWMutex
}

func NewTorrentEngine(config Config) *TorrentEngine {
    settings := libtorrent.NewSettingsPack()
    settings.SetInt("alert_mask", libtorrent.AlertAllCategories)
    settings.SetBool("enable_dht", true)
    settings.SetInt("connections_limit", 200)
    settings.SetInt("download_rate_limit", config.MaxDownloadRate)
    settings.SetInt("upload_rate_limit", config.MaxUploadRate)
    
    session := libtorrent.NewSession(settings)
    
    return &TorrentEngine{
        session:     session,
        torrents:    make(map[string]*TorrentHandle),
        cache:       NewLRUCache(config.CacheMaxSize),
        prioritizer: NewPiecePrioritizer(),
    }
}

func (e *TorrentEngine) AddTorrent(magnetLink string) (*TorrentHandle, error) {
    params := libtorrent.ParseMagnetUri(magnetLink)
    params.SetFlag(libtorrent.TorrentFlagsSequentialDownload, true)
    
    handle := e.session.AddTorrent(params)
    torrentID := generateID(handle.InfoHash())
    
    e.mu.Lock()
    e.torrents[torrentID] = &TorrentHandle{
        handle:    handle,
        added:     time.Now(),
    }
    e.mu.Unlock()
    
    // Start alert processing
    go e.processAlerts(torrentID)
    
    return e.torrents[torrentID], nil
}

func (e *TorrentEngine) ReadData(torrentID string, fileIndex int, offset, length int64) ([]byte, error) {
    // 1. Calculate required pieces
    pieces := e.calculatePieces(torrentID, fileIndex, offset, length)
    
    // 2. Check cache
    result := make([]byte, 0, length)
    for _, pieceIdx := range pieces {
        cacheKey := fmt.Sprintf("%s:%d", torrentID, pieceIdx)
        
        if data, found := e.cache.Get(cacheKey); found {
            result = append(result, extractRelevantBytes(data, offset, length)...)
            continue
        }
        
        // 3. Wait for piece download
        if err := e.waitForPiece(torrentID, pieceIdx, 30*time.Second); err != nil {
            return nil, err
        }
        
        // 4. Read from libtorrent
        pieceData := e.torrents[torrentID].handle.ReadPiece(pieceIdx)
        
        // 5. Cache the piece
        e.cache.Put(cacheKey, pieceData, CacheMetadata{
            TorrentID:  torrentID,
            PieceIndex: pieceIdx,
        })
        
        result = append(result, extractRelevantBytes(pieceData, offset, length)...)
    }
    
    return result, nil
}
```

#### Piece Prioritizer

```go
type PiecePrioritizer struct {
    bufferAhead int  // Pieces to buffer (default: 40 = ~10MB)
}

func (p *PiecePrioritizer) UpdatePriorities(torrentID string, playbackOffset int64) {
    currentPiece := p.byteToPiece(playbackOffset)
    
    // Priority levels (0-7):
    // 7 = On-demand (user seek)
    // 6 = Current playback position
    // 5 = Buffer ahead (next 40 pieces)
    // 4 = Sequential (next 100 pieces)
    // 1 = Rest of file
    
    for i := 0; i < numPieces; i++ {
        var priority int
        
        if p.hasOnDemandRequest(i) {
            priority = 7
        } else if i == currentPiece {
            priority = 6
        } else if i > currentPiece && i <= currentPiece + p.bufferAhead {
            priority = 5
        } else if i > currentPiece && i <= currentPiece + 100 {
            priority = 4
        } else {
            priority = 1
        }
        
        handle.SetPiecePriority(i, priority)
    }
}
```

#### LRU Cache

```go
type LRUCache struct {
    maxSize     int64
    currentSize int64
    items       map[string]*list.Element
    lruList     *list.List
    mu          sync.RWMutex
}

type cacheEntry struct {
    key          string
    data         []byte
    size         int64
    lastAccessed time.Time
}

func (c *LRUCache) Get(key string) ([]byte, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    elem, found := c.items[key]
    if !found {
        return nil, false
    }
    
    // Move to front (most recently used)
    c.lruList.MoveToFront(elem)
    entry := elem.Value.(*cacheEntry)
    entry.lastAccessed = time.Now()
    
    return entry.data, true
}

func (c *LRUCache) Put(key string, data []byte, metadata CacheMetadata) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Evict if necessary
    for c.currentSize + int64(len(data)) > c.maxSize {
        c.evictOldest()
    }
    
    // Add new entry
    entry := &cacheEntry{
        key:          key,
        data:         data,
        size:         int64(len(data)),
        lastAccessed: time.Now(),
    }
    
    elem := c.lruList.PushFront(entry)
    c.items[key] = elem
    c.currentSize += entry.size
}

func (c *LRUCache) evictOldest() {
    elem := c.lruList.Back()
    if elem == nil {
        return
    }
    
    entry := elem.Value.(*cacheEntry)
    c.lruList.Remove(elem)
    delete(c.items, entry.key)
    c.currentSize -= entry.size
}
```

---

## 3. Data Flow & Protocols

### 3.1 Torrent Addition Flow

```
User → React → Spring Boot → Go Sidecar → libtorrent → Trackers/DHT

Timeline:
0ms:    User pastes magnet link
100ms:  POST /api/torrents
150ms:  gRPC AddTorrent()
200ms:  libtorrent connects to trackers
2-10s:  Metadata received
10.2s:  Parse file list, extract video metadata
10.3s:  Return AddTorrentResponse
10.4s:  React displays file selector
```

**Sequence Diagram:**
```
┌──────┐  ┌───────┐  ┌────────┐  ┌─────────┐  ┌──────────┐
│ User │  │ React │  │ Backend│  │ Sidecar │  │libtorrent│
└──┬───┘  └───┬───┘  └───┬────┘  └────┬────┘  └────┬─────┘
   │          │          │            │            │
   │ Paste    │          │            │            │
   │ magnet   │          │            │            │
   ├─────────>│          │            │            │
   │          │ POST     │            │            │
   │          ├─────────>│            │            │
   │          │          │ gRPC       │            │
   │          │          │ AddTorrent │            │
   │          │          ├───────────>│            │
   │          │          │            │ Add torrent│
   │          │          │            ├───────────>│
   │          │          │            │            │
   │          │          │            │<───────────┤
   │          │          │            │ Metadata   │
   │          │          │<───────────┤            │
   │          │<─────────┤            │            │
   │<─────────┤          │            │            │
   │ Show     │          │            │            │
   │ files    │          │            │            │
```

### 3.2 Streaming Flow (HTTP 206)

```
React → Spring Boot → Go Sidecar → Cache/libtorrent

Request:
GET /api/stream/abc123/0
Range: bytes=0-1048575

Response:
HTTP/1.1 206 Partial Content
Content-Type: video/x-matroska
Content-Range: bytes 0-1048575/1073741824
Content-Length: 1048576

[binary video data]
```

**Decision Tree:**
```
Are required pieces available?
├─ YES: Return HTTP 206 immediately
│   └─ Stream from cache/libtorrent
│
└─ NO: Choose fallback
    ├─ First piece available?
    │   └─ YES: Use chunked transfer
    │       └─ Stream as pieces arrive
    │
    └─ NO: Wait for first piece (10s timeout)
        ├─ Timeout: Return 503 Service Unavailable
        └─ Success: Use chunked transfer
```

### 3.3 Seek Operation Flow

```
User seeks to 5:30 (330 seconds)
│
├─ 1. Video.js fires 'seeking' event
│
├─ 2. Calculate byte offset
│   (330s × bitrate ÷ 8 = ~500MB)
│
├─ 3. POST /api/torrents/{id}/prioritize
│   Body: { byteOffset: 500MB, byteLength: 10MB }
│
├─ 4. gRPC PrioritizePieces()
│   Pieces 1900-1940 priority = 7
│
├─ 5. libtorrent requests pieces from peers
│
├─ 6. Show "Buffering..." indicator
│
├─ 7. Pieces arrive → Cache → WebSocket event
│
├─ 8. React retries HTTP request
│   GET /api/stream/abc123/0
│   Range: bytes=500000000-510485759
│
└─ 9. Video.js resumes playback
```

---

## 4. Streaming Strategy

### 4.1 Dual Protocol Approach

**HTTP 206 Partial Content (Primary)**

*Advantages:*
- Precise control over byte ranges
- Seeking-friendly
- Browser-native support
- Resumable downloads

*Use Cases:*
- Seeking to already-downloaded regions
- Sequential playback after initial buffer
- Downloading subtitle files

**Chunked Transfer Encoding (Fallback)**

*Advantages:*
- Progressive delivery
- No timeout issues
- Handles slow torrents
- No Content-Length required

*Use Cases:*
- Initial playback (first 30s)
- Very slow torrents (< 100 KB/s)
- Unreliable networks

### 4.2 Protocol Selection Logic

```java
public enum StreamingProtocol {
    HTTP_206,
    CHUNKED_TRANSFER
}

public StreamingProtocol selectProtocol(TorrentStatus status, HttpRange range) {
    List<Integer> requiredPieces = calculatePieces(range);
    
    // Rule 1: All pieces available → HTTP 206
    if (allPiecesAvailable(requiredPieces)) {
        return HTTP_206;
    }
    
    // Rule 2: Seeking far ahead → Wait then HTTP 206
    if (range.getStart() > status.getDownloadedBytes() + bufferSize) {
        prioritizePieces(requiredPieces);
        if (waitForPieces(requiredPieces, Duration.ofSeconds(10))) {
            return HTTP_206;
        }
    }
    
    // Rule 3: First piece available → Chunked
    if (hasPiece(requiredPieces.get(0))) {
        return CHUNKED_TRANSFER;
    }
    
    // Rule 4: Wait for first piece, then chunked
    waitForPiece(requiredPieces.get(0), Duration.ofSeconds(10));
    return CHUNKED_TRANSFER;
}
```

### 4.3 Buffering Strategy

**Initial Buffer (Cold Start):**
```
Target: 10 seconds of video

1. Request first 1MB (Range: bytes=0-1048575)
2. Check pieces 0-3 (assuming 256KB pieces)
3. If available: Instant HTTP 206 response
4. If not: Chunked response, playback starts at 256KB
5. Player buffers while downloading ahead
```

**Ongoing Playback:**
```
Sidecar downloads 10MB ahead (40 pieces)
│
├─ Current playback: Piece 100
├─ High priority: Pieces 100-140
├─ Medium priority: Pieces 140-200
└─ Low priority: Pieces 200-end

Cache hit rate: 85%+ (most requests from cache)
```

**Seek Forward (Unwatched):**
```
User seeks from 2:00 to 5:00
│
├─ Calculate target pieces: 1900-1940
├─ Set priority = 7 (max)
├─ Show "Buffering..." with ETA
├─ First piece arrives: Start chunked transfer
├─ After 3-4 pieces: Switch to HTTP 206
└─ Resume smooth playback
```

**Seek Backward (Watched):**
```
User seeks from 5:00 to 2:00
│
├─ Check cache: Likely hit (85% rate)
├─ Immediate HTTP 206 response
└─ Instant playback resumption
```

### 4.4 Piece-to-Byte Mapping

```java
public class PieceCalculator {
    private final int pieceLength;      // 256KB or 512KB
    private final long fileOffset;      // File's start in torrent
    private final long fileSize;
    
    public List<Integer> calculatePieces(long byteOffset, long length) {
        long absoluteStart = fileOffset + byteOffset;
        long absoluteEnd = absoluteStart + length;
        
        int firstPiece = (int) (absoluteStart / pieceLength);
        int lastPiece = (int) (absoluteEnd / pieceLength);
        
        return IntStream.rangeClosed(firstPiece, lastPiece)
            .boxed()
            .collect(Collectors.toList());
    }
}

// Example:
// File offset: 0
// Piece length: 262144 (256KB)
// Byte range: 0-1048575 (1MB)
// Result: Pieces [0, 1, 2, 3]
```

---

## 5. Torrent Management

### 5.1 libtorrent Configuration

```go
settings := libtorrent.NewSettingsPack()

// Connection settings
settings.SetInt("connections_limit", 200)
settings.SetInt("max_peerlist_size", 3000)
settings.SetInt("peer_connect_timeout", 10)

// DHT and peer discovery
settings.SetBool("enable_dht", true)
settings.SetBool("enable_lsd", true)  // Local Service Discovery

// Rate limits
settings.SetInt("download_rate_limit", 0)        // Unlimited
settings.SetInt("upload_rate_limit", 102400)     // 100 KB/s

// Cache settings
settings.SetInt("cache_size", 512)               // 512 MB
settings.SetInt("cache_expiry", 60)

// Alerts (for monitoring)
settings.SetInt("alert_mask", libtorrent.AlertAllCategories)

session := libtorrent.NewSession(settings)
```

### 5.2 Session Persistence

**What Persists:**
```go
type SessionData struct {
    Torrents map[string]*TorrentMetadata `json:"torrents"`
    SavedAt  time.Time                   `json:"savedAt"`
}

type TorrentMetadata struct {
    InfoHash     string    `json:"infoHash"`
    MagnetLink   string    `json:"magnetLink"`
    TotalSize    int64     `json:"totalSize"`
    Downloaded   int64     `json:"downloaded"`
    AddedAt      time.Time `json:"addedAt"`
    LastAccessed time.Time `json:"lastAccessed"`
    
    // Compressed piece bitmap (1 bit per piece)
    PieceBitmap  []byte    `json:"pieceBitmap"`
}
```

**Save/Load Behavior:**
```
On shutdown or every 5 minutes:
├─ Serialize active torrents to JSON
├─ Save to /data/session.json
└─ Persist piece availability bitmap

On startup:
├─ Load session.json
├─ Re-add torrents from magnet links
├─ libtorrent performs fast resume
│   └─ Verifies existing pieces (5-10s)
└─ Continue downloads from last position
```

**Resume After Page Refresh:**
- Frontend state lost (React unmounts)
- Backend and Sidecar continue running
- Frontend reconnects, fetches torrent list
- Playback can resume immediately

### 5.3 Tracker Management

**Whitelisted Trackers:**
```go
var DefaultTrackers = []string{
    "udp://tracker.opentrackr.org:1337/announce",
    "udp://open.stealth.si:80/announce",
    "udp://tracker.torrent.eu.org:451/announce",
    "udp://exodus.desync.com:6969/announce",
}
```

**Validation:**
```go
func validateMagnetLink(magnetLink string) error {
    // 1. Check format
    if !strings.HasPrefix(magnetLink, "magnet:?xt=urn:btih:") {
        return errors.New("invalid magnet link format")
    }
    
    // 2. Extract info hash
    infoHash := extractInfoHash(magnetLink)
    if len(infoHash) != 40 && len(infoHash) != 32 {
        return errors.New("invalid info hash length")
    }
    
    // 3. Check trackers
    trackers := extractTrackers(magnetLink)
    for _, tracker := range trackers {
        if isBlacklisted(tracker) {
            return fmt.Errorf("blacklisted tracker: %s", tracker)
        }
    }
    
    return nil
}
```

### 5.4 Error Handling

**Corrupted Pieces:**
```go
func (h *TorrentHandle) onHashFailed(alert *libtorrent.HashFailedAlert) {
    pieceIndex := alert.PieceIndex()
    log.Warnf("Piece %d failed hash check", pieceIndex)
    
    h.failedPieces++
    
    // Ban peer if too many failures
    if h.peerFailures[alert.PeerIP()] > 3 {
        h.handle.BanPeer(alert.PeerIP())
        log.Infof("Banned peer %s for repeated hash failures", alert.PeerIP())
    }
    
    // Alert if too many total failures
    if h.failedPieces > 10 {
        h.broadcast("error", map[string]interface{}{
            "type": "CORRUPTED_TORRENT",
            "message": "Multiple piece verification failures",
        })
    }
    
    // libtorrent will automatically retry download
}
```

**No Peers:**
```go
func (e *TorrentEngine) monitorPeerCount(torrentID string) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    noPeersCount := 0
    
    for range ticker.C {
        status := e.torrents[torrentID].Status()
        
        if status.NumPeers == 0 {
            noPeersCount++
            
            if noPeersCount == 3 {
                // 30 seconds with no peers
                log.Warn("No peers found, forcing DHT announce")
                e.torrents[torrentID].handle.ForceReannounce()
            }
            
            if noPeersCount == 9 {
                // 90 seconds with no peers
                e.broadcast("error", map[string]interface{}{
                    "type": "NO_PEERS",
                    "message": "Unable to find peers. Check your network or try a different torrent.",
                })
            }
        } else {
            noPeersCount = 0
        }
    }
}
```

---

## 6. Performance & Optimization

### 6.1 Caching Strategy

**Three-Level Cache:**

```
Level 1: Browser (Video.js buffer)
├─ Size: ~20 seconds of video
├─ Purpose: Smooth playback
└─ Eviction: Automatic (player controls)

Level 2: Spring Boot (Metadata)
├─ Size: < 1MB
├─ Purpose: API response speed
└─ Eviction: TTL-based (24 hours)

Level 3: Go Sidecar (LRU)
├─ Size: 10GB
├─ Purpose: Torrent pieces
└─ Eviction: LRU policy

Level 4: Disk (libtorrent)
├─ Size: Limited by volume
├─ Purpose: Persistent storage
└─ Eviction: Manual cleanup
```

**Cache Performance Targets:**

| Scenario | Hit Rate | Latency |
|----------|----------|---------|
| Sequential playback | 95% | < 10ms |
| Random seeks | 60% | < 100ms |
| Multi-file torrent | 40% | < 200ms |
| **Average** | **85%** | **< 50ms** |

### 6.2 Memory Management

**JVM Settings (Spring Boot):**
```bash
-Xms512m                           # Initial heap
-Xmx2g                             # Max heap
-XX:+UseG1GC                       # G1 garbage collector
-XX:MaxGCPauseMillis=200           # GC pause target
-XX:+HeapDumpOnOutOfMemoryError    # Debugging
```

**Go Memory (Sidecar):**
```bash
GOGC=100                           # GC trigger (heap doubles)
# Cache: 10GB
# Piece buffer: 10MB
# Overhead: ~1GB
# Total: ~11GB
```

**Docker Resource Limits:**
```yaml
services:
  sidecar:
    deploy:
      resources:
        limits:
          memory: 12G
          cpus: '4'
        reservations:
          memory: 4G
          cpus: '2'
  
  backend:
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '2'
        reservations:
          memory: 512M
          cpus: '1'
```

### 6.3 Network Optimization

**TCP Tuning (Host Machine):**
```bash
# Increase buffer sizes
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"

# Enable TCP Fast Open
sysctl -w net.ipv4.tcp_fastopen=3
```

**gRPC Configuration:**
```java
@Bean
public ManagedChannel grpcChannel() {
    return ManagedChannelBuilder
        .forAddress(host, port)
        .usePlaintext()
        .maxInboundMessageSize(100 * 1024 * 1024)  // 100MB
        .keepAliveTime(30, TimeUnit.SECONDS)
        .keepAliveTimeout(10, TimeUnit.SECONDS)
        .keepAliveWithoutCalls(true)
        .build();
}
```

### 6.4 Concurrent Download Optimization

**Piece Parallelism:**
```go
// Download 4-6 pieces simultaneously
settings.SetInt("max_suggest_pieces", 6)
settings.SetInt("request_queue_time", 3)

// Optimize for streaming
settings.SetBool("strict_super_seeding", false)
settings.SetBool("rate_limit_ip_overhead", false)
```

**Connection Pooling:**
```yaml
server:
  tomcat:
    threads:
      max: 200
      min-spare: 10
    max-connections: 10000
    accept-count: 100
```

---

## 7. Error Handling

### 7.1 Error Categories

| Category | Examples | Recovery |
|----------|----------|----------|
| **Transient** | Network timeout, peer disconnect | Retry with backoff |
| **Permanent** | Invalid magnet, corrupted file | Fail fast, notify user |
| **Resource** | Disk full, OOM | Graceful degradation |
| **External** | Tracker down, no peers | Fallback to DHT |

### 7.2 Retry Policies

**gRPC Retries:**
```yaml
resilience4j:
  retry:
    instances:
      grpcSidecar:
        max-attempts: 3
        wait-duration: 500ms
        enable-exponential-backoff: true
        exponential-backoff-multiplier: 2
```

**Circuit Breaker:**
```yaml
resilience4j:
  circuitbreaker:
    instances:
      grpcSidecar:
        sliding-window-size: 10
        failure-rate-threshold: 50
        wait-duration-in-open-state: 10s
```

### 7.3 Timeout Configuration

```yaml
Timeouts:
  Add Torrent (metadata):  60s
  Read Data (piece):       30s
  HTTP Stream (first byte): 10s
  gRPC Health Check:        5s
  Prioritize Pieces:        5s
```

### 7.4 Graceful Degradation

**Slow Torrent Matrix:**

| Download Rate | Action |
|---------------|--------|
| > 500 KB/s | Normal playback |
| 100-500 KB/s | Occasional buffering |
| 50-100 KB/s | Frequent buffering, show warning |
| < 50 KB/s | "Slow torrent" warning + ETA |
| < 10 KB/s (60s) | Suggest alternative torrent |

**Example UI Message:**
```
⚠️ Slow Torrent Detected

Current speed: 45 KB/s
Estimated time to buffer: 3 minutes

Suggestions:
• Try a different torrent with more seeders
• Check your network connection
• Consider using a VPN if region-blocked
```

---

## 8. Security

### 8.1 Input Validation

**Magnet Link Validation:**
```java
@Component
public class MagnetLinkValidator {
    private static final Pattern MAGNET_PATTERN = 
        Pattern.compile("^magnet:\\?xt=urn:btih:([a-fA-F0-9]{40}|[a-zA-Z2-7]{32}).*$");
    
    public void validate(String magnetLink) {
        // 1. Format check
        if (!MAGNET_PATTERN.matcher(magnetLink).matches()) {
            throw new ValidationException("Invalid magnet link format");
        }
        
        // 2. Length check (prevent DoS)
        if (magnetLink.length() > 10000) {
            throw new ValidationException("Magnet link too long");
        }
        
        // 3. Tracker validation
        List<String> trackers = extractTrackers(magnetLink);
        for (String tracker : trackers) {
            if (isBlacklisted(tracker)) {
                throw new ValidationException("Blacklisted tracker: " + tracker);
            }
        }
    }
}
```

**Path Traversal Prevention:**
```go
func sanitizePath(torrentID, filePath string) string {
    // Clean path
    filePath = filepath.Clean(filePath)
    
    // Remove leading slashes
    filePath = strings.TrimPrefix(filePath, "/")
    
    // Construct safe path
    safeDir := filepath.Join(downloadDir, torrentID)
    safePath := filepath.Join(safeDir, filePath)
    
    // Verify within jail
    if !strings.HasPrefix(safePath, safeDir) {
        panic("Path traversal attempt")
    }
    
    return safePath
}
```

### 8.2 Rate Limiting

```java
@Configuration
public class RateLimitConfig {
    @Bean
    public RateLimiter addTorrentLimiter() {
        return RateLimiter.create(10.0 / 60.0);  // 10 per minute
    }
    
    @Bean
    public RateLimiter streamLimiter() {
        return RateLimiter.create(100.0);  // 100 per second
    }
}

@RestController
public class TorrentController {
    @PostMapping("/torrents")
    public ResponseEntity<?> addTorrent(...) {
        if (!rateLimiter.tryAcquire()) {
            return ResponseEntity.status(429).body("Rate limit exceeded");
        }
        // ...
    }
}
```

### 8.3 CORS Configuration

```yaml
cors:
  allowed-origins:
    - http://localhost:3000
    - http://192.168.1.100:3000  # LAN access
  allowed-methods: GET,POST,PUT,DELETE,OPTIONS
  allowed-headers: "*"
  allow-credentials: true
  max-age: 3600
```

### 8.4 Network Isolation

**Docker Network:**
```yaml
networks:
  streaming-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

**Firewall Rules (Recommended):**
```bash
# Allow frontend
iptables -A INPUT -p tcp --dport 3000 -j ACCEPT

# Allow backend
iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# Block external gRPC access
iptables -A INPUT -p tcp --dport 50051 -j DROP

# Allow BitTorrent DHT
iptables -A INPUT -p udp --dport 6881:6889 -j ACCEPT
```

---

## 9. Monitoring

### 9.1 Prometheus Metrics

**Backend Metrics:**
```java
// Counters
torrent_added_total
stream_requests_total
cache_hits_total
cache_misses_total

// Gauges
active_torrents
active_streams
cache_size_bytes

// Histograms
stream_latency_seconds
grpc_call_duration_seconds
```

**Sidecar Metrics:**
```go
// Gauges
torrent_peers_connected{torrent_id}
torrent_download_rate_bytes_per_sec{torrent_id}
torrent_upload_rate_bytes_per_sec{torrent_id}
cache_size_bytes

// Counters
torrent_pieces_downloaded_total{torrent_id}
torrent_pieces_failed_total{torrent_id}
cache_evictions_total
```

### 9.2 Logging Strategy

**Structured Logging (JSON):**
```json
{
  "timestamp": "2025-01-27T10:30:45.123Z",
  "level": "INFO",
  "service": "backend",
  "traceId": "abc123",
  "torrentId": "def456",
  "action": "stream_request",
  "details": {
    "fileIndex": 0,
    "range": "bytes=0-1048575",
    "protocol": "http_206",
    "cacheHit": true
  }
}
```

**Log Levels:**
```
TRACE: gRPC payloads, piece-level details
DEBUG: Cache operations, piece downloads
INFO:  User actions, torrent lifecycle
WARN:  Slow torrents, retries, degraded state
ERROR: Failures, exceptions, data corruption
```

### 9.3 Grafana Dashboards

**Dashboard 1: System Overview**
- Active Torrents (gauge)
- Total Download/Upload Rate (graph)
- Cache Hit Ratio (pie chart)
- Error Rate (graph)

**Dashboard 2: Torrent Details**
- Per-Torrent Download Speed (table)
- Peer Count (bar chart)
- Piece Progress (heatmap)

**Dashboard 3: Performance**
- Stream Latency p50/p95/p99 (histogram)
- gRPC Call Duration (graph)
- JVM Memory Usage (graph)

### 9.4 Health Checks

**Endpoints:**
```
GET /actuator/health           # Overall health
GET /actuator/health/liveness  # Kubernetes probe
GET /actuator/health/readiness # Kubernetes probe
GET /actuator/prometheus       # Metrics scrape
```

**Custom Health Indicator:**
```java
@Component
public class SidecarHealthIndicator implements HealthIndicator {
    @Override
    public Health health() {
        try {
            HealthCheckResponse response = healthStub.check(...);
            
            return Health.up()
                .withDetail("sidecar", "connected")
                .withDetail("torrents", activeTorrents.size())
                .build();
        } catch (Exception e) {
            return Health.down()
                .withDetail("sidecar", "disconnected")
                .withDetail("error", e.getMessage())
                .build();
        }
    }
}
```

---

## 10. Deployment

### 10.1 Docker Compose

```yaml
version: '3.8'

services:
  frontend:
    build: ./frontend
    ports:
      - "3000:80"
    environment:
      - REACT_APP_API_URL=http://localhost:8080
    depends_on:
      - backend
  
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      - GRPC_SIDECAR_HOST=sidecar
      - GRPC_SIDECAR_PORT=50051
    depends_on:
      sidecar:
        condition: service_healthy
  
  sidecar:
    build: ./sidecar
    ports:
      - "50051:50051"
    volumes:
      - torrent-data:/data/torrents
      - torrent-cache:/data/cache
    environment:
      - CACHE_MAX_SIZE=10737418240
    healthcheck:
      test: ["CMD", "/app/healthcheck"]
      interval: 30s

volumes:
  torrent-data:
  torrent-cache:
```

### 10.2 Resource Requirements

**Minimum:**
- CPU: 4 cores
- RAM: 16GB
- Disk: 50GB (cache + downloads)
- Network: 10 Mbps download

**Recommended:**
- CPU: 8 cores
- RAM: 32GB
- Disk: 500GB SSD
- Network: 100 Mbps download

---

## 11. API Specifications

### 11.1 REST Endpoints

**Add Torrent:**
```http
POST /api/torrents
Content-Type: application/json

{
  "magnetLink": "magnet:?xt=urn:btih:...",
  "trackerWhitelist": ["udp://tracker.opentrackr.org:1337"]
}

Response: 200 OK
{
  "torrentId": "abc123",
  "infoHash": "dd8255ec...",
  "status": "downloading_metadata"
}
```

**Stream Video:**
```http
GET /api/stream/{torrentId}/{fileIndex}
Range: bytes=0-1048575

Response: 206 Partial Content
Content-Type: video/x-matroska
Content-Range: bytes 0-1048575/1073741824
[binary data]
```

**Get Status:**
```http
GET /api/torrents/{torrentId}/status

Response: 200 OK
{
  "state": "downloading",
  "progress": 0.50,
  "downloadRate": 1048576,
  "numPeers": 15,
  "etaSeconds": 512
}
```

### 11.2 WebSocket Events

```javascript
// Connection
const ws = new WebSocket('ws://localhost:8080/ws');

// Events
{
  "type": "piece_downloaded",
  "torrentId": "abc123",
  "pieceIndex": 42,
  "downloadSpeed": 1048576
}

{
  "type": "status_update",
  "torrentId": "abc123",
  "progress": 0.52,
  "numPeers": 15
}
```

---

## 12. Testing Strategy

### 12.1 Unit Tests

**Backend (JUnit):**
```java
@Test
void testHttp206ForAvailablePieces() {
    when(stub.getTorrentStatus(any()))
        .thenReturn(statusWithPieces());
    
    ResponseEntity<?> response = streamingService.stream(...);
    
    assertEquals(HttpStatus.PARTIAL_CONTENT, response.getStatusCode());
}
```

**Sidecar (Go):**
```go
func TestLRUCacheEviction(t *testing.T) {
    cache := NewLRUCache(1024)
    
    cache.Put("key1", make([]byte, 512), meta1)
    cache.Put("key2", make([]byte, 512), meta2)
    cache.Put("key3", make([]byte, 512), meta3)
    
    _, found := cache.Get("key1")
    assert.False(t, found)  // Evicted
}
```

### 12.2 Integration Tests

```java
@SpringBootTest
@Testcontainers
class StreamingIntegrationTest {
    @Container
    static GenericContainer<?> sidecar = new GenericContainer<>("sidecar:latest");
    
    @Test
    void testEndToEndStreaming() {
        String torrentId = addTestTorrent();
        waitForMetadata(torrentId, Duration.ofSeconds(15));
        
        byte[] chunk = requestRange(torrentId, 0, 0, 1048575);
        assertEquals(1048576, chunk.length);
    }
}
```

---

## 13. Future Enhancements

### Short-term (1-3 months)
1. **Subtitle Support**: Auto-detect and render .srt/.vtt files
2. **Quality Selection**: Multiple video files (720p, 1080p, 4K)
3. **Playlist Mode**: Queue multiple episodes
4. **Advanced Analytics**: Peer maps, bandwidth graphs

### Medium-term (3-6 months)
1. **Mobile Apps**: React Native for iOS/Android
2. **Chromecast Support**: Cast to TV
3. **Content Discovery**: Torrent search integration
4. **Transcoding**: FFmpeg for unsupported codecs

### Long-term (6-12 months)
1. **Multi-User Support**: Accounts and authentication
2. **Kubernetes Deployment**: Horizontal scaling
3. **VPN Integration**: Built-in privacy
4. **Plugin System**: Extensible architecture

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **Piece** | Fixed-size chunk of torrent (256KB-512KB) |
| **Info Hash** | SHA-1 hash identifying torrent |
| **Magnet Link** | URI with info hash and trackers |
| **Tracker** | Server coordinating peers |
| **DHT** | Distributed Hash Table for peer discovery |
| **LRU** | Least Recently Used eviction policy |
| **gRPC** | Google Remote Procedure Call |
| **HTTP 206** | Partial Content (byte-range) |

---

## Appendix B: Troubleshooting

**Problem: No peers found**
- Wait 5 minutes for DHT
- Check tracker whitelist
- Verify UDP ports 6881-6889 open

**Problem: Slow playback**
- Increase buffer: `streaming.buffer-size=20971520`
- Check disk space for cache
- Monitor metrics: GET `/api/metrics`

**Problem: Sidecar won't start**
- Check logs: `docker-compose logs sidecar`
- Verify port 50051 available
- Ensure libtorrent installed

---

## Conclusion

This design document provides a complete blueprint for a production-ready MagnetPlay. The architecture prioritizes:

1. **Immediate Playback**: < 10s to first frame
2. **Resilience**: Dual protocol, circuit breakers
3. **Performance**: 85% cache hit rate, 10MB buffer
4. **Debuggability**: Structured logs, real-time metrics
5. **Storage Efficiency**: LRU cache with 10GB limit

**Next Steps:**
1. Review design with stakeholders
2. Set up development environment
3. Implement components (sidecar → backend → frontend)
4. Write tests alongside code
5. Deploy to staging and iterate

---

**Document Version:** 1.0  
**Last Updated:** January 27, 2026  
**Status:** Final Design  
**Approved By:** Architecture Team
