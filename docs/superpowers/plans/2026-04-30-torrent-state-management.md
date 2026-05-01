# Torrent State Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement four coordinated changes: persistent anacrolix torrents (session-only delete), FFmpeg/FFprobe stderr logging + subtitle extraction, shutdown-only dataDir cleanup, and removal of the "delete with files" option.

**Architecture:** Go sidecar owns all torrent lifetime — anacrolix handles never drop until shutdown. Spring and frontend are simplified to a single delete action. Proto is the contract shared by Go and Spring; changes there must trigger regeneration in both.

**Tech Stack:** Go (anacrolix/torrent), gRPC/protobuf, Spring Boot WebFlux, React/TypeScript

---

## File Map

| File | Change |
|------|--------|
| `backend/proto/torrent.proto` | Remove `bool delete_files = 2` from `DeleteTorrentRequest` |
| `backend/go-server/internal/torrent/service.go` | DeleteTorrent: remove Drop + deleteFiles block; runFFprobe: stderr+subtitles; AddTorrent: info log |
| `backend/go-server/internal/torrent/repository.go` | Remove pendingDelete/QueueDelete; fix GetOrAdd duplicate-drop; simplify Clearup; add CleanupDataDir |
| `backend/go-server/internal/grpc/server.go` | Reorder defers: CleanupDataDir registered first (runs last after client.Close) |
| `backend/go-server/internal/hls/transcoder.go` | Add FFmpeg stderr goroutine after cmd.Start |
| `backend/go-server/proto/` | Regenerated Go proto stubs (auto from make proto) |
| `backend/mp-spring/.../TorrentController.java` | Remove deleteFiles @RequestParam |
| `backend/mp-spring/.../TorrentService.java` | Remove deleteFiles param from deleteTorrent |
| `backend/mp-spring/.../TorrentGrpcClient.java` | Remove deleteFiles from DeleteTorrentRequest build |
| `frontend/src/components/TorrentsPage.tsx` | Simplify delete confirm dialog to single action |

---

## Task 1: Remove `delete_files` from Proto

**Files:**
- Modify: `backend/proto/torrent.proto:144`

- [ ] **Step 1: Edit the proto**

In `backend/proto/torrent.proto`, change line 144 from:
```proto
message DeleteTorrentRequest  { string info_hash = 1; bool delete_files = 2; }
```
to:
```proto
message DeleteTorrentRequest  { string info_hash = 1; }
```

- [ ] **Step 2: Regenerate Go proto stubs**

```bash
cd backend && make -f MakeFile proto
```

Expected: No errors. Files `backend/go-server/proto/torrent.pb.go` and `torrent_grpc.pb.go` updated — `DeleteTorrentRequest` no longer has `delete_files` field or `GetDeleteFiles()` method.

- [ ] **Step 3: Verify Go compiles**

```bash
cd backend/go-server && go build ./...
```

Expected: Compile error on `req.GetDeleteFiles()` in `service.go` (we fix that in Task 2). If the only error is that one call, proceed.

---

## Task 2: Simplify `DeleteTorrent` in service.go (Changes 1 + 4)

**Files:**
- Modify: `backend/go-server/internal/torrent/service.go:406-445`

**What to remove:**
- `info.Torrent().Drop()` call (line 413)
- entire `if req.GetDeleteFiles() { ... }` block (lines 418–443)

- [ ] **Step 1: Replace the DeleteTorrent function body**

In `service.go`, replace the `DeleteTorrent` function with:

```go
func (s *TorrentService) DeleteTorrent(ctx context.Context, req *pb.DeleteTorrentRequest) (*pb.DeleteTorrentResponse, error) {
	_, err := s.repo.GetTorrentInfo(req.GetInfoHash())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.GetInfoHash())
	}
	infoHash := req.GetInfoHash()
	s.repo.Remove(infoHash)
	s.trackerMu.Lock()
	delete(s.speedTrackers, infoHash)
	s.trackerMu.Unlock()
	return &pb.DeleteTorrentResponse{Success: true, Message: "removed from dashboard"}, nil
}
```

- [ ] **Step 2: Remove unused imports caused by removal**

The `os`, `path/filepath`, and `time` imports in `service.go` may still be used elsewhere (StreamFile uses `time.Sleep`; `os` is not used). Check: `os` was only used in the delete block. Remove `"os"` from the import list in `service.go` if it appears there and is now unused.

- [ ] **Step 3: Verify Go compiles**

```bash
cd backend/go-server && go build ./...
```

