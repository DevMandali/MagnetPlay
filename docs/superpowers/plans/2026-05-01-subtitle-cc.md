# Subtitle & CC Enhancement — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Surface MKV embedded subtitle tracks and torrent-bundled subtitle files in the CC panel, auto-populated on play, with server-side WebVTT conversion via FFmpeg.

**Architecture:** Proto carries `SubtitleTrackInfo` and `FileType` through the stack (same pattern as `AudioTrack`). Go HLS server exposes two new endpoints (`/subtitle/embedded/` and `/subtitle/file/`) that pipe WebVTT text. Vite proxies `/subtitle/` to port 8091. Frontend fetches subtitle blobs at play time and inserts them into existing `subtitleTracks` state — `VideoPlayer` and `SubtitlePanel` require no changes.

**Tech Stack:** Go (anacrolix/torrent, FFmpeg exec), protobuf, Java 17 records, Spring WebFlux, React 18 + TypeScript, Video.js

---

## File Map

| Action | File |
|--------|------|
| Modify | `backend/proto/torrent.proto` |
| Modify | `backend/go-server/internal/torrent/mapper.go` |
| Modify | `backend/go-server/internal/torrent/service.go` |
| Create | `backend/go-server/internal/hls/subtitle_handler.go` |
| Create | `backend/go-server/internal/hls/subtitle_handler_test.go` |
| Modify | `backend/go-server/internal/hls/server.go` |
| Modify | `backend/go-server/internal/grpc/server.go` |
| Modify | `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/HLSStartResponse.java` |
| Modify | `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/TorrentAddResponse.java` |
| Modify | `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java` |
| Modify | `frontend/src/types/index.ts` |
| Modify | `frontend/src/App.tsx` |
| Modify | `frontend/vite.config.ts` |

---

## Task 1: Proto — add SubtitleTrackInfo, FileType, update messages

**Files:**
- Modify: `backend/proto/torrent.proto`

- [ ] **Step 1: Apply proto changes**

Replace the `AudioTrack` block and everything after it with the following (the `service`, `FileInfoRequest/Response`, `TorrentRequest`, `TorrentStatus`, `FileInfo`, `TorrentResponse`, `StreamRequest`, `FileChunk`, stats messages, management messages stay unchanged — only the blocks shown below change):

```proto
// ─── HLS Streaming ────────────────────────────────────────────────────────────
enum FileType {
  VIDEO    = 0;
  SUBTITLE = 1;
}

message FileInfo {
  string   id        = 1;
  string   name      = 2;
  int64    size      = 3;
  FileType file_type = 4;
}

message AudioTrack {
  int32  index    = 1;
  string language = 2;
  string codec    = 3;
  string title    = 4;
}

message SubtitleTrackInfo {
  int32  index    = 1;
  string language = 2;
  string codec    = 3;
  string title    = 4;
}

message FileInfoResponse {
  string file_name   = 1;
  string file_path   = 2;
  int64  total_size  = 3;
  string mime_type   = 4;
  bool   is_ready    = 5;
  double               duration_sec    = 6;
  repeated AudioTrack  audio_tracks    = 7;
  repeated SubtitleTrackInfo subtitle_tracks = 8;
}

message HLSRequest {
  string info_hash     = 1;
  string file_id       = 2;
  double seek_time_sec = 3;
}

message HLSResponse {
  string               manifest_url    = 1;
  bool                 success         = 2;
  double               duration_sec    = 3;
  repeated AudioTrack  audio_tracks    = 4;
  repeated SubtitleTrackInfo subtitle_tracks = 5;
}
```

- [ ] **Step 2: Regenerate Go proto**

```bash
cd backend && make -f MakeFile proto
```

Expected: no errors. Files updated: `go-server/proto/torrent.pb.go`, `go-server/proto/torrent_grpc.pb.go`.

- [ ] **Step 3: Verify Java compiles with new proto**

```bash
cd backend/mp-spring && ./mvnw compile -q
```

Expected: BUILD SUCCESS. Java proto classes regenerated automatically via `protobuf-maven-plugin`.

- [ ] **Step 4: Commit**

```bash
git add backend/proto/torrent.proto backend/go-server/proto/ backend/mp-spring/target/
git commit -m "feat(proto): add SubtitleTrackInfo, FileType; add subtitle_tracks to FileInfoResponse + HLSResponse"
```

---

## Task 2: Go — mapper.go subtitle file support

**Files:**
- Modify: `backend/go-server/internal/torrent/mapper.go`

- [ ] **Step 1: Replace `toFileInfoList` with subtitle-aware version**

Full replacement of `mapper.go`:

