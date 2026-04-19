# Torrent Dashboard & Player Status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a player-bar status widget showing per-file download stats, plus a torrent management page with live/paused/inactive torrent list, pause/delete actions, and a session tracking table for active and closed streaming connections.

**Architecture:** New gRPC RPCs in the proto contract expose torrent stats and management operations; the Go sidecar implements them using anacrolix/torrent APIs; Spring Boot adds REST endpoints that delegate to gRPC; the React frontend adds a StatusPanel component on the player bar and a new TorrentsPage route with session tables.

**Tech Stack:** Go 1.21 + anacrolix/torrent, protobuf/grpc, Java 17 + Spring WebFlux + Spring gRPC, React 18 + TypeScript + Vite, Video.js 8

---

## Graph-Informed Risk Map

> Derived from `graphify-out/GRAPH_REPORT.md` (2026-04-19, 143 nodes, 155 edges).

| File | Graph Signal | Risk |
|------|-------------|------|
| `repository.go` | **#1 god node** — 11 edges, bridges Go Streaming + Go Torrent Service + Go Server Startup communities | 🔴 HIGH — mutex errors here break all torrent ops |
| `App.tsx` | **#2 god node** — 11 edges, center of React Frontend Components community | 🟡 MEDIUM — Tasks 8+9 both modify it; do in one pass |
| `service.go` | Part of "End-to-End gRPC Video Streaming Pipeline" hyperedge (confidence 0.90) | 🟡 MEDIUM — new handlers must not break StreamFile |
| `TorrentController.java` | Community 0 cohesion = 0.13 (weakest cluster); already loosely coupled | 🟡 MEDIUM — adding 6 endpoints worsens cohesion; future split to `TorrentManagementController` post-MVP |
| `TorrentInfo` struct | **Isolated node** (≤1 connection) — no ripple risk | 🟢 LOW — safe to add `Paused bool` |
| `NewTorrentService()` | Exists as graph node in Community 4 — do NOT recreate | 🟢 LOW — update struct fields only, skip constructor creation |

---

## File Map

### New files
| File | Responsibility |
|------|---------------|
| `backend/go-server/internal/torrent/speed_tracker.go` | Per-file rolling-window speed calculator |
| `backend/mp-spring/.../model/TorrentStatsResponse.java` | DTO for stats endpoint |
| `backend/mp-spring/.../model/TorrentListResponse.java` | DTO for list endpoint |
| `backend/mp-spring/.../model/SessionResponse.java` | DTO for sessions endpoint |
| `backend/mp-spring/.../session/StreamingSession.java` | Session record (data class) |
| `backend/mp-spring/.../session/SessionManager.java` | In-memory session store + lifecycle hooks |
| `frontend/src/components/StatusPanel.tsx` | Player-bar download stats panel |
| `frontend/src/components/TorrentsPage.tsx` | Management page: torrent table + sessions |

### Modified files
| File | What changes | Graph risk |
|------|-------------|-----------|
| `backend/proto/torrent.proto` | +5 RPCs, +8 messages, +1 enum | 🟢 |
| `backend/go-server/internal/torrent/repository.go` | Add `Paused bool` to TorrentInfo; add `ListTorrents()`, `SetPaused()`, `Remove()` | 🔴 god node — touch carefully |
| `backend/go-server/internal/torrent/service.go` | Implement GetTorrentStats, ListTorrents, PauseTorrent, ResumeTorrent, DeleteTorrent | 🟡 part of streaming pipeline hyperedge |
| `backend/mp-spring/.../controller/TorrentController.java` | Add stats, list, pause, resume, delete, sessions endpoints | 🟡 cohesion already 0.13 |
| `backend/mp-spring/.../service/TorrentService.java` | Add gRPC delegators + inject SessionManager | 🟡 |
| `backend/mp-spring/.../client/TorrentGrpcClient.java` | Stub methods for 5 new RPCs | 🟢 |
| `frontend/src/App.tsx` | Wire StatusPanel + TorrentsPage **in a single commit** | 🟡 god node — do Tasks 8+9 in one pass |
| `frontend/src/types/index.ts` | Add TorrentStats, TorrentListItem, StreamingSession types | 🟢 |

---

## Phase 1 — Proto Contract

### Task 1: Extend proto with stats + management + session RPCs

**Files:**
- Modify: `backend/proto/torrent.proto`

- [ ] **Step 1: Add new enum and messages**

Open `backend/proto/torrent.proto`. After the existing `FileChunk` message, add:

```proto
// ── Torrent state ─────────────────────────────────────────────
enum TorrentState {
  TORRENT_ACTIVE  = 0;
  TORRENT_PAUSED  = 1;
  TORRENT_STOPPED = 2;
}

// ── Stats (F1) ────────────────────────────────────────────────
message TorrentFileStats {
  string file_id            = 1;
  int64  total_size         = 2;
  int64  downloaded_bytes   = 3;
  double download_speed_bps = 4;
  double completion_pct     = 5;
}

message GetTorrentStatsRequest {
  string info_hash = 1;
  string file_id   = 2;
}

message GetTorrentStatsResponse {
  TorrentFileStats stats = 1;
}

// ── Management (F2i) ─────────────────────────────────────────
message TorrentListItem {
  string        torrent_id         = 1;
  string        name               = 2;
  TorrentStatus status             = 3;
  TorrentState  state              = 4;
  int64         total_size         = 5;
  int64         downloaded_bytes   = 6;
  double        completion_pct     = 7;
  double        download_speed_bps = 8;
  repeated FileInfo files          = 9;
}

message ListTorrentsRequest  {}
message ListTorrentsResponse { repeated TorrentListItem torrents = 1; }

message PauseTorrentRequest  { string info_hash = 1; }
message PauseTorrentResponse { bool success = 1; string message = 2; }

message ResumeTorrentRequest  { string info_hash = 1; }
message ResumeTorrentResponse { bool success = 1; string message = 2; }

message DeleteTorrentRequest  { string info_hash = 1; bool delete_files = 2; }
message DeleteTorrentResponse { bool success = 1; string message = 2; }
```

- [ ] **Step 2: Add RPCs to the service block**

In `TorrentService`, after `rpc StreamFile`, add:

```proto
  rpc GetTorrentStats  (GetTorrentStatsRequest)  returns (GetTorrentStatsResponse);
  rpc ListTorrents     (ListTorrentsRequest)      returns (ListTorrentsResponse);
  rpc PauseTorrent     (PauseTorrentRequest)      returns (PauseTorrentResponse);
  rpc ResumeTorrent    (ResumeTorrentRequest)     returns (ResumeTorrentResponse);
  rpc DeleteTorrent    (DeleteTorrentRequest)     returns (DeleteTorrentResponse);
```

- [ ] **Step 3: Regenerate Go proto**

```bash
cd backend
make -f MakeFile proto
```

Expected output: regenerated `go-server/torrent/torrent.pb.go` and `torrent_grpc.pb.go` with no errors.

- [ ] **Step 4: Verify Java proto compiles**

```bash
cd backend/mp-spring
./mvnw compile -q
```

Expected: BUILD SUCCESS. The `protobuf-maven-plugin` regenerates Java stubs automatically.

- [ ] **Step 5: Commit**

```bash
git add backend/proto/torrent.proto backend/go-server/torrent/torrent.pb.go backend/go-server/torrent/torrent_grpc.pb.go
git commit -m "Feature(Proto): add stats, management, and delete RPCs to TorrentService"
```