Expected: Clean compile. Fix any remaining import errors.

---

## Task 3: Fix Repository — Remove pendingDelete, Fix Duplicate-Drop, Add CleanupDataDir

**Files:**
- Modify: `backend/go-server/internal/torrent/repository.go`

**Three sub-changes:**

**3a — Remove `pendingDelete` infrastructure:**
- Remove fields `pendingDelete []string` and `deleteMu sync.Mutex` from `Repository` struct
- Remove `QueueDelete()` method entirely

**3b — Fix `GetOrAdd` duplicate-guard:**
- Remove `t.Drop()` on line 91 (inside the double-check lock block after metadata resolves)
- The comment says "Drop duplicate" — remove that line, keep the return of existing torrent

**3c — Simplify `Clearup` + add `CleanupDataDir`:**
- `Clearup()`: keep only the torrent Drop loop, remove the `pendingDelete` loop entirely
- Add new method `CleanupDataDir`

- [ ] **Step 1: Rewrite repository.go**

Replace the `Repository` struct definition:
```go
type Repository struct {
	client          *lt.Client
	dataDir         string
	torrents        map[string]*TorrentInfo
	mu              sync.RWMutex
	metadataTimeout time.Duration
}
```

- [ ] **Step 2: Fix GetOrAdd duplicate-guard (line ~89-93)**

Replace:
```go
	if existingInfo, exists := r.torrents[key]; exists {
		r.mu.Unlock()
		t.Drop() // Drop duplicate
		return existingInfo.torrent, nil
	}
```
with:
```go
	if existingInfo, exists := r.torrents[key]; exists {
		r.mu.Unlock()
		return existingInfo.torrent, nil
	}
```

- [ ] **Step 3: Replace QueueDelete + Clearup + add CleanupDataDir**

Remove the `QueueDelete` method entirely.

Replace `Clearup` with:
```go
func (r *Repository) Clearup() {
	r.mu.Lock()
	for key, tInfo := range r.torrents {
		tInfo.torrent.Drop()
		delete(r.torrents, key)
	}
	r.mu.Unlock()
}
```

Add after `Clearup`:
```go
func (r *Repository) CleanupDataDir(dataDir string) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		log.Printf("[shutdown] failed to read dataDir %s: %v", dataDir, err)
		return
	}
	for _, entry := range entries {
		path := filepath.Join(dataDir, entry.Name())
		log.Printf("[shutdown] removing %s", path)
		if err := os.RemoveAll(path); err != nil {
			log.Printf("[shutdown] failed to remove %s: %v", path, err)
		}
	}
}
```

- [ ] **Step 4: Verify imports in repository.go**