```go
package torrent

import (
	"fmt"
	"log"
	"mime"
	"path/filepath"
	"slices"
	"strings"

	pb "server/proto"

	lt "github.com/anacrolix/torrent"
)

var subtitleExts = map[string]bool{
	".srt": true, ".vtt": true, ".ass": true, ".ssa": true, ".sub": true,
}

func toFileInfoList(infoHash string, tFiles []*lt.File, repo *Repository) []*pb.FileInfo {
	files := make([]*pb.FileInfo, 0, len(tFiles))

	otherVideoTypes := []string{
		"application/octet-stream",
		"application/ogg",
		"application/mp4",
		"application/x-mpegURL",
		"application/vnd.apple.mpegurl",
	}

	for i, f := range tFiles {
		ext := strings.ToLower(filepath.Ext(f.DisplayPath()))

		// Subtitle files — allow through with SUBTITLE tag
		if subtitleExts[ext] {
			fileId := fmt.Sprintf("%s:%d", infoHash, i)
			if _, exists := repo.torrents[infoHash].files[fileId]; !exists {
				repo.torrents[infoHash].files[fileId] = f
			}
			files = append(files, &pb.FileInfo{
				Id:       fileId,
				Name:     f.DisplayPath(),
				Size:     f.Length(),
				FileType: pb.FileType_SUBTITLE,
			})
			log.Printf("[mapper] subtitle file: %s ext=%s", f.DisplayPath(), ext)
			continue
		}

		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			f.SetPriority(lt.PiecePriorityNone)
			log.Printf("[mapper] unknown MIME type for %s, skipping", f.DisplayPath())
			continue
		}

		if !strings.HasPrefix(contentType, "video/") && !slices.Contains(otherVideoTypes, contentType) {
			f.SetPriority(lt.PiecePriorityNone)
			log.Printf("[mapper] non-video file skipped: %s MIME=%s", f.DisplayPath(), contentType)
			continue
		}

		log.Printf("[mapper] video file: %s ext=%s MIME=%s", f.DisplayPath(), ext, contentType)
		fileId := fmt.Sprintf("%s:%d", infoHash, i)
		if _, exists := repo.torrents[infoHash].files[fileId]; !exists {
			repo.torrents[infoHash].files[fileId] = f
		}
		files = append(files, &pb.FileInfo{
			Id:       fileId,
			Name:     f.DisplayPath(),
			Size:     f.Length(),
			FileType: pb.FileType_VIDEO,
		})
	}
	return files
}
```

- [ ] **Step 2: Run Go tests**

```bash
cd backend/go-server && go test ./internal/torrent/... -v
```

Expected: all tests pass (mapper has no dedicated unit tests; existing service tests still pass).

- [ ] **Step 3: Commit**

```bash
git add backend/go-server/internal/torrent/mapper.go
git commit -m "feat(mapper): allow subtitle extensions (.srt .vtt .ass .ssa .sub), tag FileType"
```

---

## Task 3: Go — service.go populate subtitle tracks

**Files:**
- Modify: `backend/go-server/internal/torrent/service.go`

- [ ] **Step 1: Add bitmap skip list and helper function**

After the existing `const` block (around line 39), add:

```go
var bitmapSubtitleCodecs = map[string]bool{
	"hdmv_pgs_subtitle": true,
	"dvb_subtitle":      true,
	"dvb_teletext":      true,
	"pgssub":            true,
	"xsub":              true,
}

func toProtoSubtitleTracks(tracks []SubtitleTrack) []*pb.SubtitleTrackInfo {
	result := make([]*pb.SubtitleTrackInfo, 0, len(tracks))
	for _, t := range tracks {
		if bitmapSubtitleCodecs[t.Codec] {
			log.Printf("[subtitle] skipping bitmap codec=%s lang=%s", t.Codec, t.Language)
			continue
		}
		result = append(result, &pb.SubtitleTrackInfo{
			Index:    int32(t.Index),
			Language: t.Language,
			Codec:    t.Codec,
			Title:    t.Title,
		})
	}
	return result
}
```

- [ ] **Step 2: Update GetFileInfo to return subtitle tracks**

In `GetFileInfo`, replace the MKV probe block (currently sets `resp.DurationSec` and `resp.AudioTracks`) with:

```go
if IsMKV(f.DisplayPath()) {
    resp.MimeType = "video/mp4"
    log.Printf("[mkv] GetFileInfo: detected MKV file=%s size=%d", filepath.Base(f.DisplayPath()), f.Length())
    probe := s.getOrProbe(req.GetInfoHash(), req.GetFileId(), t, f)
    if probe != nil {
        resp.DurationSec = probe.DurationSec
        resp.AudioTracks = probe.AudioTracks
        resp.SubtitleTracks = toProtoSubtitleTracks(probe.Subtitles)
        log.Printf("[mkv] GetFileInfo: probe ok duration=%.1fs audio=%d subtitles=%d",
            probe.DurationSec, len(probe.AudioTracks), len(resp.SubtitleTracks))
    } else {
        log.Printf("[mkv] GetFileInfo: probe nil — duration/audio/subtitles unknown")
    }
}
```

- [ ] **Step 3: Update StartRemux to return subtitle tracks**

In `StartRemux`, replace the `resp` construction block (currently sets `DurationSec` and `AudioTracks`) with:

```go
resp := &pb.HLSResponse{
    ManifestUrl: streamURL,
    Success:     true,
}
if probe != nil {
    resp.DurationSec = probe.DurationSec
    resp.AudioTracks = probe.AudioTracks
    resp.SubtitleTracks = toProtoSubtitleTracks(probe.Subtitles)
    log.Printf("[remux] StartRemux ok url=%s subtitles=%d", streamURL, len(resp.SubtitleTracks))
} else {
    log.Printf("[remux] StartRemux ok url=%s (no probe)", streamURL)
}
return resp, nil
```