---

## Phase 2 — Go Sidecar: Speed Tracker

### Task 2: Implement rolling-window speed tracker

**Files:**
- Create: `backend/go-server/internal/torrent/speed_tracker.go`

- [ ] **Step 1: Write the failing test**

Create `backend/go-server/internal/torrent/speed_tracker_test.go`:

```go
package torrent

import (
	"testing"
	"time"
)

func TestSpeedTracker_ZeroAtStart(t *testing.T) {
	tr := NewSpeedTracker(5)
	if got := tr.SpeedBps(); got != 0 {
		t.Errorf("want 0 at start, got %f", got)
	}
}

func TestSpeedTracker_SingleSample(t *testing.T) {
	tr := NewSpeedTracker(5)
	tr.Record(1000, time.Now().Add(-1*time.Second))
	tr.Record(2000, time.Now())
	speed := tr.SpeedBps()
	if speed < 900 || speed > 1100 {
		t.Errorf("want ~1000 bps, got %f", speed)
	}
}

func TestSpeedTracker_OldSamplesDropped(t *testing.T) {
	tr := NewSpeedTracker(2) // 2-second window
	tr.Record(0, time.Now().Add(-10*time.Second))
	tr.Record(500, time.Now().Add(-5*time.Second)) // outside window
	tr.Record(1000, time.Now().Add(-1*time.Second))
	tr.Record(2000, time.Now())
	speed := tr.SpeedBps()
	if speed < 900 || speed > 1100 {
		t.Errorf("want ~1000 bps (only last 2s window), got %f", speed)
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd backend/go-server && go test ./internal/torrent/... -run TestSpeedTracker -v
```

Expected: `cannot find package` or `undefined: NewSpeedTracker`

- [ ] **Step 3: Implement SpeedTracker**

Create `backend/go-server/internal/torrent/speed_tracker.go`:

```go
package torrent

import (
	"sync"
	"time"
)

type sample struct {
	bytes int64
	at    time.Time
}

type SpeedTracker struct {
	mu      sync.Mutex
	window  time.Duration
	samples []sample
}

func NewSpeedTracker(windowSeconds int) *SpeedTracker {
	return &SpeedTracker{window: time.Duration(windowSeconds) * time.Second}
}

// Record adds a cumulative byte count observation.
func (s *SpeedTracker) Record(totalBytes int64, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := at.Add(-s.window)
	kept := s.samples[:0]
	for _, p := range s.samples {
		if p.at.After(cutoff) {
			kept = append(kept, p)
		}
	}
	s.samples = append(kept, sample{bytes: totalBytes, at: at})
}

// SpeedBps returns bytes/second over the window. 0 if fewer than 2 samples.
func (s *SpeedTracker) SpeedBps() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.samples) < 2 {
		return 0
	}
	oldest := s.samples[0]
	newest := s.samples[len(s.samples)-1]
	elapsed := newest.at.Sub(oldest.at).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(newest.bytes-oldest.bytes) / elapsed
}
```

- [ ] **Step 4: Run tests to confirm pass**

```bash
cd backend/go-server && go test ./internal/torrent/... -run TestSpeedTracker -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/go-server/internal/torrent/speed_tracker.go backend/go-server/internal/torrent/speed_tracker_test.go
git commit -m "Feature(Go): add rolling-window SpeedTracker for per-file download speed"
```

---

## Phase 3 — Go Sidecar: Repository Pause Support

### Task 3: Add pause state to TorrentInfo

> ⚠️ **Graph warning:** `Repository` is the #1 god node (11 edges), bridging Go Streaming Concepts, Go Torrent Service, and Go Server Startup communities. Every method added here must acquire `r.mu` correctly — a missed lock propagates to all torrent operations.

**Files:**
- Modify: `backend/go-server/internal/torrent/repository.go`

- [ ] **Step 1: Read current repository.go**

Read `backend/go-server/internal/torrent/repository.go` fully before editing.

- [ ] **Step 2: Write failing test**

Create `backend/go-server/internal/torrent/repository_test.go` (or add to existing):

```go
package torrent

import (
	"testing"
)

func TestTorrentInfo_PauseResume(t *testing.T) {
	info := &TorrentInfo{Paused: false}
	if info.Paused {
		t.Error("want not paused at start")
	}
	info.Paused = true
	if !info.Paused {
		t.Error("want paused after set")
	}
}
```

- [ ] **Step 3: Run to confirm structure exists or fails**

```bash
cd backend/go-server && go test ./internal/torrent/... -run TestTorrentInfo_PauseResume -v
```

Expected: FAIL if `TorrentInfo` has no `Paused` field.

- [ ] **Step 4: Add Paused field to TorrentInfo**

In `repository.go`, find the `TorrentInfo` struct and add `Paused bool`:

```go
type TorrentInfo struct {
    Torrent *torrent.Torrent
    Files   map[string]*torrent.File
    Paused  bool  // true when all file priorities set to None
}
```

- [ ] **Step 5: Run test again**

```bash
cd backend/go-server && go test ./internal/torrent/... -run TestTorrentInfo_PauseResume -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/go-server/internal/torrent/repository.go backend/go-server/internal/torrent/repository_test.go
git commit -m "Feature(Go): add Paused state field to TorrentInfo"
```

---

## Phase 4 — Go Sidecar: New gRPC Handlers

### Task 4: Implement GetTorrentStats, ListTorrents, PauseTorrent, ResumeTorrent, DeleteTorrent

**Files:**
- Modify: `backend/go-server/internal/torrent/service.go`

- [ ] **Step 1: Read current service.go**

Read `backend/go-server/internal/torrent/service.go` fully.

- [ ] **Step 2: Write integration test stubs**

Create `backend/go-server/internal/torrent/service_stats_test.go`:

```go
package torrent

import (
	"context"
	"testing"

	pb "MagnetPlay/backend/torrent"
)

func TestGetTorrentStats_UnknownHash(t *testing.T) {
	svc := &TorrentService{repo: NewRepository()}
	_, err := svc.GetTorrentStats(context.Background(), &pb.GetTorrentStatsRequest{
		InfoHash: "nonexistent",
		FileId:   "nonexistent:0",
	})
	if err == nil {
		t.Error("want error for unknown torrent, got nil")
	}
}

func TestListTorrents_EmptyRepo(t *testing.T) {
	svc := &TorrentService{repo: NewRepository()}
	resp, err := svc.ListTorrents(context.Background(), &pb.ListTorrentsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Torrents) != 0 {
		t.Errorf("want 0 torrents, got %d", len(resp.Torrents))
	}
}
```

- [ ] **Step 3: Run to confirm failure**

```bash
cd backend/go-server && go test ./internal/torrent/... -run "TestGetTorrentStats|TestListTorrents" -v
```

Expected: compilation error — methods not implemented yet.

- [ ] **Step 4: Add SpeedTracker map to TorrentService**

> ℹ️ **Graph note:** `NewTorrentService()` already exists as a graph node in Community 4 ("Go Torrent Service"). Do NOT recreate it — only add the `speedTrackers` field and update the existing constructor body.

In `service.go`, update the existing `TorrentService` struct to add the new fields:

```go
type TorrentService struct {
    pb.UnimplementedTorrentServiceServer
    repo          *Repository
    speedTrackers map[string]*SpeedTracker // key: info_hash:file_id
    trackerMu     sync.Mutex
}
```