Ensure `"os"`, `"path/filepath"`, and `"log"` are imported (they are needed by `CleanupDataDir`). Remove `"sync"` sub-import for `deleteMu` if it was the only Mutex — the `mu sync.RWMutex` still uses `sync` so keep it. Remove `"strings"` only if unused (it's used in `GetTorrent`/`GetFile` for `strings.ToLower` — keep it).

- [ ] **Step 5: Build**

```bash
cd backend/go-server && go build ./...
```

Expected: Clean. Any reference to `QueueDelete` or `pendingDelete` or `deleteMu` will error — fix those call sites (the only caller was `service.go:DeleteTorrent` which we already removed in Task 2).

---

## Task 4: Reorder Defers in server.go (Shutdown Sequence)

**Files:**
- Modify: `backend/go-server/internal/grpc/server.go`

The current shutdown order has `client.Close()` registered as the first (and only) defer, which means it runs *before* `CleanupDataDir`. We need `CleanupDataDir` to run after `client.Close()`.

LIFO defer order: register `CleanupDataDir` first → it runs last (after `client.Close()`).

- [ ] **Step 1: Move client.Close defer and add CleanupDataDir defer**

In `server.go`, find the existing defer block around line 57-61:
```go
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}
	}()
```

Replace with two defers in this order (CleanupDataDir registered FIRST so it runs LAST):
```go
	defer repo.CleanupDataDir(cfg.DataDir)

	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}
	}()
```

Note: `repo` is declared after `client`, so move these two defers to after `repo` is initialized (after the `repo := torrent.NewRepository(...)` line). The `defer pm.Stop()` at the top can stay where it is — Prowlarr stop order doesn't matter here.

- [ ] **Step 2: Verify shutdown sequence in code**

After your edit, the relevant section of `StartServer` should look like:

```go
	client, err := torrent.NewClient(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}

	repo := torrent.NewRepository(client, cfg.DataDir, cfg.MetadataTimeout)

	defer repo.CleanupDataDir(cfg.DataDir)   // runs LAST (registered first)
	defer func() {                            // runs 2nd-to-last
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}
	}()
```

- [ ] **Step 3: Build**

```bash
cd backend/go-server && go build ./...
```

Expected: Clean compile.

---

## Task 5: Add FFmpeg Stderr Goroutine in transcoder.go

**Files:**
- Modify: `backend/go-server/internal/hls/transcoder.go:133-138`

- [ ] **Step 1: Add bufio import**

In `transcoder.go`, add `"bufio"` and `"strings"` to the import block (if not already present).

- [ ] **Step 2: Wire stderr goroutine after cmd.Start()**

In `ServeHTTP`, after the `cmd.Start()` block (after the `log.Printf("[remux] ffmpeg spawned ...")`), insert:

```go
	stderr, stderrErr := cmd.StderrPipe()
	if stderrErr == nil {
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				lower := strings.ToLower(line)
				if strings.Contains(lower, "error") ||
					strings.Contains(lower, "warning") ||
					strings.Contains(lower, "invalid") ||
					strings.Contains(lower, "failed") ||
					strings.Contains(lower, "no such") {
					log.Printf("[ffmpeg-stderr hash=%s fileId=%s] %s", infoHash, fileId, line)
				}
			}
		}()
	}
```

Place this block **before** `cmd.Start()` call — `StderrPipe` must be called before `Start`. Update the ordering:

```go
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[remux] stdout pipe error: %v", err)
		http.Error(w, "pipe error", http.StatusInternalServerError)
		return
	}

	stderr, stderrErr := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		log.Printf("[remux] ffmpeg start error: %v", err)
		http.Error(w, "ffmpeg start failed", http.StatusInternalServerError)
		return
	}
	log.Printf("[remux] ffmpeg spawned hash=%s fileId=%s seek=%.1fs", infoHash, fileId, seekSec)

	if stderrErr == nil {
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				lower := strings.ToLower(line)
				if strings.Contains(lower, "error") ||
					strings.Contains(lower, "warning") ||
					strings.Contains(lower, "invalid") ||
					strings.Contains(lower, "failed") ||
					strings.Contains(lower, "no such") {
					log.Printf("[ffmpeg-stderr hash=%s fileId=%s] %s", infoHash, fileId, line)
				}
			}
		}()
	}
```

- [ ] **Step 3: Build**

```bash
cd backend/go-server && go build ./...
```

Expected: Clean.

---

## Task 6: FFprobe Stderr + Subtitle Logging in service.go

**Files:**
- Modify: `backend/go-server/internal/torrent/service.go`

Three sub-changes:
- 6a: Add `SubtitleTrack` type, add `Subtitles` to `probeResult`
- 6b: Update `runFFprobe` to capture stderr, extract subtitles, log structured summary
- 6c: Add torrent info log in `AddTorrent` after metadata resolves

- [ ] **Step 1: Add SubtitleTrack type and update probeResult**

After the `probeResult` struct definition (around line 42), replace with:

```go
type SubtitleTrack struct {
	Index    int
	Language string
	Codec    string
	Title    string
}

type probeResult struct {
	DurationSec float64
	AudioTracks []*pb.AudioTrack
	VideoCodec  string
	AudioCodec  string
	Subtitles   []SubtitleTrack
}
```

- [ ] **Step 2: Rewrite runFFprobe to capture stderr and log structured output**

Replace the `runFFprobe` function:

```go
func runFFprobe(ffprobePath string, r io.Reader) *probeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		"-analyzeduration", "3000000",
		"-probesize", "3000000",
		"pipe:0",
	)
	cmd.Stdin = r

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf

	log.Printf("[ffprobe] exec started (timeout=12s probesize=3MB)")
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			log.Printf("[ffprobe] timed out after 12s")
		} else {
			log.Printf("[ffprobe] exec failed: %v", err)
		}
		if stderrBuf.Len() > 0 {
			log.Printf("[ffprobe-stderr] %s", stderrBuf.String())
		}
		return nil
	}

	if stderrBuf.Len() > 0 {
		log.Printf("[ffprobe-stderr] %s", stderrBuf.String())
	}
	log.Printf("[ffprobe] exec ok output_bytes=%d", outBuf.Len())

	var probe struct {
		Streams []struct {
			Index     int    `json:"index"`
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Tags      struct {
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"tags"`
			Duration string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &probe); err != nil {
		log.Printf("[ffprobe] json parse error: %v", err)
		return nil
	}

	result := &probeResult{}
	for _, s := range probe.Streams {
		if s.CodecType == "video" && result.VideoCodec == "" {
			result.VideoCodec = s.CodecName
			if d, err := strconv.ParseFloat(s.Duration, 64); err == nil {
				result.DurationSec = d
			}
		}
		if s.CodecType == "audio" {
			if result.AudioCodec == "" {
				result.AudioCodec = s.CodecName
			}
			result.AudioTracks = append(result.AudioTracks, &pb.AudioTrack{
				Index:    int32(s.Index),
				Language: s.Tags.Language,
				Codec:    s.CodecName,
				Title:    s.Tags.Title,
			})
		}
		if s.CodecType == "subtitle" {
			result.Subtitles = append(result.Subtitles, SubtitleTrack{
				Index:    s.Index,
				Language: s.Tags.Language,
				Codec:    s.CodecName,
				Title:    s.Tags.Title,
			})
		}
	}
	if result.DurationSec == 0 {
		if d, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			result.DurationSec = d
		}
	}

	log.Printf("[ffprobe] file=<stream> duration=%.1fs video=%s audio=%s audio_tracks=%d subtitles=%d",
		result.DurationSec, result.VideoCodec, result.AudioCodec,
		len(result.AudioTracks), len(result.Subtitles))
	return result
}
```

Note: `cmd.Output()` is replaced by `cmd.Stdout = &outBuf` + `cmd.Stderr = &stderrBuf` + `cmd.Run()`. Remove the old `out, err := cmd.Output()` usage.

- [ ] **Step 3: Add torrent info log in AddTorrent**

In `AddTorrent`, after `t.Info()` is confirmed non-nil (around line 90-94), add:

```go
	info := t.Info()
	if info == nil {
		return nil, status.Error(codes.Unavailable, "")
	}
	infoHash := t.InfoHash().String()

	var totalSize int64
	for _, f := range t.Files() {
		totalSize += f.Length()
	}
	log.Printf("[torrent] name=%s infoHash=%s files=%d totalSize=%d",
		info.Name, infoHash, len(t.Files()), totalSize)