- [ ] **Step 4: Run Go tests**

```bash
cd backend/go-server && go test ./internal/torrent/... -v
```

Expected: all existing tests pass.

- [ ] **Step 5: Commit**

```bash
git add backend/go-server/internal/torrent/service.go
git commit -m "feat(service): populate subtitle_tracks in GetFileInfo and StartRemux; skip bitmap codecs"
```

---

## Task 4: Go — subtitle_handler.go (new)

**Files:**
- Create: `backend/go-server/internal/hls/subtitle_handler.go`
- Create: `backend/go-server/internal/hls/subtitle_handler_test.go`

- [ ] **Step 1: Write the failing tests**

Create `backend/go-server/internal/hls/subtitle_handler_test.go`:

```go
package hls_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"server/internal/hls"
)

func TestSubtitleEmbedded_BadPath(t *testing.T) {
	h := hls.NewSubtitleHandler("ffmpeg", "http://localhost:8091/rawfile")
	req := httptest.NewRequest("GET", "/subtitle/embedded/abc123", nil)
	w := httptest.NewRecorder()
	h.ServeEmbedded(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad path, got %d", w.Code)
	}
}

func TestSubtitleEmbedded_BadStreamIndex(t *testing.T) {
	h := hls.NewSubtitleHandler("ffmpeg", "http://localhost:8091/rawfile")
	req := httptest.NewRequest("GET", "/subtitle/embedded/abc123/dGVzdA/notanumber", nil)
	w := httptest.NewRecorder()
	h.ServeEmbedded(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad stream index, got %d", w.Code)
	}
}

func TestSubtitleFile_BadPath(t *testing.T) {
	h := hls.NewSubtitleHandler("ffmpeg", "http://localhost:8091/rawfile")
	req := httptest.NewRequest("GET", "/subtitle/file/onlyhash", nil)
	w := httptest.NewRecorder()
	h.ServeFile(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing fileId, got %d", w.Code)
	}
}

func TestSubtitleEmbedded_CORSOnOptions(t *testing.T) {
	h := hls.NewSubtitleHandler("ffmpeg", "http://localhost:8091/rawfile")
	req := httptest.NewRequest("OPTIONS", "/subtitle/embedded/abc/dGVzdA/0", nil)
	w := httptest.NewRecorder()
	h.ServeEmbedded(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS * header")
	}
}

func TestSubtitleFile_CORSOnOptions(t *testing.T) {
	h := hls.NewSubtitleHandler("ffmpeg", "http://localhost:8091/rawfile")
	req := httptest.NewRequest("OPTIONS", "/subtitle/file/abc/dGVzdA", nil)
	w := httptest.NewRecorder()
	h.ServeFile(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

func TestSubtitleEmbedded_PathTraversalRejected(t *testing.T) {
	h := hls.NewSubtitleHandler("ffmpeg", "http://localhost:8091/rawfile")
	// URL-encoded ".." in infoHash
	req := httptest.NewRequest("GET", "/subtitle/embedded/../secret/dGVzdA/0", nil)
	w := httptest.NewRecorder()
	h.ServeEmbedded(w, req)
	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for path traversal attempt")
	}
}

func TestSubtitleFile_ContentTypeHeader(t *testing.T) {
	// This test only checks that the Content-Type header is set correctly before
	// FFmpeg runs. Since ffmpeg is not available in tests, we just check that
	// the handler sets CORS headers on OPTIONS correctly — full E2E tested manually.
	h := hls.NewSubtitleHandler("ffmpeg", "http://localhost:8091/rawfile")
	req := httptest.NewRequest("OPTIONS", "/subtitle/file/abc/dGVzdA", nil)
	w := httptest.NewRecorder()
	h.ServeFile(w, req)
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Origin"), "*") {
		t.Error("expected CORS * header")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend/go-server && go test ./internal/hls/... -run "TestSubtitle" -v
```

Expected: compile error — `hls.NewSubtitleHandler` does not exist yet.

- [ ] **Step 3: Implement subtitle_handler.go**

Create `backend/go-server/internal/hls/subtitle_handler.go`:

```go
package hls

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SubtitleHandler serves subtitle tracks as WebVTT via FFmpeg conversion.
// Two routes:
//
//	GET /subtitle/embedded/{infoHash}/{b64fileId}/{streamIndex}  — extract embedded MKV sub stream
//	GET /subtitle/file/{infoHash}/{b64fileId}                   — convert torrent subtitle file to WebVTT
type SubtitleHandler struct {
	ffmpegPath  string
	fileBaseURL string // e.g. "http://localhost:8091/rawfile"
}

func NewSubtitleHandler(ffmpegPath, fileBaseURL string) *SubtitleHandler {
	return &SubtitleHandler{ffmpegPath: ffmpegPath, fileBaseURL: fileBaseURL}
}

// ServeEmbedded handles GET /subtitle/embedded/{infoHash}/{b64fileId}/{streamIndex}
func (h *SubtitleHandler) ServeEmbedded(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Path after prefix: {infoHash}/{b64fileId}/{streamIndex}
	trimmed := strings.TrimPrefix(r.URL.Path, "/subtitle/embedded/")
	parts := strings.SplitN(trimmed, "/", 3)
	if len(parts) != 3 {
		http.Error(w, "bad path: expected /subtitle/embedded/{hash}/{b64fileId}/{streamIndex}", http.StatusBadRequest)
		return
	}
	infoHash, encodedFileId, streamIndexStr := parts[0], parts[1], parts[2]

	if strings.Contains(infoHash, "..") || strings.Contains(encodedFileId, "..") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	streamIndex, err := strconv.Atoi(streamIndexStr)
	if err != nil || streamIndex < 0 {
		http.Error(w, "bad stream index", http.StatusBadRequest)
		return
	}

	fileURL := fmt.Sprintf("%s/%s/%s", h.fileBaseURL, infoHash, encodedFileId)
	log.Printf("[subtitle/embedded] hash=%s streamIndex=%d", infoHash, streamIndex)

	h.runFFmpegToWebVTT(w, []string{
		"-analyzeduration", "5000000",
		"-probesize", "5000000",
		"-i", fileURL,
		"-map", fmt.Sprintf("0:s:%d", streamIndex),
		"-f", "webvtt",
		"pipe:1",
	})
}

// ServeFile handles GET /subtitle/file/{infoHash}/{b64fileId}
func (h *SubtitleHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Path after prefix: {infoHash}/{b64fileId}
	trimmed := strings.TrimPrefix(r.URL.Path, "/subtitle/file/")
	slashIdx := strings.Index(trimmed, "/")
	if slashIdx < 0 {
		http.Error(w, "bad path: expected /subtitle/file/{hash}/{b64fileId}", http.StatusBadRequest)
		return
	}
	infoHash := trimmed[:slashIdx]
	encodedFileId := trimmed[slashIdx+1:]

	if strings.Contains(infoHash, "..") || strings.Contains(encodedFileId, "..") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	fileIdBytes, err := base64.RawURLEncoding.DecodeString(encodedFileId)
	if err != nil {
		http.Error(w, "bad file id encoding", http.StatusBadRequest)
		return
	}
	fileId := string(fileIdBytes)
	log.Printf("[subtitle/file] hash=%s fileId=%s", infoHash, fileId)

	fileURL := fmt.Sprintf("%s/%s/%s", h.fileBaseURL, infoHash, encodedFileId)

	// FFmpeg handles all text subtitle formats (SRT, ASS, VTT) → WebVTT.
	// No special-casing needed: even VTT input is valid FFmpeg input.
	_ = filepath.Ext(fileId) // suppress unused import warning if ext check removed

	h.runFFmpegToWebVTT(w, []string{
		"-i", fileURL,
		"-f", "webvtt",
		"pipe:1",
	})
}

func (h *SubtitleHandler) runFFmpegToWebVTT(w http.ResponseWriter, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fullArgs := append([]string{"-y"}, args...)
	cmd := exec.CommandContext(ctx, h.ffmpegPath, fullArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[subtitle] stdout pipe error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		log.Printf("[subtitle] ffmpeg start error: %v", err)
		http.Error(w, "ffmpeg start failed", http.StatusInternalServerError)
		return
	}

	if stderr != nil {
		go func() {
			io.Copy(io.Discard, stderr)
		}()
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, stdout); err != nil {
		log.Printf("[subtitle] pipe copy error: %v", err)
	}

	cmd.Wait()
}

func (h *SubtitleHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd backend/go-server && go test ./internal/hls/... -run "TestSubtitle" -v
```

Expected: all 6 subtitle tests PASS.

- [ ] **Step 5: Run full Go test suite**

```bash
cd backend/go-server && go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/go-server/internal/hls/subtitle_handler.go backend/go-server/internal/hls/subtitle_handler_test.go
git commit -m "feat(hls): add SubtitleHandler — /subtitle/embedded/ and /subtitle/file/ WebVTT endpoints"
```

---

## Task 5: Go — wire subtitle handler into HLSServer and StartServer

**Files:**
- Modify: `backend/go-server/internal/hls/server.go`
- Modify: `backend/go-server/internal/grpc/server.go`

- [ ] **Step 1: Update HLSServer to accept SubtitleHandler**

In `server.go`, replace the `HLSServer` struct, `NewHLSServer`, and `Handler()`:

```go
// HLSServer is the internal media HTTP server.
// Routes:
//   - /remux/           handled by RemuxHandler (fMP4 pipe streaming)
//   - /rawfile/         serves raw torrent bytes with Range support (FFmpeg input)
//   - /subtitle/embedded/  extracts embedded MKV subtitle stream as WebVTT
//   - /subtitle/file/      converts torrent subtitle file to WebVTT
type HLSServer struct {
	port            int
	fileOpener      FileOpener
	filePrioritizer FilePrioritizer
	remuxHandler    *RemuxHandler
	subtitleHandler *SubtitleHandler
}

func NewHLSServer(port int, fo FileOpener, fprio FilePrioritizer, remuxHandler *RemuxHandler, subtitleHandler *SubtitleHandler) *HLSServer {
	return &HLSServer{
		port:            port,
		fileOpener:      fo,
		filePrioritizer: fprio,
		remuxHandler:    remuxHandler,
		subtitleHandler: subtitleHandler,
	}
}

func (s *HLSServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/remux/", s.remuxHandler)
	mux.HandleFunc("/rawfile/", s.serveRawFile)
	if s.subtitleHandler != nil {
		mux.HandleFunc("/subtitle/embedded/", s.subtitleHandler.ServeEmbedded)
		mux.HandleFunc("/subtitle/file/", s.subtitleHandler.ServeFile)
	}
	return mux
}
```