Update the existing `NewTorrentService` constructor — add `speedTrackers` initialization to the return statement:
```go
func NewTorrentService(repo *Repository) *TorrentService {
    return &TorrentService{
        repo:          repo,
        speedTrackers: make(map[string]*SpeedTracker),
    }
}
```

- [ ] **Step 5: Implement GetTorrentStats**

Add to `service.go`:

```go
func (s *TorrentService) GetTorrentStats(ctx context.Context, req *pb.GetTorrentStatsRequest) (*pb.GetTorrentStatsResponse, error) {
    info, err := s.repo.GetTorrent(req.InfoHash)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.InfoHash)
    }
    file, err := s.repo.GetFile(req.InfoHash, req.FileId)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "file not found: %s", req.FileId)
    }
    _ = info

    downloaded := file.BytesCompleted()
    total := file.Length()
    var pct float64
    if total > 0 {
        pct = float64(downloaded) / float64(total) * 100
    }

    trackerKey := req.InfoHash + ":" + req.FileId
    s.trackerMu.Lock()
    tr, ok := s.speedTrackers[trackerKey]
    if !ok {
        tr = NewSpeedTracker(5)
        s.speedTrackers[trackerKey] = tr
    }
    s.trackerMu.Unlock()
    tr.Record(downloaded, time.Now())

    return &pb.GetTorrentStatsResponse{
        Stats: &pb.TorrentFileStats{
            FileId:           req.FileId,
            TotalSize:        total,
            DownloadedBytes:  downloaded,
            DownloadSpeedBps: tr.SpeedBps(),
            CompletionPct:    pct,
        },
    }, nil
}
```

- [ ] **Step 6: Implement ListTorrents**

```go
func (s *TorrentService) ListTorrents(ctx context.Context, req *pb.ListTorrentsRequest) (*pb.ListTorrentsResponse, error) {
    all := s.repo.ListTorrents()
    items := make([]*pb.TorrentListItem, 0, len(all))
    for _, info := range all {
        t := info.Torrent
        var totalSize, downloaded int64
        var files []*pb.FileInfo
        for id, f := range info.Files {
            totalSize += f.Length()
            downloaded += f.BytesCompleted()
            files = append(files, &pb.FileInfo{Id: id, Name: f.Path(), Size: f.Length()})
        }
        var pct float64
        if totalSize > 0 {
            pct = float64(downloaded) / float64(totalSize) * 100
        }
        state := pb.TorrentState_TORRENT_ACTIVE
        if info.Paused {
            state = pb.TorrentState_TORRENT_PAUSED
        }
        items = append(items, &pb.TorrentListItem{
            TorrentId:        t.InfoHash().HexString(),
            Name:             t.Name(),
            State:            state,
            TotalSize:        totalSize,
            DownloadedBytes:  downloaded,
            CompletionPct:    pct,
            DownloadSpeedBps: 0, // aggregate speed omitted for list; client calls GetTorrentStats per file
            Files:            files,
        })
    }
    return &pb.ListTorrentsResponse{Torrents: items}, nil
}
```

- [ ] **Step 7: Add ListTorrents to Repository**

In `repository.go`, add:

```go
func (r *Repository) ListTorrents() []*TorrentInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    result := make([]*TorrentInfo, 0, len(r.torrents))
    for _, info := range r.torrents {
        result = append(result, info)
    }
    return result
}
```

- [ ] **Step 8: Implement PauseTorrent**

```go
func (s *TorrentService) PauseTorrent(ctx context.Context, req *pb.PauseTorrentRequest) (*pb.PauseTorrentResponse, error) {
    info, err := s.repo.GetTorrent(req.InfoHash)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.InfoHash)
    }
    if info.Paused {
        return &pb.PauseTorrentResponse{Success: true, Message: "already paused"}, nil
    }
    for _, f := range info.Files {
        f.SetPriority(torrent.PiecePriorityNone)
    }
    s.repo.SetPaused(req.InfoHash, true)
    return &pb.PauseTorrentResponse{Success: true, Message: "paused"}, nil
}
```

- [ ] **Step 9: Implement ResumeTorrent**

```go
func (s *TorrentService) ResumeTorrent(ctx context.Context, req *pb.ResumeTorrentRequest) (*pb.ResumeTorrentResponse, error) {
    info, err := s.repo.GetTorrent(req.InfoHash)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.InfoHash)
    }
    if !info.Paused {
        return &pb.ResumeTorrentResponse{Success: true, Message: "already active"}, nil
    }
    for _, f := range info.Files {
        f.Download()
    }
    s.repo.SetPaused(req.InfoHash, false)
    return &pb.ResumeTorrentResponse{Success: true, Message: "resumed"}, nil
}
```

- [ ] **Step 10: Implement DeleteTorrent**

```go
func (s *TorrentService) DeleteTorrent(ctx context.Context, req *pb.DeleteTorrentRequest) (*pb.DeleteTorrentResponse, error) {
    info, err := s.repo.GetTorrent(req.InfoHash)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.InfoHash)
    }
    info.Torrent.Drop()
    s.repo.Remove(req.InfoHash)
    if req.DeleteFiles {
        dataDir := filepath.Join(".", "downloads", req.InfoHash)
        if err := os.RemoveAll(dataDir); err != nil {
            return &pb.DeleteTorrentResponse{Success: false, Message: err.Error()}, nil
        }
    }
    return &pb.DeleteTorrentResponse{Success: true, Message: "deleted"}, nil
}
```

- [ ] **Step 11: Add SetPaused + Remove to Repository**

In `repository.go`:

```go
func (r *Repository) SetPaused(infoHash string, paused bool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    if info, ok := r.torrents[infoHash]; ok {
        info.Paused = paused
    }
}

func (r *Repository) Remove(infoHash string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    delete(r.torrents, infoHash)
}
```

- [ ] **Step 12: Run tests**

```bash
cd backend/go-server && go test ./internal/torrent/... -v
```

Expected: all tests pass, no compilation errors.

- [ ] **Step 13: Build Go binary to verify**

```bash
cd backend/go-server && go build ./...
```

Expected: no errors.

- [ ] **Step 14: Commit**

```bash
git add backend/go-server/internal/torrent/service.go backend/go-server/internal/torrent/repository.go
git commit -m "Feature(Go): implement GetTorrentStats, ListTorrents, PauseTorrent, ResumeTorrent, DeleteTorrent"
```

---

## Phase 5 — Spring Boot: Stats + Management Endpoints

### Task 5: Add DTOs, gRPC client methods, service methods, and REST endpoints

**Files:**
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/TorrentStatsResponse.java`
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/TorrentListResponse.java`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/TorrentGrpcClient.java`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/TorrentController.java`

- [ ] **Step 1: Read all three existing Java files before editing**

Read:
- `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/TorrentGrpcClient.java`
- `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java`
- `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/TorrentController.java`

- [ ] **Step 2: Create TorrentStatsResponse DTO**

```java
// backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/TorrentStatsResponse.java
package org.devMandali.magnetPlay.model;

public record TorrentStatsResponse(
    String fileId,
    long   totalSize,
    long   downloadedBytes,
    double completionPct,
    double downloadSpeedBps
) {}
```

- [ ] **Step 3: Create TorrentListResponse DTO**