```

- [ ] **Step 4: Build**

```bash
cd backend/go-server && go build ./...
```

Expected: Clean compile.

---

## Task 7: Spring Boot — Remove deleteFiles from Controller, Service, GrpcClient

**Files:**
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/TorrentController.java:107-111`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java:132-136`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/TorrentGrpcClient.java:130-139`

- [ ] **Step 1: Update TorrentController.java**

Replace the `delete` endpoint:
```java
@DeleteMapping("/{infoHash}")
public Mono<ResponseEntity<String>> delete(
        @PathVariable String infoHash,
        @RequestParam(defaultValue = "false") boolean deleteFiles) {
    return service.deleteTorrent(infoHash, deleteFiles).map(ResponseEntity::ok);
}
```
with:
```java
@DeleteMapping("/{infoHash}")
public Mono<ResponseEntity<String>> delete(@PathVariable String infoHash) {
    return service.deleteTorrent(infoHash).map(ResponseEntity::ok);
}
```

- [ ] **Step 2: Update TorrentService.java**

Replace:
```java
public Mono<String> deleteTorrent(String infoHash, boolean deleteFiles) {
    return grpcClient.deleteTorrent(infoHash, deleteFiles)
            .map(r -> r.getMessage())
            .doOnError(e -> logger.error("deleteTorrent error for {}", infoHash, e));
}
```
with:
```java
public Mono<String> deleteTorrent(String infoHash) {
    return grpcClient.deleteTorrent(infoHash)
            .map(r -> r.getMessage())
            .doOnError(e -> logger.error("deleteTorrent error for {}", infoHash, e));
}
```

- [ ] **Step 3: Update TorrentGrpcClient.java**

Replace:
```java
public Mono<DeleteTorrentResponse> deleteTorrent(String infoHash, boolean deleteFiles) {
    return Mono.fromCallable(() -> torrentServiceBlockingStub
            .withDeadlineAfter(10, TimeUnit.SECONDS)
            .deleteTorrent(DeleteTorrentRequest.newBuilder()
                    .setInfoHash(infoHash)
                    .setDeleteFiles(deleteFiles)
                    .build()))
            .subscribeOn(grpcScheduler)
            .doOnError(e -> logger.error("deleteTorrent gRPC error for {}", infoHash, e));
}
```
with:
```java
public Mono<DeleteTorrentResponse> deleteTorrent(String infoHash) {
    return Mono.fromCallable(() -> torrentServiceBlockingStub
            .withDeadlineAfter(10, TimeUnit.SECONDS)
            .deleteTorrent(DeleteTorrentRequest.newBuilder()
                    .setInfoHash(infoHash)
                    .build()))
            .subscribeOn(grpcScheduler)
            .doOnError(e -> logger.error("deleteTorrent gRPC error for {}", infoHash, e));
}
```