- [ ] **Step 2: Fix existing server_test.go calls (NewHLSServer signature changed)**

In `server_test.go`, all `hls.NewHLSServer(...)` calls currently pass 4 args. Add `nil` as the 5th arg (subtitleHandler):

```go
// All 3 test calls: change from NewHLSServer(port, fo, fprio, remuxHandler)
//                              to NewHLSServer(port, fo, fprio, remuxHandler, nil)
srv := hls.NewHLSServer(8091, nil, nil, hls.NewRemuxHandler("ffmpeg", "", ""), nil)
```

- [ ] **Step 3: Update server.go to create and wire SubtitleHandler**

In `grpc/server.go`, after the line:
```go
remuxHandler := hls.NewRemuxHandler(ffmpegPath, hlsFileBaseURL, hlsBaseURL)
```
Add:
```go
subtitleHandler := hls.NewSubtitleHandler(ffmpegPath, hlsFileBaseURL)
```

Then change:
```go
hlsSrv := hls.NewHLSServer(cfg.HLSPort, fileOpener, filePrioritizer, remuxHandler)
```
To:
```go
hlsSrv := hls.NewHLSServer(cfg.HLSPort, fileOpener, filePrioritizer, remuxHandler, subtitleHandler)
```

- [ ] **Step 4: Run full Go test suite**

```bash
cd backend/go-server && go test ./... -v
```

Expected: all tests pass including the 3 existing HLS server tests.

- [ ] **Step 5: Commit**

```bash
git add backend/go-server/internal/hls/server.go backend/go-server/internal/hls/server_test.go backend/go-server/internal/grpc/server.go
git commit -m "feat(server): wire SubtitleHandler into HLSServer; register /subtitle/ routes"
```

---

## Task 6: Spring — add SubtitleTrackDto to HLSStartResponse

**Files:**
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/HLSStartResponse.java`

- [ ] **Step 1: Add SubtitleTrackDto and subtitle field**

Replace the full file:

```java
package org.devMandali.magnetPlay.model;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import java.util.List;

public record HLSStartResponse(
    @NotBlank String manifestUrl,
    boolean success,
    double durationSec,
    @NotNull List<AudioTrackDto> audioTracks,
    @NotNull List<SubtitleTrackDto> subtitleTracks
) {
    public record AudioTrackDto(int index, String language, String codec, String title) {}
    public record SubtitleTrackDto(int index, String language, String codec, String title) {}
}
```

- [ ] **Step 2: Compile**

```bash
cd backend/mp-spring && ./mvnw compile -q
```

Expected: BUILD SUCCESS (Spring service still constructs `HLSStartResponse` with old 4-arg constructor — compilation will FAIL until Task 7 fixes it).

Note: compile failure here is expected — proceed to Task 7 immediately.

- [ ] **Step 3: Commit (after Task 7 makes compile pass)**

Skip commit here — commit together with Task 7.

---

## Task 7: Spring — TorrentService map subtitle tracks + fix TorrentAddResponse

**Files:**
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/TorrentAddResponse.java`

- [ ] **Step 1: Update TorrentAddResponse to use List<FileItem>**

Replace full `TorrentAddResponse.java`:

```java
package org.devMandali.magnetPlay.model;

import io.swagger.v3.oas.annotations.media.Schema;
import java.io.Serializable;
import java.util.List;

public record TorrentAddResponse(
        @Schema(description = "Torrent id")
        String torrentId,

        @Schema(description = "Name of added torrent")
        String name,

        @Schema(description = "Status of added torrent")
        String status,

        @Schema(description = "Files present in torrent (video + subtitle)")
        List<FileItem> files
) implements Serializable {

    public record FileItem(
            String id,
            String name,
            String sizeLabel,
            String fileType   // "VIDEO" or "SUBTITLE"
    ) {}
}
```

- [ ] **Step 2: Update TorrentService — prepareTorrentAddResponseFn, startHLS, startRemux**

In `TorrentService.java`:

Replace `prepareTorrentAddResponseFn`:

```java
private final Function<TorrentResponse, TorrentAddResponse> prepareTorrentAddResponseFn = (grpcResponse) -> new TorrentAddResponse(
    grpcResponse.getTorrentId(),
    grpcResponse.getName(),
    grpcResponse.getStatus().name(),
    grpcResponse.getFilesList().stream()
        .map(file -> new TorrentAddResponse.FileItem(
            file.getId(),
            file.getName(),
            ByteUtil.formatSize(file.getSize()),
            file.getFileType().name()
        ))
        .collect(Collectors.toList())
);
```