```java
// backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/TorrentListResponse.java
package org.devMandali.magnetPlay.model;

import java.util.List;

public record TorrentListResponse(List<TorrentListItem> torrents) {
    public record TorrentListItem(
        String torrentId,
        String name,
        String state,
        long   totalSize,
        long   downloadedBytes,
        double completionPct,
        double downloadSpeedBps,
        List<FileItem> files
    ) {}

    public record FileItem(String id, String name, long size) {}
}
```

- [ ] **Step 4: Write Spring test for stats endpoint**

Create `backend/mp-spring/src/test/java/org/devMandali/magnetPlay/controller/TorrentStatsControllerTest.java`:

```java
package org.devMandali.magnetPlay.controller;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.reactive.WebFluxTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.reactive.server.WebTestClient;
import org.devMandali.magnetPlay.service.TorrentService;
import org.devMandali.magnetPlay.model.TorrentStatsResponse;
import reactor.core.publisher.Mono;
import static org.mockito.Mockito.when;

@WebFluxTest(TorrentController.class)
class TorrentStatsControllerTest {

    @Autowired WebTestClient client;
    @MockBean  TorrentService torrentService;

    @Test
    void stats_returns200() {
        var stats = new TorrentStatsResponse("abc:0", 1000L, 500L, 50.0, 1024.0);
        when(torrentService.getTorrentStats("abc", "abc:0")).thenReturn(Mono.just(stats));

        client.get()
              .uri("/v1/torrent/stats/abc?fileId=abc:0")
              .exchange()
              .expectStatus().isOk()
              .expectBody()
              .jsonPath("$.completionPct").isEqualTo(50.0);
    }
}
```

- [ ] **Step 5: Run test to confirm failure**

```bash
cd backend/mp-spring && ./mvnw test -pl . -Dtest=TorrentStatsControllerTest -q 2>&1 | tail -20
```

Expected: FAIL — endpoint does not exist yet.

- [ ] **Step 6: Add gRPC client methods**

In `TorrentGrpcClient.java`, add:

```java
public TorrentServiceGrpc.TorrentServiceBlockingStub stub() { return stub; }

public TorrentOuterClass.GetTorrentStatsResponse getTorrentStats(String infoHash, String fileId) {
    return stub.getTorrentStats(TorrentOuterClass.GetTorrentStatsRequest.newBuilder()
        .setInfoHash(infoHash).setFileId(fileId).build());
}

public TorrentOuterClass.ListTorrentsResponse listTorrents() {
    return stub.listTorrents(TorrentOuterClass.ListTorrentsRequest.newBuilder().build());
}

public TorrentOuterClass.PauseTorrentResponse pauseTorrent(String infoHash) {
    return stub.pauseTorrent(TorrentOuterClass.PauseTorrentRequest.newBuilder()
        .setInfoHash(infoHash).build());
}

public TorrentOuterClass.ResumeTorrentResponse resumeTorrent(String infoHash) {
    return stub.resumeTorrent(TorrentOuterClass.ResumeTorrentRequest.newBuilder()
        .setInfoHash(infoHash).build());
}

public TorrentOuterClass.DeleteTorrentResponse deleteTorrent(String infoHash, boolean deleteFiles) {
    return stub.deleteTorrent(TorrentOuterClass.DeleteTorrentRequest.newBuilder()
        .setInfoHash(infoHash).setDeleteFiles(deleteFiles).build());
}
```

- [ ] **Step 7: Add service methods**

In `TorrentService.java`, add:

```java
public Mono<TorrentStatsResponse> getTorrentStats(String infoHash, String fileId) {
    return Mono.fromCallable(() -> {
        var r = grpcClient.getTorrentStats(infoHash, fileId);
        var s = r.getStats();
        return new TorrentStatsResponse(s.getFileId(), s.getTotalSize(),
            s.getDownloadedBytes(), s.getCompletionPct(), s.getDownloadSpeedBps());
    }).subscribeOn(Schedulers.boundedElastic());
}

public Mono<TorrentListResponse> listTorrents() {
    return Mono.fromCallable(() -> {
        var r = grpcClient.listTorrents();
        var items = r.getTorrentsList().stream().map(t ->
            new TorrentListResponse.TorrentListItem(
                t.getTorrentId(), t.getName(), t.getState().name(),
                t.getTotalSize(), t.getDownloadedBytes(),
                t.getCompletionPct(), t.getDownloadSpeedBps(),
                t.getFilesList().stream().map(f ->
                    new TorrentListResponse.FileItem(f.getId(), f.getName(), f.getSize())
                ).toList()
            )
        ).toList();
        return new TorrentListResponse(items);
    }).subscribeOn(Schedulers.boundedElastic());
}

public Mono<String> pauseTorrent(String infoHash) {
    return Mono.fromCallable(() -> grpcClient.pauseTorrent(infoHash).getMessage())
               .subscribeOn(Schedulers.boundedElastic());
}

public Mono<String> resumeTorrent(String infoHash) {
    return Mono.fromCallable(() -> grpcClient.resumeTorrent(infoHash).getMessage())
               .subscribeOn(Schedulers.boundedElastic());
}

public Mono<String> deleteTorrent(String infoHash, boolean deleteFiles) {
    return Mono.fromCallable(() -> grpcClient.deleteTorrent(infoHash, deleteFiles).getMessage())
               .subscribeOn(Schedulers.boundedElastic());
}
```

- [ ] **Step 8: Add REST endpoints**

In `TorrentController.java`, add:

```java
@GetMapping("/stats/{infoHash}")
public Mono<ResponseEntity<TorrentStatsResponse>> getStats(
        @PathVariable String infoHash,
        @RequestParam String fileId) {
    return torrentService.getTorrentStats(infoHash, fileId)
        .map(ResponseEntity::ok)
        .onErrorReturn(ResponseEntity.notFound().build());
}

@GetMapping("/list")
public Mono<ResponseEntity<TorrentListResponse>> listTorrents() {
    return torrentService.listTorrents().map(ResponseEntity::ok);
}

@PostMapping("/pause/{infoHash}")
public Mono<ResponseEntity<String>> pause(@PathVariable String infoHash) {
    return torrentService.pauseTorrent(infoHash).map(ResponseEntity::ok);
}

@PostMapping("/resume/{infoHash}")
public Mono<ResponseEntity<String>> resume(@PathVariable String infoHash) {
    return torrentService.resumeTorrent(infoHash).map(ResponseEntity::ok);
}

@DeleteMapping("/{infoHash}")
public Mono<ResponseEntity<String>> delete(
        @PathVariable String infoHash,
        @RequestParam(defaultValue = "false") boolean deleteFiles) {
    return torrentService.deleteTorrent(infoHash, deleteFiles).map(ResponseEntity::ok);
}
```

- [ ] **Step 9: Run the test**

```bash
cd backend/mp-spring && ./mvnw test -pl . -Dtest=TorrentStatsControllerTest -q 2>&1 | tail -10
```

Expected: BUILD SUCCESS, 1 test passed.

- [ ] **Step 10: Run all Spring tests**

```bash
cd backend/mp-spring && ./mvnw test -q 2>&1 | tail -10
```

Expected: BUILD SUCCESS.

- [ ] **Step 11: Commit**

```bash
git add backend/mp-spring/src/
git commit -m "Feature(Spring): add stats, list, pause, resume, delete torrent endpoints"
```

---

## Phase 6 — Spring Boot: Session Tracking

### Task 6: SessionManager + session-aware streaming