- [ ] **Step 4: Compile Spring (this also regenerates Java proto from .proto file)**

```bash
cd backend/mp-spring && ./mvnw compile -q
```

Expected: BUILD SUCCESS. The Java proto regeneration removes `deleteFiles`/`setDeleteFiles` from `DeleteTorrentRequest`. If `setDeleteFiles` compile error appears, verify `torrent.proto` change from Task 1 was saved correctly.

---

## Task 8: Frontend — Simplify Delete Confirm Dialog

**Files:**
- Modify: `frontend/src/components/TorrentsPage.tsx`

- [ ] **Step 1: Update deleteTorrent function signature**

Replace:
```ts
const deleteTorrent = async (id: string, deleteFiles: boolean) => {
  setDeleteConfirm(null);
  await withAction(id, () => fetch(`${apiBase}/v1/torrent/${id}?deleteFiles=${deleteFiles}`, { method: 'DELETE' }));
};
```
with:
```ts
const deleteTorrent = async (id: string) => {
  setDeleteConfirm(null);
  await withAction(id, () => fetch(`${apiBase}/v1/torrent/${id}`, { method: 'DELETE' }));
};
```

- [ ] **Step 2: Replace confirm dialog JSX**

Replace the entire `{deleteConfirm && ( ... )}` block (lines 188-212) with:

```tsx
{deleteConfirm && (
  <div style={{
    position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)',
    display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10001,
  }}>
    <div style={{ background: '#1a1a1a', borderRadius: 12, padding: 32, maxWidth: 360 }}>
      <h3 style={{ margin: '0 0 8px' }}>Remove Torrent?</h3>
      <p style={{ color: '#aaa', fontSize: 14 }}>Remove from dashboard. Torrent continues seeding in background.</p>
      <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
        <button onClick={() => deleteTorrent(deleteConfirm)} style={{
          background: '#f87171', color: '#fff', border: 'none',
          borderRadius: 6, padding: '8px 16px', cursor: 'pointer',
        }}>Remove from dashboard</button>
        <button onClick={() => setDeleteConfirm(null)} style={{
          background: '#333', color: '#fff', border: 'none',
          borderRadius: 6, padding: '8px 16px', cursor: 'pointer',
        }}>Cancel</button>
      </div>
    </div>
  </div>
)}
```

- [ ] **Step 3: TypeScript check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: No errors. The `deleteTorrent` call in the dialog now passes one argument matching the updated signature.

---

## Task 9: Run Go Tests

- [ ] **Step 1: Run all Go tests**

```bash
cd backend/go-server && go test ./...
```

Expected: PASS or no test files (test suite is minimal). Any failure indicates a broken contract — fix before proceeding.

---

## Task 10: Integration Smoke Test

- [ ] **Step 1: Start Go sidecar**

```bash
cd backend/go-server && go run main.go
```

Expected: Server starts on :50051, no crash.

- [ ] **Step 2: Start Spring Boot**

```bash
cd backend/mp-spring && ./mvnw spring-boot:run -q
```

Expected: Started on :8080, no gRPC connection error.

- [ ] **Step 3: Verify delete endpoint (no deleteFiles param)**

```bash
curl -s -X DELETE http://localhost:8080/v1/torrent/SOMEHASH
```

Expected: 404 (hash not found) or 200 — either way no 400 from unexpected deleteFiles param handling.

- [ ] **Step 4: Start frontend, open Torrent Dashboard**

```bash
cd frontend && npm run dev
```

Open browser at http://localhost:5173. Navigate to Torrent Dashboard. Click Delete on any torrent.

Expected: Dialog shows **"Remove from dashboard"** and **"Cancel"** buttons only — no "Delete + files" button.

- [ ] **Step 5: Verify shutdown cleanup**

Stop Go sidecar with Ctrl+C. Check logs for `[shutdown] removing ...` entries showing `CleanupDataDir` ran after `client.Close()`.

Expected: Log line sequence:
```
Shutting down...
[shutdown] removing ./downloads/<torrent-name>
```