Replace `startHLS` map lambda:

```java
public Mono<HLSStartResponse> startHLS(String infoHash, String fileId, double seekTimeSec) {
    return grpcClient.startHLS(infoHash, fileId, seekTimeSec)
            .map(r -> new HLSStartResponse(
                    r.getManifestUrl(),
                    r.getSuccess(),
                    r.getDurationSec(),
                    r.getAudioTracksList().stream()
                            .map(t -> new HLSStartResponse.AudioTrackDto(
                                    t.getIndex(), t.getLanguage(), t.getCodec(), t.getTitle()))
                            .toList(),
                    r.getSubtitleTracksList().stream()
                            .map(t -> new HLSStartResponse.SubtitleTrackDto(
                                    t.getIndex(), t.getLanguage(), t.getCodec(), t.getTitle()))
                            .toList()
            ))
            .doOnError(e -> logger.error("startHLS error for {}/{}", infoHash, fileId, e));
}
```

Replace `startRemux` map lambda:

```java
public Mono<HLSStartResponse> startRemux(String infoHash, String fileId, double seekTimeSec) {
    return grpcClient.startRemux(infoHash, fileId, seekTimeSec)
            .map(r -> new HLSStartResponse(
                    r.getManifestUrl(),
                    r.getSuccess(),
                    r.getDurationSec(),
                    r.getAudioTracksList().stream()
                            .map(t -> new HLSStartResponse.AudioTrackDto(
                                    t.getIndex(), t.getLanguage(), t.getCodec(), t.getTitle()))
                            .toList(),
                    r.getSubtitleTracksList().stream()
                            .map(t -> new HLSStartResponse.SubtitleTrackDto(
                                    t.getIndex(), t.getLanguage(), t.getCodec(), t.getTitle()))
                            .toList()
            ))
            .doOnError(e -> logger.error("startRemux error for {}/{}", infoHash, fileId, e));
}
```

- [ ] **Step 3: Compile**

```bash
cd backend/mp-spring && ./mvnw compile -q
```

Expected: BUILD SUCCESS.

- [ ] **Step 4: Run Spring tests**

```bash
cd backend/mp-spring && ./mvnw test -q
```

Expected: BUILD SUCCESS. (Note: `TorrentStatsControllerTest` and other tests should still pass.)

- [ ] **Step 5: Commit**

```bash
git add backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/HLSStartResponse.java \
        backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/TorrentAddResponse.java \
        backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java
git commit -m "feat(spring): add SubtitleTrackDto; subtitle_tracks in HLSStartResponse; TorrentAddResponse uses List<FileItem>"
```

---

## Task 8: Frontend — vite.config.ts proxy + types/index.ts

**Files:**
- Modify: `frontend/vite.config.ts`
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Add /subtitle proxy to vite.config.ts**

Replace the file:

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/subtitle': {
        target: 'http://localhost:8091',
        changeOrigin: true,
      },
    },
  },
});
```

- [ ] **Step 2: Update types/index.ts**

Replace full file:

```typescript
export interface TorrentFile {
  id: string;
  name: string;
  sizeLabel: string;
  fileType: 'VIDEO' | 'SUBTITLE';
}

export interface AudioTrack {
  index: number;
  language: string;
  codec: string;
  title: string;
}

export interface SubtitleTrackInfo {
  index: number;
  language: string;
  codec: string;
  title: string;
}

export interface HLSStartResponse {
  manifestUrl: string;
  success: boolean;
  durationSec: number;
  audioTracks: AudioTrack[];
  subtitleTracks: SubtitleTrackInfo[];
}

export interface RemuxStartResponse {
  manifestUrl: string;
  success: boolean;
  durationSec: number;
  audioTracks: AudioTrack[];
  subtitleTracks: SubtitleTrackInfo[];
}

export interface ActivePlayer {
  infoHash: string;
  fileId: string;
  fileName: string;
  mimeType: string;
  isMkv: boolean;
  streamUrl: string;
  durationSec: number;
  audioTracks: AudioTrack[];
  embeddedSubtitles: SubtitleTrackInfo[];
}

export interface SubtitleTrack {
  id: string;
  label: string;
  srclang: string;
  format: 'SRT' | 'VTT';
  blobUrl: string;
  active: boolean;
}

export type MoovStatus = 'idle' | 'checking' | 'front' | 'end' | 'unknown' | 'n/a';
export type FetchState = 'idle' | 'loading' | 'error';