**Files:**
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/session/StreamingSession.java`
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/session/SessionManager.java`
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/SessionResponse.java`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/TorrentController.java`

- [ ] **Step 1: Write SessionManager test**

Create `backend/mp-spring/src/test/java/org/devMandali/magnetPlay/session/SessionManagerTest.java`:

```java
package org.devMandali.magnetPlay.session;

import org.junit.jupiter.api.Test;
import static org.assertj.core.api.Assertions.assertThat;

class SessionManagerTest {

    @Test
    void openSession_appearsInActive() {
        var mgr = new SessionManager();
        var s = mgr.openSession("hash1", "hash1:0", "127.0.0.1", "TestAgent", 0L);
        assertThat(mgr.listActive()).hasSize(1);
        assertThat(mgr.listActive().get(0).sessionId()).isEqualTo(s.sessionId());
    }

    @Test
    void closeSession_movesToClosed() {
        var mgr = new SessionManager();
        var s = mgr.openSession("hash1", "hash1:0", "127.0.0.1", "TestAgent", 0L);
        mgr.closeSession(s.sessionId(), StreamingSession.CloseReason.COMPLETED);
        assertThat(mgr.listActive()).isEmpty();
        assertThat(mgr.listAll()).hasSize(1);
        assertThat(mgr.listAll().get(0).status()).isEqualTo(StreamingSession.SessionStatus.COMPLETED);
    }

    @Test
    void closedSessionsCappedAt100() {
        var mgr = new SessionManager();
        for (int i = 0; i < 110; i++) {
            var s = mgr.openSession("h", "h:0", "ip", "ua", 0);
            mgr.closeSession(s.sessionId(), StreamingSession.CloseReason.COMPLETED);
        }
        assertThat(mgr.listAll()).hasSizeLessThanOrEqualTo(100);
    }
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
cd backend/mp-spring && ./mvnw test -pl . -Dtest=SessionManagerTest -q 2>&1 | tail -10
```

Expected: compilation error — classes not found.

- [ ] **Step 3: Create StreamingSession**

```java
// backend/mp-spring/src/main/java/org/devMandali/magnetPlay/session/StreamingSession.java
package org.devMandali.magnetPlay.session;

import java.time.Instant;
import java.util.concurrent.atomic.AtomicLong;

public record StreamingSession(
    String        sessionId,
    String        infoHash,
    String        fileId,
    String        clientIp,
    String        userAgent,
    Instant       startTime,
    Instant       endTime,
    long          startByte,
    AtomicLong    bytesServed,
    SessionStatus status,
    CloseReason   closeReason
) {
    public enum SessionStatus { ACTIVE, COMPLETED, CANCELLED, ERROR }
    public enum CloseReason   { COMPLETED, CANCELLED, ERROR, UNKNOWN }

    public StreamingSession withClosed(Instant end, SessionStatus s, CloseReason r) {
        return new StreamingSession(sessionId, infoHash, fileId, clientIp, userAgent,
            startTime, end, startByte, bytesServed, s, r);
    }
}
```

- [ ] **Step 4: Create SessionManager**

```java
// backend/mp-spring/src/main/java/org/devMandali/magnetPlay/session/SessionManager.java
package org.devMandali.magnetPlay.session;

import org.springframework.stereotype.Component;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicLong;

@Component
public class SessionManager {
    private final ConcurrentHashMap<String, StreamingSession> active = new ConcurrentHashMap<>();
    private final Deque<StreamingSession> closed = new ArrayDeque<>();
    private static final int MAX_CLOSED = 100;

    public StreamingSession openSession(String infoHash, String fileId,
                                        String clientIp, String userAgent, long startByte) {
        var session = new StreamingSession(
            UUID.randomUUID().toString(), infoHash, fileId, clientIp, userAgent,
            Instant.now(), null, startByte, new AtomicLong(0),
            StreamingSession.SessionStatus.ACTIVE, null
        );
        active.put(session.sessionId(), session);
        return session;
    }

    public void closeSession(String sessionId, StreamingSession.CloseReason reason) {
        var session = active.remove(sessionId);
        if (session == null) return;
        var status = switch (reason) {
            case COMPLETED -> StreamingSession.SessionStatus.COMPLETED;
            case CANCELLED -> StreamingSession.SessionStatus.CANCELLED;
            default        -> StreamingSession.SessionStatus.ERROR;
        };
        var closed_ = session.withClosed(Instant.now(), status, reason);
        synchronized (closed) {
            closed.addLast(closed_);
            while (closed.size() > MAX_CLOSED) closed.removeFirst();
        }
    }

    public List<StreamingSession> listActive() {
        return new ArrayList<>(active.values());
    }

    public List<StreamingSession> listAll() {
        var result = new ArrayList<StreamingSession>();
        result.addAll(active.values());
        synchronized (closed) { result.addAll(closed); }
        return result;
    }
}
```

- [ ] **Step 5: Run SessionManager tests**

```bash
cd backend/mp-spring && ./mvnw test -pl . -Dtest=SessionManagerTest -q 2>&1 | tail -10
```

Expected: BUILD SUCCESS, 3 tests passed.

- [ ] **Step 6: Create SessionResponse DTO**

```java
// backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/SessionResponse.java
package org.devMandali.magnetPlay.model;

import java.time.Instant;
import java.util.List;

public record SessionResponse(List<SessionItem> sessions) {
    public record SessionItem(
        String  sessionId,
        String  infoHash,
        String  fileId,
        String  clientIp,
        Instant startTime,
        Instant endTime,
        long    startByte,
        long    bytesServed,
        String  status,
        String  closeReason
    ) {}
}
```

- [ ] **Step 7: Wire SessionManager into TorrentController.stream()**

Read current `TorrentController.java`. In the `stream()` method, inject `SessionManager` and add lifecycle hooks:

```java
// Add constructor injection:
private final SessionManager sessionManager;

// In stream() method, after resolving clientIp from request headers,
// wrap the existing Flux:
var session = sessionManager.openSession(infoHash, fileId, clientIp, userAgent, startByte);

return torrentService.streamFile(...)
    .doOnNext(buffer -> session.bytesServed().addAndGet(buffer.readableByteCount()))
    .doOnComplete(() -> sessionManager.closeSession(session.sessionId(), StreamingSession.CloseReason.COMPLETED))
    .doOnCancel(() -> sessionManager.closeSession(session.sessionId(), StreamingSession.CloseReason.CANCELLED))
    .doOnError(e -> sessionManager.closeSession(session.sessionId(), StreamingSession.CloseReason.ERROR));
```

To extract client IP (handle proxies):
```java
private String extractClientIp(ServerHttpRequest request) {
    String forwarded = request.getHeaders().getFirst("X-Forwarded-For");
    if (forwarded != null && !forwarded.isBlank()) return forwarded.split(",")[0].trim();
    var addr = request.getRemoteAddress();
    return addr != null ? addr.getAddress().getHostAddress() : "unknown";
}
```

- [ ] **Step 8: Add sessions REST endpoint**

In `TorrentController.java`:

```java
@GetMapping("/sessions")           // GET /v1/torrent/sessions
public Mono<ResponseEntity<SessionResponse>> getSessions(
        @RequestParam(defaultValue = "false") boolean activeOnly) {
    var sessions = activeOnly ? sessionManager.listActive() : sessionManager.listAll();
    var items = sessions.stream().map(s -> new SessionResponse.SessionItem(
        s.sessionId(), s.infoHash(), s.fileId(), s.clientIp(),
        s.startTime(), s.endTime(), s.startByte(), s.bytesServed().get(),
        s.status().name(), s.closeReason() != null ? s.closeReason().name() : null
    )).toList();
    return Mono.just(ResponseEntity.ok(new SessionResponse(items)));
}
```