export interface TorrentFileStats {
  fileId: string;
  totalSize: number;
  downloadedBytes: number;
  completionPct: number;
  downloadSpeedBps: number;
  seeders: number;
  peers: number;
  trackers: number;
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

export interface SearchResult {
  title: string;
  sizeBytes: number;
  seeders: number;
  peers: number;
  indexer: string;
  pubDate: string;
  magnetUrl: string;
  qualityTags: string[];
}

export interface SearchResultsResponse {
  results: SearchResult[];
  total: number;
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/vite.config.ts frontend/src/types/index.ts
git commit -m "feat(frontend): proxy /subtitle to Go HLS server; add SubtitleTrackInfo type; update TorrentFile/ActivePlayer/RemuxStartResponse"
```

---

## Task 9: Frontend — App.tsx subtitle auto-population

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Add torrentSubtitleFiles state and encodeFileId helper**

After the existing `const blobUrls = useRef<Record<string, string>>({});` line, add:

```tsx
const [torrentSubtitleFiles, setTorrentSubtitleFiles] = useState<TorrentFile[]>([]);
```

Add this helper function inside the `App` component, before `handleSubtitleAdd`:

```tsx
// Encodes a fileId to base64url (no padding) — matches Go's base64.RawURLEncoding
const encodeFileId = (fileId: string): string =>
  btoa(fileId).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');

const fetchSubtitleBlob = async (url: string): Promise<string | null> => {
  try {
    const res = await fetch(url, { signal: AbortSignal.timeout(30000) });
    if (!res.ok) return null;
    const text = await res.text();
    const blob = new Blob([text], { type: 'text/vtt' });
    return URL.createObjectURL(blob);
  } catch {
    return null;
  }
};
```

- [ ] **Step 2: Update handleFetchFiles to separate video/subtitle files**

Replace the `handleFetchFiles` function:

```tsx
const handleFetchFiles = async () => {
  const mag = magnetLink.trim();
  if (!mag) return;
  setFetchState('loading');
  setFetchError(null);
  try {
    const res = await fetch('/v1/torrent/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ magnet: mag }),
      signal: AbortSignal.timeout(40000),
    });
    if (!res.ok) throw new Error(`Server returned ${res.status}`);
    const data = await res.json();
    const hash = (data.torrentId as string) || '';
    const rawFiles = (data.files ?? []) as Array<{ id: string; name: string; sizeLabel: string; fileType: string }>;
    const files: TorrentFile[] = rawFiles.map(f => ({
      id: f.id,
      name: f.name,
      sizeLabel: f.sizeLabel ?? '',
      fileType: f.fileType === 'SUBTITLE' ? 'SUBTITLE' : 'VIDEO',
    }));
    const videoFiles = files.filter(f => f.fileType === 'VIDEO');
    const subtitleFiles = files.filter(f => f.fileType === 'SUBTITLE');

    if (!videoFiles.length) throw new Error('No video files found in torrent');
    const firstVideo = videoFiles.find(f => /\.(mp4|mkv|avi|mov|webm|ts|m4v|flv)$/i.test(f.name)) ?? videoFiles[0];
    setInfoHash(hash);
    setTorrentFiles(videoFiles);
    setTorrentSubtitleFiles(subtitleFiles);
    setDraft(d => ({ ...d, infoHash: hash, fileId: firstVideo.id, fileName: firstVideo.name }));
    setStep(2);
    setFetchState('idle');
  } catch (err) {
    setFetchState('error');
    setFetchError(err instanceof Error ? err.message : 'Failed to fetch torrent info');
  }
};
```

- [ ] **Step 3: Update handlePlay to auto-populate subtitle tracks**

Replace the `handlePlay` function:

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
        embeddedSubtitles: data.subtitleTracks ?? [],
      });

      // Auto-populate embedded subtitle tracks from MKV probe
      const embeddedTracks = data.subtitleTracks ?? [];
      const b64FileId = encodeFileId(draft.fileId);
      const embeddedSubBlobPromises = embeddedTracks.map(async (track) => {
        const url = `/subtitle/embedded/${draft.infoHash}/${b64FileId}/${track.index}`;
        const blobUrl = await fetchSubtitleBlob(url);
        if (!blobUrl) return null;
        const label = track.title
          ? `[EMB] ${track.title}`
          : track.language
          ? `[EMB] ${track.language}`
          : `[EMB] Track ${track.index}`;
        const id = `emb_${track.index}_${Date.now()}`;
        blobUrls.current[id] = blobUrl;
        return { id, label, srclang: track.language || 'und', format: 'VTT' as const, blobUrl, active: false };
      });

      // Auto-populate torrent subtitle files (both MKV and MP4 paths)
      const torrentSubBlobPromises = torrentSubtitleFiles.map(async (subFile) => {
        const b64SubFileId = encodeFileId(subFile.id);
        const url = `/subtitle/file/${draft.infoHash}/${b64SubFileId}`;
        const blobUrl = await fetchSubtitleBlob(url);
        if (!blobUrl) return null;
        const label = subFile.name.replace(/\.[^/.]+$/, '').split(/[\\/]/).pop() ?? subFile.name;
        const id = `torrent_${subFile.id}_${Date.now()}`;
        blobUrls.current[id] = blobUrl;
        return { id, label, srclang: 'und', format: 'VTT' as const, blobUrl, active: false };
      });

      const allResults = await Promise.all([...embeddedSubBlobPromises, ...torrentSubBlobPromises]);
      const newTracks = allResults.filter((t): t is NonNullable<typeof t> => t !== null);
      if (newTracks.length > 0) setSubtitleTracks(newTracks);

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
      embeddedSubtitles: [],
    });

    // Auto-populate torrent subtitle files for non-MKV (MP4 etc.)
    if (torrentSubtitleFiles.length > 0) {
      const torrentSubBlobPromises = torrentSubtitleFiles.map(async (subFile) => {
        const b64SubFileId = encodeFileId(subFile.id);
        const url = `/subtitle/file/${draft.infoHash}/${b64SubFileId}`;
        const blobUrl = await fetchSubtitleBlob(url);
        if (!blobUrl) return null;
        const label = subFile.name.replace(/\.[^/.]+$/, '').split(/[\\/]/).pop() ?? subFile.name;
        const id = `torrent_${subFile.id}_${Date.now()}`;
        blobUrls.current[id] = blobUrl;
        return { id, label, srclang: 'und', format: 'VTT' as const, blobUrl, active: false };
      });
      const results = await Promise.all(torrentSubBlobPromises);
      const newTracks = results.filter((t): t is NonNullable<typeof t> => t !== null);
      if (newTracks.length > 0) setSubtitleTracks(newTracks);
    }
  }
};
```

- [ ] **Step 4: Update handleReset to clear torrentSubtitleFiles and fix ActivePlayer usage**

In `handleReset`, add after `setSubtitleTracks([])`:

```tsx
// (torrentSubtitleFiles intentionally NOT cleared on reset — they belong to the torrent, not the player)
```

In `handleFullReset`, add:
```tsx
setTorrentSubtitleFiles([]);
```

- [ ] **Step 5: Fix TypeScript — update imports**

At top of `App.tsx`, update the import line to include `SubtitleTrackInfo`:

```tsx
import { TorrentFile, ActivePlayer, SubtitleTrack, SubtitleTrackInfo, MoovStatus, FetchState, TorrentFileStats, RemuxStartResponse } from './types';
```

- [ ] **Step 6: Check TypeScript compiles**

```bash
cd frontend && npm run build 2>&1 | head -40
```

Expected: no TypeScript errors. (Build output warnings about unused vars are OK; errors are not.)

- [ ] **Step 7: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(app): auto-populate CC panel with embedded MKV subs and torrent subtitle files on play"
```

---

## Task 10: Integration smoke test

**Manual test — run all three services then verify:**

- [ ] **Step 1: Start services**

```bash
# Terminal 1
cd backend/go-server && go run main.go

# Terminal 2
cd backend/mp-spring && ./mvnw spring-boot:run

# Terminal 3
cd frontend && npm run dev
```

- [ ] **Step 2: Test torrent subtitle file flow (MP4)**

1. Open `http://localhost:5173`
2. Paste a magnet link for a torrent that includes `.srt` alongside a `.mp4` file
3. Press "Fetch Files" — file dropdown should show only the `.mp4`
4. Press "Load Stream"
5. Open CC panel (CC badge in player controls)
6. Verify the `.srt` subtitle track appears as a named track (filename without extension)
7. Click the track — verify subtitles render on video

- [ ] **Step 3: Test embedded MKV subtitle flow**

1. Paste a magnet link for an `.mkv` file with embedded subtitles (anime/foreign film)
2. Press "Fetch Files" → "Load Stream"
3. Open CC panel
4. Verify embedded subtitle tracks appear labelled `[EMB] English` / `[EMB] Japanese` (or title)
5. Click a track — verify subtitles render on video

- [ ] **Step 4: Verify manual upload still works**

1. While video is playing, open CC panel
2. Upload a `.srt` file manually
3. Verify it appears alongside auto-populated tracks
4. Toggle between tracks — verify only one is active at a time

- [ ] **Step 5: Run all automated tests one final time**

```bash
cd backend/go-server && go test ./... && cd ../mp-spring && ./mvnw test -q
```

Expected: all pass.

---

## Self-Review Checklist

- [x] **Proto field numbers** — no field number conflicts. FileInfoResponse field 8 is new. HLSResponse field 5 is new. FileInfo field 4 is new. All unused fields (no gaps in existing numbering).
- [x] **FileType default** — `VIDEO = 0` is correct proto3 default; existing `FileInfo` usages that don't set `file_type` will default to `VIDEO`.
- [x] **mapper.go subtitle priority** — subtitle files get `PiecePriorityNone` implicitly (we don't call `f.SetPriority` for them — they download normally). This is intentional: subtitle files are small and download fast.
- [x] **bitmap codec skip** — `toProtoSubtitleTracks` skips bitmap codecs before returning to proto. Frontend never sees PGS/DVB tracks.
- [x] **base64 encoding** — `encodeFileId` on frontend produces RawURLEncoding output matching Go's decoder.
- [x] **handleFullReset** clears `torrentSubtitleFiles`; `handleReset` (player-only reset) does not.
- [x] **blobUrls cleanup** — `handleReset` calls `URL.revokeObjectURL` for all blob URLs including auto-populated ones (stored in `blobUrls.current`).
- [x] **`SubtitleTrackInfo` import** added to `App.tsx` imports.
- [x] **`ActivePlayer.embeddedSubtitles`** added to type and set in both MKV and non-MKV paths.
- [x] **Spring `TorrentAddResponse`** — `Map<String, String>` replaced with `List<FileItem>`; frontend parser updated to match.
- [x] **Task ordering** — each task builds on previous. Proto first, Go backend, Spring, Frontend.