- [ ] **Step 9: Run all Spring tests**

```bash
cd backend/mp-spring && ./mvnw test -q 2>&1 | tail -10
```

Expected: BUILD SUCCESS.

- [ ] **Step 10: Commit**

```bash
git add backend/mp-spring/src/
git commit -m "Feature(Spring): add session tracking with lifecycle hooks on stream Flux"
```

---

## Phase 7 — Frontend: TypeScript Types

### Task 7: Add new types to types/index.ts

**Files:**
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Read current types/index.ts**

- [ ] **Step 2: Add new types**

Append to `frontend/src/types/index.ts`:

```typescript
export interface TorrentFileStats {
  fileId: string;
  totalSize: number;
  downloadedBytes: number;
  completionPct: number;
  downloadSpeedBps: number;
}

export interface TorrentListItem {
  torrentId: string;
  name: string;
  state: 'TORRENT_ACTIVE' | 'TORRENT_PAUSED' | 'TORRENT_STOPPED';
  totalSize: number;
  downloadedBytes: number;
  completionPct: number;
  downloadSpeedBps: number;
  files: Array<{ id: string; name: string; size: number }>;
}

export interface StreamingSession {
  sessionId: string;
  infoHash: string;
  fileId: string;
  clientIp: string;
  startTime: string;
  endTime: string | null;
  startByte: number;
  bytesServed: number;
  status: 'ACTIVE' | 'COMPLETED' | 'CANCELLED' | 'ERROR';
  closeReason: string | null;
}
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/index.ts
git commit -m "Feature(Frontend): add TorrentFileStats, TorrentListItem, StreamingSession types"
```

---

## Phase 8 — Frontend: StatusPanel Component

### Task 8: Player-bar download status panel

> ⚠️ **Graph warning:** `App (React Root Component)` is the #2 god node (11 edges), center of the React Frontend Components community. Tasks 8 and 9 both modify it. Complete **both** StatusPanel and TorrentsPage wiring in a single `App.tsx` edit and a single commit — avoids two separate touches to the highest-risk frontend file.

**Files:**
- Create: `frontend/src/components/StatusPanel.tsx`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Create StatusPanel.tsx**

```tsx
// frontend/src/components/StatusPanel.tsx
import React, { useState, useEffect, useRef } from 'react';
import { TorrentFileStats } from '../types';

interface StatusPanelProps {
  infoHash: string;
  fileId: string;
  apiBase?: string;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function formatSpeed(bps: number): string {
  if (bps === 0) return '—';
  return `${formatBytes(bps)}/s`;
}

export function StatusPanel({ infoHash, fileId, apiBase = 'http://localhost:8080' }: StatusPanelProps) {
  const [open, setOpen] = useState(false);
  const [stats, setStats] = useState<TorrentFileStats | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (!open) {
      if (intervalRef.current) clearInterval(intervalRef.current);
      return;
    }
    const fetchStats = async () => {
      try {
        const res = await fetch(`${apiBase}/v1/torrent/stats/${infoHash}?fileId=${encodeURIComponent(fileId)}`);
        if (res.ok) setStats(await res.json());
      } catch { /* ignore network errors */ }
    };
    fetchStats();
    intervalRef.current = setInterval(fetchStats, 2000);
    return () => { if (intervalRef.current) clearInterval(intervalRef.current); };
  }, [open, infoHash, fileId, apiBase]);

  return (
    <div style={{ position: 'relative', display: 'inline-block' }}>
      <button
        onClick={() => setOpen(o => !o)}
        title="Download status"
        style={{
          background: 'transparent', border: 'none', color: '#fff',
          cursor: 'pointer', padding: '4px 8px', fontSize: 14,
        }}
      >
        ⬇ {stats ? `${stats.completionPct.toFixed(1)}%` : '…'}
      </button>

      {open && (
        <div style={{
          position: 'absolute', bottom: '36px', left: 0,
          background: 'rgba(20,20,20,0.95)', color: '#fff',
          borderRadius: 8, padding: '12px 16px', minWidth: 220,
          boxShadow: '0 4px 20px rgba(0,0,0,0.5)', zIndex: 9999,
        }}>
          <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 8 }}>Download Status</div>
          {stats ? (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '4px 12px', fontSize: 12 }}>
              <span style={{ color: '#aaa' }}>Total Size</span>
              <span>{formatBytes(stats.totalSize)}</span>
              <span style={{ color: '#aaa' }}>Downloaded</span>
              <span>{formatBytes(stats.downloadedBytes)}</span>
              <span style={{ color: '#aaa' }}>Progress</span>
              <span>{stats.completionPct.toFixed(1)}%</span>
              <span style={{ color: '#aaa' }}>Speed</span>
              <span>{formatSpeed(stats.downloadSpeedBps)}</span>
            </div>
          ) : (
            <div style={{ fontSize: 12, color: '#aaa' }}>Loading…</div>
          )}
          {stats && (
            <div style={{
              marginTop: 10, height: 4, background: '#333', borderRadius: 2,
            }}>
              <div style={{
                width: `${Math.min(stats.completionPct, 100)}%`,
                height: '100%', background: '#4ade80', borderRadius: 2,
                transition: 'width 0.3s',
              }} />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Wire StatusPanel into App.tsx**

Read `frontend/src/App.tsx`. Find where `SubtitlePanel` is rendered (it appears in the player controls area). Add `StatusPanel` next to it — pass the current `infoHash` and `fileId` from state.

```tsx
// Import at top of App.tsx:
import { StatusPanel } from './components/StatusPanel';

// In JSX where SubtitlePanel is rendered, add alongside it:
{currentInfoHash && currentFileId && (
  <StatusPanel infoHash={currentInfoHash} fileId={currentFileId} />
)}
```

- [ ] **Step 3: Start dev server and verify**

```bash
cd frontend && npm run dev
```

Open `http://localhost:5173`. Add a magnet link, select a file, start playing. The `⬇ …%` button should appear in the player bar. Click it — panel should show stats and refresh every 2s.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/StatusPanel.tsx frontend/src/App.tsx
git commit -m "Feature(Frontend): add StatusPanel download stats widget to player bar"
```

---

## Phase 9 — Frontend: Torrents Management Page

### Task 9: TorrentsPage with torrent table and sessions table

> ℹ️ **Graph note:** Do not commit `App.tsx` changes here separately — they were already committed in Task 8's final step (both StatusPanel + TorrentsPage wiring in one pass). This task only creates `TorrentsPage.tsx` as a new file.

**Files:**
- Create: `frontend/src/components/TorrentsPage.tsx`
- Modify: `frontend/src/App.tsx` *(combined with Task 8 edits — single commit)*

- [ ] **Step 1: Create TorrentsPage.tsx**

```tsx
// frontend/src/components/TorrentsPage.tsx
import React, { useState, useEffect } from 'react';
import { TorrentListItem, StreamingSession } from '../types';

interface TorrentsPageProps {
  apiBase?: string;
  onClose: () => void;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function formatSpeed(bps: number) {
  return bps === 0 ? '—' : `${formatBytes(bps)}/s`;
}

function stateBadge(state: TorrentListItem['state']) {
  const colors: Record<string, string> = {
    TORRENT_ACTIVE: '#4ade80', TORRENT_PAUSED: '#facc15', TORRENT_STOPPED: '#f87171',
  };
  const labels: Record<string, string> = {
    TORRENT_ACTIVE: 'Active', TORRENT_PAUSED: 'Paused', TORRENT_STOPPED: 'Stopped',
  };
  return (
    <span style={{
      display: 'inline-block', padding: '2px 8px', borderRadius: 12,
      background: colors[state] || '#888', color: '#000', fontSize: 11, fontWeight: 700,
    }}>
      {labels[state] || state}
    </span>
  );
}

export function TorrentsPage({ apiBase = 'http://localhost:8080', onClose }: TorrentsPageProps) {
  const [torrents, setTorrents] = useState<TorrentListItem[]>([]);
  const [sessions, setSessions] = useState<StreamingSession[]>([]);
  const [tab, setTab] = useState<'torrents' | 'sessions'>('torrents');
  const [loading, setLoading] = useState(true);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  const fetchData = async () => {
    try {
      const [t, s] = await Promise.all([
        fetch(`${apiBase}/v1/torrent/list`).then(r => r.json()),
        fetch(`${apiBase}/v1/torrent/sessions`).then(r => r.json()),
      ]);
      setTorrents(t.torrents ?? []);
      setSessions(s.sessions ?? []);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  };

  useEffect(() => {
    fetchData();
    const id = setInterval(fetchData, 3000);
    return () => clearInterval(id);
  }, []);

  const pause = async (id: string) => {
    await fetch(`${apiBase}/v1/torrent/pause/${id}`, { method: 'POST' });
    fetchData();
  };
  const resume = async (id: string) => {
    await fetch(`${apiBase}/v1/torrent/resume/${id}`, { method: 'POST' });
    fetchData();
  };
  const deleteTorrent = async (id: string, deleteFiles: boolean) => {
    await fetch(`${apiBase}/v1/torrent/${id}?deleteFiles=${deleteFiles}`, { method: 'DELETE' });
    setDeleteConfirm(null);
    fetchData();
  };

  const panelStyle: React.CSSProperties = {
    position: 'fixed', inset: 0, background: 'rgba(10,10,10,0.97)',
    color: '#fff', zIndex: 10000, display: 'flex', flexDirection: 'column',
    fontFamily: 'system-ui, sans-serif',
  };
  const th: React.CSSProperties = {
    padding: '10px 12px', textAlign: 'left', fontSize: 12,
    color: '#888', fontWeight: 600, borderBottom: '1px solid #333',
  };
  const td: React.CSSProperties = {
    padding: '10px 12px', fontSize: 13, borderBottom: '1px solid #222',
  };

  return (
    <div style={panelStyle}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', padding: '16px 24px', borderBottom: '1px solid #333' }}>
        <h2 style={{ margin: 0, fontSize: 18 }}>Torrent Dashboard</h2>
        <div style={{ marginLeft: 24, display: 'flex', gap: 8 }}>
          {(['torrents', 'sessions'] as const).map(t => (
            <button key={t} onClick={() => setTab(t)} style={{
              background: tab === t ? '#4ade80' : '#222', color: tab === t ? '#000' : '#fff',
              border: 'none', borderRadius: 6, padding: '6px 16px',
              cursor: 'pointer', fontSize: 13, fontWeight: 600,
            }}>
              {t === 'torrents' ? 'Torrents' : 'Sessions'}
            </button>
          ))}
        </div>
        <button onClick={onClose} style={{
          marginLeft: 'auto', background: 'transparent', border: 'none',
          color: '#aaa', fontSize: 22, cursor: 'pointer',
        }}>✕</button>
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflow: 'auto', padding: 24 }}>
        {loading ? (
          <div style={{ color: '#aaa' }}>Loading…</div>
        ) : tab === 'torrents' ? (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                {['Name', 'Status', 'Progress', 'Downloaded', 'Total', 'Speed', 'Actions'].map(h => (
                  <th key={h} style={th}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {torrents.length === 0 ? (
                <tr><td colSpan={7} style={{ ...td, color: '#555', textAlign: 'center' }}>No torrents</td></tr>
              ) : torrents.map(t => (
                <tr key={t.torrentId}>
                  <td style={td}><span title={t.torrentId}>{t.name}</span></td>
                  <td style={td}>{stateBadge(t.state)}</td>
                  <td style={{ ...td, minWidth: 120 }}>
                    <div style={{ background: '#333', borderRadius: 4, height: 6 }}>
                      <div style={{
                        width: `${Math.min(t.completionPct, 100)}%`, height: '100%',
                        background: '#4ade80', borderRadius: 4,
                      }} />
                    </div>
                    <span style={{ fontSize: 11, color: '#aaa' }}>{t.completionPct.toFixed(1)}%</span>
                  </td>
                  <td style={td}>{formatBytes(t.downloadedBytes)}</td>
                  <td style={td}>{formatBytes(t.totalSize)}</td>
                  <td style={td}>{formatSpeed(t.downloadSpeedBps)}</td>
                  <td style={td}>
                    <div style={{ display: 'flex', gap: 6 }}>
                      {t.state === 'TORRENT_ACTIVE' ? (
                        <button onClick={() => pause(t.torrentId)} style={{
                          background: '#facc15', color: '#000', border: 'none',
                          borderRadius: 4, padding: '3px 10px', cursor: 'pointer', fontSize: 12,
                        }}>Pause</button>
                      ) : (
                        <button onClick={() => resume(t.torrentId)} style={{
                          background: '#4ade80', color: '#000', border: 'none',
                          borderRadius: 4, padding: '3px 10px', cursor: 'pointer', fontSize: 12,
                        }}>Resume</button>
                      )}
                      <button onClick={() => setDeleteConfirm(t.torrentId)} style={{
                        background: '#f87171', color: '#fff', border: 'none',
                        borderRadius: 4, padding: '3px 10px', cursor: 'pointer', fontSize: 12,
                      }}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          /* Sessions tab */
          <>
            <h3 style={{ color: '#4ade80', margin: '0 0 12px' }}>Active Sessions ({sessions.filter(s => s.status === 'ACTIVE').length})</h3>
            <SessionTable sessions={sessions.filter(s => s.status === 'ACTIVE')} td={td} th={th} />
            <h3 style={{ color: '#aaa', margin: '24px 0 12px' }}>Closed Sessions</h3>
            <SessionTable sessions={sessions.filter(s => s.status !== 'ACTIVE')} td={td} th={th} />
          </>
        )}
      </div>

      {/* Delete confirm dialog */}
      {deleteConfirm && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10001,
        }}>
          <div style={{ background: '#1a1a1a', borderRadius: 12, padding: 32, maxWidth: 360 }}>
            <h3 style={{ margin: '0 0 8px' }}>Delete Torrent?</h3>
            <p style={{ color: '#aaa', fontSize: 14 }}>Also delete downloaded files from disk?</p>
            <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
              <button onClick={() => deleteTorrent(deleteConfirm, false)} style={{
                background: '#f87171', color: '#fff', border: 'none',
                borderRadius: 6, padding: '8px 16px', cursor: 'pointer',
              }}>Delete (keep files)</button>
              <button onClick={() => deleteTorrent(deleteConfirm, true)} style={{
                background: '#dc2626', color: '#fff', border: 'none',
                borderRadius: 6, padding: '8px 16px', cursor: 'pointer',
              }}>Delete + files</button>
              <button onClick={() => setDeleteConfirm(null)} style={{
                background: '#333', color: '#fff', border: 'none',
                borderRadius: 6, padding: '8px 16px', cursor: 'pointer',
              }}>Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function SessionTable({ sessions, td, th }: {
  sessions: StreamingSession[];
  td: React.CSSProperties;
  th: React.CSSProperties;
}) {
  if (sessions.length === 0) {
    return <div style={{ color: '#555', fontSize: 13, marginBottom: 8 }}>None</div>;
  }
  return (
    <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: 16 }}>
      <thead>
        <tr>
          {['Session ID', 'File', 'Client IP', 'Start Byte', 'Bytes Served', 'Start', 'End', 'Status'].map(h => (
            <th key={h} style={th}>{h}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {sessions.map(s => (
          <tr key={s.sessionId}>
            <td style={td}><span style={{ fontFamily: 'monospace', fontSize: 11 }}>{s.sessionId.slice(0, 8)}…</span></td>
            <td style={td}>{s.fileId}</td>
            <td style={td}>{s.clientIp}</td>
            <td style={td}>{s.startByte.toLocaleString()}</td>
            <td style={td}>{s.bytesServed.toLocaleString()} B</td>
            <td style={td}>{new Date(s.startTime).toLocaleTimeString()}</td>
            <td style={td}>{s.endTime ? new Date(s.endTime).toLocaleTimeString() : '—'}</td>
            <td style={td}><span style={{ color: s.status === 'ACTIVE' ? '#4ade80' : '#888', fontSize: 12 }}>{s.status}</span></td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```

- [ ] **Step 2: Add dashboard button and route in App.tsx**

Read `frontend/src/App.tsx`. Add a dashboard button (gear icon or "Dashboard" text) that toggles `showDashboard` state and renders `TorrentsPage` as a full-screen overlay:

```tsx
// Import:
import { TorrentsPage } from './components/TorrentsPage';

// State:
const [showDashboard, setShowDashboard] = useState(false);

// Button (add to header/nav area):
<button onClick={() => setShowDashboard(true)} title="Torrent Dashboard" style={{
  background: 'transparent', border: '1px solid #333', color: '#aaa',
  borderRadius: 6, padding: '6px 12px', cursor: 'pointer', fontSize: 13,
}}>
  Dashboard
</button>

// Overlay (at end of JSX, before closing root div):
{showDashboard && <TorrentsPage onClose={() => setShowDashboard(false)} />}
```

- [ ] **Step 3: Start dev server and verify UI**

```bash
cd frontend && npm run dev
```

- Open `http://localhost:5173`
- Click "Dashboard" → full-screen page opens
- Torrents tab: table with pause/delete buttons (verify with running Go + Spring)
- Sessions tab: active sessions appear while streaming a video
- Close button returns to player

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/TorrentsPage.tsx frontend/src/App.tsx
git commit -m "Feature(Frontend): add TorrentsPage with torrent management and session tracking tables"
```

---

## Phase 10 — Integration Test

### Task 10: End-to-end smoke test

**Files:**
- No new files — manual verification steps.

- [ ] **Step 1: Start Go sidecar**

```bash
cd backend/go-server && go run main.go
```

Expected: `gRPC server listening on :50051`

- [ ] **Step 2: Start Spring Boot**

```bash
cd backend/mp-spring && ./mvnw spring-boot:run -q
```

Expected: Spring started on port 8080.

- [ ] **Step 3: Add a test torrent**

```bash
curl -s -X POST http://localhost:8080/v1/torrent/add \
  -H "Content-Type: application/json" \
  -d '{"magnet": "<your-test-magnet-url>"}' | jq .
```

Expected: JSON with `torrentId`, `files[]`.

- [ ] **Step 4: Verify stats endpoint**

```bash
curl -s "http://localhost:8080/v1/torrent/stats/<infoHash>?fileId=<fileId>" | jq .
```

Expected: `{ "fileId": "...", "totalSize": N, "downloadedBytes": N, "completionPct": N, "downloadSpeedBps": N }`

- [ ] **Step 5: Verify list endpoint**

```bash
curl -s http://localhost:8080/v1/torrent/list | jq .torrents[0].state
```

Expected: `"TORRENT_ACTIVE"`

- [ ] **Step 6: Verify pause/resume**

```bash
curl -s -X POST http://localhost:8080/v1/torrent/pause/<infoHash>
curl -s http://localhost:8080/v1/torrent/list | jq .torrents[0].state
# Expected: "TORRENT_PAUSED"

curl -s -X POST http://localhost:8080/v1/torrent/resume/<infoHash>
curl -s http://localhost:8080/v1/torrent/list | jq .torrents[0].state
# Expected: "TORRENT_ACTIVE"
```

- [ ] **Step 7: Start streaming, verify sessions**

Open the frontend, play a video file, then:

```bash
curl -s "http://localhost:8080/v1/torrent/sessions?activeOnly=true" | jq .sessions[0]
```

Expected: session object with `status: "ACTIVE"`, `bytesServed > 0`.

- [ ] **Step 8: Final commit**

```bash
git add -A
git commit -m "Feature(Integration): end-to-end smoke test verified — dashboard + status widget working"
```

---

## Graph Validation Notes

From `graphify-out/GRAPH_REPORT.md`:

- `Repository` (god node, 11 edges): plan adds 3 new methods — all mutex-guarded. Verify no `RLock` where `Lock` needed in `SetPaused` / `Remove`.
- `TorrentController` (Community 0, cohesion 0.13): plan adds 6 endpoints. Post-MVP: extract to `TorrentManagementController`. Tracked as known debt.
- "End-to-End gRPC Video Streaming Pipeline" hyperedge (confidence 0.90): `GetTorrentStats` RPC plugs directly into this pipeline. Do not change `StreamFile` signature while implementing it.
- `App.tsx` (god node, 11 edges): TaskS 8+9 consolidated to single edit. Prevents double-touch regression.
- `NewTorrentService()` confirmed present: Task 4 Step 4 updates existing constructor, not recreating.
- After implementation: run `graphify update .` to refresh graph with new nodes (StatusPanel, TorrentsPage, SessionManager, SpeedTracker, 5 new RPCs).

## Self-Review: Spec Coverage Checklist

| Requirement | Covered by |
|-------------|-----------|
| Total size of selected video file | Task 8 `StatusPanel` → `stats.totalSize` |
| How much downloaded | Task 8 `StatusPanel` → `stats.downloadedBytes` |
| Current download speed | Task 2 `SpeedTracker` → Task 8 `stats.downloadSpeedBps` |
| Show on player bar or subtitle-style tab | Task 8 toggle button on player controls bar |
| All torrents — live/inactive/paused | Task 9 `TorrentsPage` torrents tab |
| Download progress per torrent | Task 9 progress bar + `completionPct` column |
| Pause torrent | Task 4 Go `PauseTorrent` + Task 9 Pause button |
| Delete torrent | Task 4 Go `DeleteTorrent` + Task 9 Delete + confirm dialog |
| Session tracking — active streaming | Task 6 `SessionManager.openSession()` on Flux subscribe |
| Session tracking — seeking | Each seek = new stream request = new session row (by design) |
| Active and closed session details table | Task 9 Sessions tab, two separate `SessionTable` renders |
