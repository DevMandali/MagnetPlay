# MKV HLS Streaming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stream MKV torrents in the browser by transcoding to HLS via FFmpeg in the Go sidecar, with full seekbar, multi-audio track switching, and zero changes to the existing MP4 path.

**Architecture:** Go sidecar spawns FFmpeg per MKV file, writes HLS segments to disk, serves them from a new HTTP server on port 8091. Spring Boot adds one trigger endpoint (start/stop). Frontend detects `.mkv` by file name, calls `/hls/start`, and feeds the manifest URL to VHS (already in project).

**Tech Stack:** Go `os/exec` + FFmpeg, Go `net/http` (new port 8091), gRPC new RPCs, Spring Boot WebFlux reactive, video.js VHS (existing), React `useEffect`/`useRef`.

**Spec:** `docs/superpowers/specs/2026-04-24-mkv-hls-streaming-design.md`

> **Code Review:** Code quality review runs AFTER all 16 tasks are complete — not between tasks. Do not pause for review mid-implementation.

---

## File Map

### New files
| Path | Responsibility |
|---|---|
| `backend/go-server/internal/ffmpeg/binary.go` | `EnsureFFmpeg()`, `EnsureFFprobe()`, `FFmpegBinaryName()`, `FFprobeBinaryName()` — download from BtbN/FFmpeg-Builds if not present |
| `backend/go-server/internal/ffmpeg/binary_test.go` | Unit tests for binary path resolution |
| `backend/go-server/internal/hls/transcoder.go` | `HLSTranscoder` — per-fileId FFmpeg lifecycle, audio fallback, segment polling |
| `backend/go-server/internal/hls/transcoder_test.go` | Unit tests for arg building + concurrency |
| `backend/go-server/internal/hls/server.go` | `HLSServer` — `net/http` on `:8091`, serves manifest + `.ts` segments with CORS |
| `backend/go-server/internal/hls/server_test.go` | Unit tests for 404 + CORS headers |
| `backend/go-server/internal/hls/transcoder_integration_test.go` | Integration tests (skip if `ffmpeg` absent) |
| `backend/go-server/internal/hls/testdata/sample_10s.mkv` | 10s H.264+AAC fixture for integration test |
| `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/HLSStartResponse.java` | DTO: `manifestUrl`, `success`, `durationSec`, `audioTracks` |
| `backend/mp-spring/src/test/java/org/devMandali/magnetPlay/controller/TorrentHLSControllerTest.java` | Spring controller tests |

### Modified files
| Path | Change |
|---|---|
| `backend/proto/torrent.proto` | Add `StartHLS`, `StopHLS` RPCs; `HLSRequest`, `HLSResponse` (with duration + tracks), `AudioTrack` messages; extend `FileInfoResponse` fields 6–7 |
| `backend/go-server/proto/` (generated) | Regenerated after proto change |
| `backend/mp-spring/` (generated Java) | Regenerated via `mvn compile` |
| `backend/go-server/config/config.go` | Add `FFmpegPath`, `FFprobePath`, `HLSPort`, `HLSDir` with defaults |
| `backend/go-server/internal/torrent/service.go` | Add `transcoder`, `ffprobePath`, `hlsBaseURL`, `probeCache`, `probeMu` fields; update `NewTorrentService`; extend `GetFileInfo` for MKV; add `StartHLS`/`StopHLS` handlers |
| `backend/go-server/internal/grpc/server.go` | Call `EnsureFFmpeg`/`EnsureFFprobe`, wire `HLSTranscoder`, start `HLSServer` |
| `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/TorrentGrpcClient.java` | Add `startHLS()`, `stopHLS()` |
| `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java` | Add `startHLS()`, `stopHLS()` |
| `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/TorrentController.java` | Add `POST /hls/{hash}/start`, `DELETE /hls/{hash}/stop` |
| `frontend/src/types/index.ts` | Add `AudioTrack`, `HLSStartResponse`; extend `ActivePlayer` |
| `frontend/src/App.tsx` | MKV detection in `handlePlay`; call `/hls/start`; pass `durationSec`, `audioTracks`, `isMkv` to VideoPlayer |
| `frontend/src/components/VideoPlayer.tsx` | MKV branch: VHS source, `player.duration()`, seek debounce, stop on unmount |
| `frontend/src/lib/videoSetup.ts` | Add `registerAudioTrackButton()` |

---

## Task 1: Proto — add HLS RPCs and messages

**Files:**
- Modify: `backend/proto/torrent.proto`

- [ ] **Step 1: Add new messages and RPCs to proto**

In `backend/proto/torrent.proto`, add after the `DeleteTorrent` RPC in the service block and add new message types at the end of the file:

```proto
// In service TorrentService { ... } block, after DeleteTorrent:
rpc StartHLS (HLSRequest) returns (HLSResponse);
rpc StopHLS  (HLSRequest) returns (HLSResponse);
```

Add `AudioTrack` message and `HLSRequest`/`HLSResponse` at the bottom of the file:

```proto
// ── HLS Streaming ────────────────────────────────────────────────────────────
message AudioTrack {
  int32  index    = 1;
  string language = 2;
  string codec    = 3;
  string title    = 4;
}

message HLSRequest {
  string info_hash     = 1;
  string file_id       = 2;
  double seek_time_sec = 3;
}

message HLSResponse {
  string               manifest_url  = 1;
  bool                 success       = 2;
  double               duration_sec  = 3;
  repeated AudioTrack  audio_tracks  = 4;
}
```

Extend `FileInfoResponse` (add after field 5):

```proto
message FileInfoResponse {
  string file_name   = 1;
  string file_path   = 2;
  int64  total_size  = 3;
  string mime_type   = 4;
  bool   is_ready    = 5;
  double               duration_sec  = 6;
  repeated AudioTrack  audio_tracks  = 7;
}
```

- [ ] **Step 2: Regenerate Go proto**

```bash
cd backend
make -f MakeFile proto
```

Expected: `backend/go-server/proto/torrent.pb.go` and `torrent_grpc.pb.go` regenerated without errors. New methods `StartHLS` and `StopHLS` appear in `TorrentServiceServer` interface.

- [ ] **Step 3: Regenerate Java proto**

```bash
cd backend/mp-spring
./mvnw compile -q
```

Expected: Compiles successfully. New classes `HLSRequest`, `HLSResponse`, `AudioTrack` generated under `target/generated-sources/`.

- [ ] **Step 4: Add `UnimplementedTorrentServiceServer` stubs**

In `backend/go-server/internal/torrent/service.go`, the `TorrentService` embeds `pb.UnimplementedTorrentServiceServer`. The new `StartHLS` and `StopHLS` methods are auto-stubbed by the embedded struct — no changes needed yet. Verify compilation:

```bash
cd backend/go-server
go build ./...
```

Expected: Builds successfully (stubs satisfy the interface).

- [ ] **Step 5: Commit**

```bash
git add backend/proto/torrent.proto backend/go-server/proto/ backend/mp-spring/src/
git commit -m "feat(proto): add HLS RPCs, AudioTrack, extend FileInfoResponse"
```

---

## Task 2: Go — config fields for FFmpeg and HLS

**Files:**
- Modify: `backend/go-server/config/config.go`

- [ ] **Step 1: Add new config fields**

Replace the entire `config.go` content:

```go
package config

import "time"

type ProwlarrConfig struct {
	DataDir      string
	BinDir       string
	Port         int
	SeedIndexers bool
}

type Config struct {
	GRPCPort        int
	DataDir         string
	MetadataTimeout time.Duration
	Prowlarr        ProwlarrConfig
	FFmpegPath      string
	FFprobePath     string
	HLSPort         int
	HLSDir          string
}

func Default() Config {
	return Config{
		GRPCPort:        50051,
		DataDir:         "./downloads",
		MetadataTimeout: 60 * time.Second,
		Prowlarr: ProwlarrConfig{
			DataDir:      "./prowlarr-data",
			BinDir:       "./prowlarr",
			Port:         9696,
			SeedIndexers: true,
		},
		HLSPort: 8091,
		HLSDir:  "./downloads",
	}
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend/go-server
go build ./...
```

Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add backend/go-server/config/config.go
git commit -m "feat(config): add FFmpegPath, FFprobePath, HLSPort, HLSDir"
```

---

## Task 3: Go — FFmpeg binary manager (TDD)

**Files:**
- Create: `backend/go-server/internal/ffmpeg/binary.go`
- Create: `backend/go-server/internal/ffmpeg/binary_test.go`

- [ ] **Step 1: Write failing test**

Create `backend/go-server/internal/ffmpeg/binary_test.go`:

```go
package ffmpeg_test

import (
	"os"
	"path/filepath"
	"testing"

	"server/internal/ffmpeg"
)

func TestFFmpegBinaryName(t *testing.T) {
	name := ffmpeg.FFmpegBinaryName()
	if name == "" {
		t.Fatal("FFmpegBinaryName() returned empty string")
	}
}

func TestFFprobeBinaryName(t *testing.T) {
	name := ffmpeg.FFprobeBinaryName()
	if name == "" {
		t.Fatal("FFprobeBinaryName() returned empty string")
	}
}

func TestConfigPathOverride_FFmpeg(t *testing.T) {
	tmp := t.TempDir()
	fakeBin := filepath.Join(tmp, "ffmpeg_fake")
	if err := os.WriteFile(fakeBin, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := ffmpeg.ResolvePath(fakeBin, tmp)
	if err != nil {
		t.Fatalf("ResolvePath returned error: %v", err)
	}
	if got != fakeBin {
		t.Fatalf("expected %s, got %s", fakeBin, got)
	}
}

func TestResolvePath_UsesExistingInBinDir(t *testing.T) {
	tmp := t.TempDir()
	binName := ffmpeg.FFmpegBinaryName()
	fakeBin := filepath.Join(tmp, binName)
	if err := os.WriteFile(fakeBin, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := ffmpeg.ResolvePath("", tmp)
	if err != nil {
		t.Fatalf("ResolvePath returned error: %v", err)
	}
	if got != fakeBin {
		t.Fatalf("expected %s, got %s", fakeBin, got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/go-server
go test ./internal/ffmpeg/... -v
```

Expected: FAIL — `package server/internal/ffmpeg` not found.

- [ ] **Step 3: Implement binary manager**

Create `backend/go-server/internal/ffmpeg/binary.go`:

```go
package ffmpeg

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const btbnReleasesURL = "https://api.github.com/repos/BtbN/FFmpeg-Builds/releases/latest"

var (
	apiClient = &http.Client{Timeout: 30 * time.Second}
	dlClient  = &http.Client{Timeout: 20 * time.Minute}
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func FFmpegBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

func FFprobeBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffprobe.exe"
	}
	return "ffprobe"
}

// ResolvePath returns the path to use for the binary.
// If configPath is set and the file exists, use it.
// Otherwise check binDir for the binary; if present, return that path.
// If neither, return "" and an error (caller should call EnsureFFmpeg/EnsureFFprobe first).
func ResolvePath(configPath, binDir string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		return "", fmt.Errorf("configured binary not found: %s", configPath)
	}
	candidate := filepath.Join(binDir, FFmpegBinaryName())
	if found := findInDir(binDir, FFmpegBinaryName()); found != "" {
		return found, nil
	}
	_ = candidate
	return "", fmt.Errorf("ffmpeg not found in %s", binDir)
}

func ResolveFFprobePath(configPath, binDir string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		return "", fmt.Errorf("configured ffprobe not found: %s", configPath)
	}
	if found := findInDir(binDir, FFprobeBinaryName()); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("ffprobe not found in %s", binDir)
}

// EnsureFFmpeg downloads FFmpeg to binDir if not already present.
// Returns the path to the ffmpeg binary.
func EnsureFFmpeg(configPath, binDir string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}
	if found := findInDir(binDir, FFmpegBinaryName()); found != "" {
		return found, nil
	}
	if err := downloadFFmpeg(binDir); err != nil {
		return "", err
	}
	found := findInDir(binDir, FFmpegBinaryName())
	if found == "" {
		return "", fmt.Errorf("ffmpeg not found in %s after download", binDir)
	}
	return found, nil
}

// EnsureFFprobe downloads ffprobe to binDir if not already present.
// Returns the path to the ffprobe binary.
func EnsureFFprobe(configPath, binDir string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}
	if found := findInDir(binDir, FFprobeBinaryName()); found != "" {
		return found, nil
	}
	// ffprobe ships in the same bundle as ffmpeg — download it together
	if err := downloadFFmpeg(binDir); err != nil {
		return "", err
	}
	found := findInDir(binDir, FFprobeBinaryName())
	if found == "" {
		return "", fmt.Errorf("ffprobe not found in %s after download", binDir)
	}
	return found, nil
}

func findInDir(dir, name string) string {
	var found string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func platformAssetSuffix() string {
	switch runtime.GOOS {
	case "windows":
		return "win64-gpl.zip"
	case "linux":
		return "linux64-gpl.tar.xz"
	default:
		return ""
	}
}

func downloadFFmpeg(binDir string) error {
	suffix := platformAssetSuffix()
	if suffix == "" {
		return fmt.Errorf("unsupported platform %s/%s — set FFmpegPath in config", runtime.GOOS, runtime.GOARCH)
	}

	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetch FFmpeg release: %w", err)
	}

	var assetURL string
	for _, a := range rel.Assets {
		if strings.Contains(a.Name, "master-latest") && strings.HasSuffix(a.Name, suffix) {
			assetURL = a.BrowserDownloadURL
			break
		}
	}
	if assetURL == "" {
		return fmt.Errorf("no FFmpeg asset matching suffix %q in release %s", suffix, rel.TagName)
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	ext := ".zip"
	if strings.HasSuffix(assetURL, ".tar.xz") {
		ext = ".tar.xz"
	}
	tmp, err := os.CreateTemp("", "ffmpeg-*"+ext)
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	fmt.Printf("[ffmpeg] downloading %s...\n", rel.TagName)
	if err := downloadToFile(assetURL, tmp); err != nil {
		return fmt.Errorf("download FFmpeg: %w", err)
	}

	if ext == ".zip" {
		return extractZip(tmp.Name(), binDir)
	}
	return fmt.Errorf("tar.xz extraction not implemented; set FFmpegPath in config or extract manually to %s", binDir)
}

func fetchLatestRelease() (*githubRelease, error) {
	req, _ := http.NewRequest("GET", btbnReleasesURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MagnetPlay/1.0")
	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	var rel githubRelease
	return &rel, json.NewDecoder(resp.Body).Decode(&rel)
}

func downloadToFile(url string, dest *os.File) error {
	resp, err := dlClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d downloading FFmpeg", resp.StatusCode)
	}
	_, err = io.Copy(dest, resp.Body)
	return err
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend/go-server
go test ./internal/ffmpeg/... -v
```

Expected: All 4 tests PASS.

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add backend/go-server/internal/ffmpeg/
git commit -m "feat(ffmpeg): binary manager with auto-download from BtbN/FFmpeg-Builds"
```

---

## Task 4: Go — HLS transcoder (TDD)

**Files:**
- Create: `backend/go-server/internal/hls/transcoder.go`
- Create: `backend/go-server/internal/hls/transcoder_test.go`

- [ ] **Step 1: Write failing unit tests**

Create `backend/go-server/internal/hls/transcoder_test.go`:

```go
package hls_test

import (
	"path/filepath"
	"strings"
	"testing"

	"server/internal/hls"
)

func TestBuildFFmpegArgs_SeekZero(t *testing.T) {
	tr := hls.NewHLSTranscoder("ffmpeg", "./downloads")
	args := tr.BuildArgs("/seg/dir", 0.0, "copy")

	for i, a := range args {
		if a == "-ss" {
			t.Fatalf("expected no -ss flag for seekSec=0, found at index %d", i)
		}
	}
	if !containsSeq(args, []string{"-i", "pipe:0"}) {
		t.Error("expected -i pipe:0 in args")
	}
	if !containsSeq(args, []string{"-c:a", "copy"}) {
		t.Error("expected -c:a copy in args")
	}
	if !containsSeq(args, []string{"-c:v", "copy"}) {
		t.Error("expected -c:v copy in args")
	}
	if !contains(args, "playlist.m3u8") {
		t.Error("expected playlist.m3u8 in args")
	}
}

func TestBuildFFmpegArgs_SeekNonZero(t *testing.T) {
	tr := hls.NewHLSTranscoder("ffmpeg", "./downloads")
	args := tr.BuildArgs("/seg/dir", 600.0, "copy")

	ssIdx := -1
	for i, a := range args {
		if a == "-ss" {
			ssIdx = i
			break
		}
	}
	if ssIdx == -1 {
		t.Fatal("expected -ss flag for seekSec=600")
	}
	if ssIdx+1 >= len(args) || args[ssIdx+1] != "600.000" {
		t.Fatalf("expected -ss 600.000, got %v", args[ssIdx:ssIdx+2])
	}

	// -ss must come before -i
	iIdx := -1
	for i, a := range args {
		if a == "-i" {
			iIdx = i
			break
		}
	}
	if ssIdx >= iIdx {
		t.Error("-ss must appear before -i")
	}
}

func TestBuildFFmpegArgs_AudioFallback(t *testing.T) {
	tr := hls.NewHLSTranscoder("ffmpeg", "./downloads")
	args := tr.BuildArgs("/seg/dir", 0.0, "aac")
	if !containsSeq(args, []string{"-c:a", "aac"}) {
		t.Error("expected -c:a aac in fallback args")
	}
}

func TestBuildFFmpegArgs_MapsAllAudio(t *testing.T) {
	tr := hls.NewHLSTranscoder("ffmpeg", "./downloads")
	args := tr.BuildArgs("/seg/dir", 0.0, "copy")
	if !containsSeq(args, []string{"-map", "0:v:0"}) {
		t.Error("expected -map 0:v:0")
	}
	if !containsSeq(args, []string{"-map", "0:a"}) {
		t.Error("expected -map 0:a")
	}
}

func TestBuildFFmpegArgs_SegmentFilenameContainsDir(t *testing.T) {
	tr := hls.NewHLSTranscoder("ffmpeg", "./downloads")
	args := tr.BuildArgs("/my/seg/dir", 0.0, "copy")
	found := false
	for _, a := range args {
		if strings.Contains(a, filepath.Join("/my/seg/dir", "seg%03d.ts")) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected segment filename pattern in args, got: %v", args)
	}
}

func TestIsAudioCodecError(t *testing.T) {
	cases := []struct {
		stderr string
		want   bool
	}{
		{"Invalid data found when processing input", true},
		{"codec not currently supported in container", true},
		{"frame=  42 fps=30 q=-1.0 size=1024kB", false},
	}
	for _, tc := range cases {
		got := hls.IsAudioCodecError(tc.stderr)
		if got != tc.want {
			t.Errorf("IsAudioCodecError(%q) = %v, want %v", tc.stderr, got, tc.want)
		}
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if strings.Contains(v, s) {
			return true
		}
	}
	return false
}

func containsSeq(slice []string, seq []string) bool {
	for i := 0; i <= len(slice)-len(seq); i++ {
		match := true
		for j, s := range seq {
			if slice[i+j] != s {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/go-server
go test ./internal/hls/... -v
```

Expected: FAIL — `package server/internal/hls` not found.

- [ ] **Step 3: Implement HLS transcoder**

Create `backend/go-server/internal/hls/transcoder.go`:

```go
package hls

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type HLSTranscoder struct {
	ffmpegPath string
	hlsDir     string
	mu         sync.Mutex
	jobs       map[string]*hlsJob
	baseURL    string
}

type hlsJob struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	segDir string
	reader io.Closer // stored so Stop() can release the torrent reader
}

func NewHLSTranscoder(ffmpegPath, hlsDir, baseURL string) *HLSTranscoder {
	return &HLSTranscoder{
		ffmpegPath: ffmpegPath,
		hlsDir:     hlsDir,
		baseURL:    baseURL,
		jobs:       make(map[string]*hlsJob),
	}
}

// BuildArgs constructs the FFmpeg argument slice for HLS output.
// Exported for testing. segDir must be an absolute path.
func (t *HLSTranscoder) BuildArgs(segDir string, seekSec float64, audioCodec string) []string {
	args := []string{"-y"}
	if seekSec > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", seekSec))
	}
	args = append(args,
		"-i", "pipe:0",
		"-map", "0:v:0",
		"-map", "0:a",
		"-c:v", "copy",
		"-c:a", audioCodec,
		"-f", "hls",
		"-hls_time", "4",
		"-hls_list_size", "0",
		"-hls_flags", "delete_segments",
		"-hls_segment_filename", filepath.Join(segDir, "seg%03d.ts"),
		filepath.Join(segDir, "playlist.m3u8"),
	)
	return args
}

// IsAudioCodecError returns true if stderr indicates an audio codec incompatibility.
// Exported for testing.
func IsAudioCodecError(stderr string) bool {
	return strings.Contains(stderr, "Invalid data found") ||
		strings.Contains(stderr, "codec not currently supported in container")
}

// Start kills any existing FFmpeg job for fileId, spawns a new one,
// waits for ≥2 segments, and returns the manifest URL.
// The transcoder takes ownership of reader and closes it on Stop().
// reader must be io.ReadSeekCloser — anacrolix lt.Reader satisfies this.
func (t *HLSTranscoder) Start(infoHash, fileId string, seekSec float64, reader io.ReadSeekCloser) (string, error) {
	t.mu.Lock()
	if existing, ok := t.jobs[fileId]; ok {
		existing.cancel()
		existing.cmd.Wait() // nolint — best effort
		os.RemoveAll(existing.segDir)
		delete(t.jobs, fileId)
	}
	t.mu.Unlock()

	segDir := filepath.Join(t.hlsDir, infoHash, "hls", fileId)
	if err := os.MkdirAll(segDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", segDir, err)
	}
	// Remove stale segments from previous seek
	cleanSegments(segDir)

	// Seek reader to start (required for audio-fallback retry)
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("reader seek: %w", err)
	}

	manifestURL, err := t.spawnAndWait(infoHash, fileId, segDir, seekSec, "copy", reader)
	if err != nil {
		// Audio fallback: seek reader back to start and retry with AAC
		if _, seekErr := reader.Seek(0, io.SeekStart); seekErr != nil {
			return "", fmt.Errorf("reader seek for audio fallback: %w", seekErr)
		}
		cleanSegments(segDir)
		log.Printf("[hls] audio copy failed for %s/%s, retrying with aac: %v", infoHash, fileId, err)
		manifestURL, err = t.spawnAndWait(infoHash, fileId, segDir, seekSec, "aac", reader)
		if err != nil {
			return "", fmt.Errorf("HLS failed (copy+aac): %w", err)
		}
	}
	return manifestURL, nil
}

func (t *HLSTranscoder) spawnAndWait(infoHash, fileId, segDir string, seekSec float64, audioCodec string, reader io.Reader) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())

	args := t.BuildArgs(segDir, seekSec, audioCodec)
	cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)
	cmd.Stdin = reader

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return "", fmt.Errorf("ffmpeg start: %w", err)
	}

	job := &hlsJob{cmd: cmd, cancel: cancel, segDir: segDir, reader: reader}
	t.mu.Lock()
	t.jobs[fileId] = job
	t.mu.Unlock()

	// Collect stderr asynchronously (thread-safe via channel)
	stderrCh := make(chan string, 256)
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			stderrCh <- scanner.Text()
		}
		close(stderrCh)
	}()

	// Wait for FFmpeg exit asynchronously
	exitCh := make(chan error, 1)
	go func() { exitCh <- cmd.Wait() }()

	deadline := time.Now().Add(30 * time.Second)
	var stderrLines []string

	for time.Now().Before(deadline) {
		// Drain stderr lines
	drainLoop:
		for {
			select {
			case line, ok := <-stderrCh:
				if !ok {
					break drainLoop
				}
				stderrLines = append(stderrLines, line)
			default:
				break drainLoop
			}
		}

		// Check if FFmpeg exited early
		select {
		case exitErr := <-exitCh:
			stderr := strings.Join(stderrLines, "\n")
			if audioCodec == "copy" && IsAudioCodecError(stderr) {
				cancel()
				return "", fmt.Errorf("audio codec error (will retry with aac): %s", stderr)
			}
			cancel()
			return "", fmt.Errorf("ffmpeg exited early: %v — stderr: %s", exitErr, stderr)
		default:
		}

		n := countSegments(segDir)
		if n >= 2 {
			log.Printf("[hls] %s/%s ready: %d segments (codec=%s, seek=%.1fs)", infoHash, fileId, n, audioCodec, seekSec)
			return t.manifestURL(infoHash, fileId), nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	cancel()
	return "", fmt.Errorf("timeout waiting for HLS segments after 30s")
}

// Stop kills the FFmpeg job for fileId, closes the torrent reader, and removes its segment directory.
func (t *HLSTranscoder) Stop(fileId string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if job, ok := t.jobs[fileId]; ok {
		job.cancel()
		job.cmd.Wait()    // nolint — best effort
		job.reader.Close() // release anacrolix torrent reader
		os.RemoveAll(job.segDir)
		delete(t.jobs, fileId)
		log.Printf("[hls] stopped and cleaned up fileId=%s", fileId)
	}
}

// StopAll kills all running FFmpeg jobs (called on shutdown).
func (t *HLSTranscoder) StopAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for fileId, job := range t.jobs {
		job.cancel()
		job.cmd.Wait() // nolint
		os.RemoveAll(job.segDir)
		delete(t.jobs, fileId)
	}
}

func (t *HLSTranscoder) manifestURL(infoHash, fileId string) string {
	return fmt.Sprintf("%s/hls/%s/%s/playlist.m3u8", t.baseURL, infoHash, fileId)
}

func countSegments(segDir string) int {
	entries, err := os.ReadDir(segDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ts") {
			count++
		}
	}
	return count
}

func cleanSegments(segDir string) {
	entries, _ := os.ReadDir(segDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ts") || e.Name() == "playlist.m3u8" {
			os.Remove(filepath.Join(segDir, e.Name()))
		}
	}
}
```

- [ ] **Step 4: Run unit tests**

```bash
cd backend/go-server
go test ./internal/hls/... -run "TestBuild|TestIsAudio" -v
```

Expected: All 6 unit tests PASS.

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add backend/go-server/internal/hls/transcoder.go backend/go-server/internal/hls/transcoder_test.go
git commit -m "feat(hls): transcoder with audio fallback and segment polling"
```

---

## Task 5: Go — HLS HTTP server (TDD)

**Files:**
- Create: `backend/go-server/internal/hls/server.go`
- Create: `backend/go-server/internal/hls/server_test.go`

- [ ] **Step 1: Write failing tests**

Create `backend/go-server/internal/hls/server_test.go`:

```go
package hls_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"server/internal/hls"
)

func TestManifestHandler_NotFound(t *testing.T) {
	srv := hls.NewHLSServer(8091, t.TempDir())
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/hls/abc123/file001/playlist.m3u8", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSegmentHandler_NotFound(t *testing.T) {
	srv := hls.NewHLSServer(8091, t.TempDir())
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/hls/abc123/file001/seg000.ts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestManifestHandler_ServesCORS(t *testing.T) {
	tmp := t.TempDir()
	srv := hls.NewHLSServer(8091, tmp)
	handler := srv.Handler()

	// Create a fake manifest
	segDir := filepath.Join(tmp, "abc123", "hls", "file001")
	os.MkdirAll(segDir, 0755)
	os.WriteFile(filepath.Join(segDir, "playlist.m3u8"), []byte("#EXTM3U\n"), 0644)

	req := httptest.NewRequest("GET", "/hls/abc123/file001/playlist.m3u8", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	cors := w.Header().Get("Access-Control-Allow-Origin")
	if cors != "*" {
		t.Fatalf("expected CORS *, got %q", cors)
	}
}

func TestSegmentHandler_ServesFile(t *testing.T) {
	tmp := t.TempDir()
	srv := hls.NewHLSServer(8091, tmp)
	handler := srv.Handler()

	segDir := filepath.Join(tmp, "abc123", "hls", "file001")
	os.MkdirAll(segDir, 0755)
	os.WriteFile(filepath.Join(segDir, "seg000.ts"), []byte("fake-ts-data"), 0644)

	req := httptest.NewRequest("GET", "/hls/abc123/file001/seg000.ts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "fake-ts-data" {
		t.Fatalf("unexpected body: %q", w.Body.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/go-server
go test ./internal/hls/... -run "TestManifest|TestSegment" -v
```

Expected: FAIL — `NewHLSServer` not defined.

- [ ] **Step 3: Implement HLS HTTP server**

Create `backend/go-server/internal/hls/server.go`:

```go
package hls

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type HLSServer struct {
	port   int
	hlsDir string
}

func NewHLSServer(port int, hlsDir string) *HLSServer {
	return &HLSServer{port: port, hlsDir: hlsDir}
}

// Handler returns the HTTP handler (used directly in tests and by Start).
func (s *HLSServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hls/", s.serveHLS)
	return mux
}

// Start begins listening on the configured port. Blocks until error.
func (s *HLSServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[hls-http] listening on %s", addr)
	return http.ListenAndServe(addr, s.Handler())
}

// serveHLS handles:
//
//	GET /hls/{infoHash}/{fileId}/playlist.m3u8
//	GET /hls/{infoHash}/{fileId}/seg{N}.ts
func (s *HLSServer) serveHLS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Cache-Control", "no-cache")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Path: /hls/{infoHash}/{fileId}/{filename}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/hls/"), "/")
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}
	infoHash, fileId, filename := parts[0], parts[1], parts[2]

	// Guard against path traversal
	if strings.Contains(infoHash, "..") || strings.Contains(fileId, "..") || strings.Contains(filename, "..") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(s.hlsDir, infoHash, "hls", fileId, filename)

	if strings.HasSuffix(filename, ".m3u8") {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	} else if strings.HasSuffix(filename, ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
	} else {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend/go-server
go test ./internal/hls/... -run "TestManifest|TestSegment" -v
```

Expected: All 4 tests PASS.

- [ ] **Step 5: Verify full hls package tests pass**

```bash
go test ./internal/hls/... -v
```

Expected: All tests (from Task 4 + Task 5) PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/go-server/internal/hls/server.go backend/go-server/internal/hls/server_test.go
git commit -m "feat(hls): HTTP server for manifest and segment delivery"
```

---

## Task 6: Go — GetFileInfo MKV probe (TDD)

**Files:**
- Modify: `backend/go-server/internal/torrent/service.go`

Add `probeResult`, `probeCache`, `ffprobePath`, and the ffprobe logic to `GetFileInfo`.

- [ ] **Step 1: Write failing test**

Create `backend/go-server/internal/torrent/service_mkv_test.go`:

```go
package torrent_test

import (
	"testing"

	"server/internal/torrent"
)

func TestIsMKV(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"video.mkv", true},
		{"video.MKV", true},
		{"video.mp4", false},
		{"video.avi", false},
		{"", false},
	}
	for _, tc := range cases {
		got := torrent.IsMKV(tc.path)
		if got != tc.want {
			t.Errorf("IsMKV(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/go-server
go test ./internal/torrent/... -run TestIsMKV -v
```

Expected: FAIL — `IsMKV` not defined.

- [ ] **Step 3: Add probe types and IsMKV to service.go**

At the top of `backend/go-server/internal/torrent/service.go`, add after the imports block — add new imports and fields. Replace the `TorrentService` struct and `NewTorrentService` function with:

```go
// Add to imports:
//   "encoding/json"
//   "os/exec"
//   "strings"         (already present)
//   "sync"            (already present)
//   "server/internal/hls"
```

Add the following type and function (before `TorrentService` struct):

```go
type probeResult struct {
	DurationSec float64
	AudioTracks []*pb.AudioTrack
}

// IsMKV returns true if the file path has a .mkv extension (case-insensitive).
// Exported for testing.
func IsMKV(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".mkv"
}
```

Replace the `TorrentService` struct and `NewTorrentService`:

```go
type TorrentService struct {
	pb.UnimplementedTorrentServiceServer
	repo          *Repository
	speedTrackers map[string]*SpeedTracker
	trackerMu     sync.Mutex
	transcoder    *hls.HLSTranscoder
	ffprobePath   string
	probeCache    map[string]*probeResult
	probeMu       sync.RWMutex
}

func NewTorrentService(repo *Repository, transcoder *hls.HLSTranscoder, ffprobePath string) *TorrentService {
	return &TorrentService{
		repo:          repo,
		speedTrackers: make(map[string]*SpeedTracker),
		transcoder:    transcoder,
		ffprobePath:   ffprobePath,
		probeCache:    make(map[string]*probeResult),
	}
}
```

- [ ] **Step 4: Update GetFileInfo to return MKV metadata**

Replace the `GetFileInfo` method body (after the existing file lookup) to add MKV probe logic. Find the return statement `return &pb.FileInfoResponse{...}` and replace it with:

```go
	mimeType := mime.TypeByExtension(filepath.Ext(f.DisplayPath()))
	resp := &pb.FileInfoResponse{
		FileName:  filepath.Base(f.DisplayPath()),
		FilePath:  f.DisplayPath(),
		TotalSize: f.Length(),
		MimeType:  mimeType,
		IsReady:   true,
	}

	if IsMKV(f.DisplayPath()) {
		resp.MimeType = "video/mp4"
		probe := s.getOrProbe(req.GetInfoHash(), req.GetFileId(), t, f)
		if probe != nil {
			resp.DurationSec = probe.DurationSec
			resp.AudioTracks = probe.AudioTracks
		}
	}

	return resp, nil
```

Add the `getOrProbe` method to `TorrentService`:

```go
func (s *TorrentService) getOrProbe(infoHash, fileId string, t *lt.Torrent, f *lt.File) *probeResult {
	key := infoHash + ":" + fileId
	s.probeMu.RLock()
	if cached, ok := s.probeCache[key]; ok {
		s.probeMu.RUnlock()
		return cached
	}
	s.probeMu.RUnlock()

	if s.ffprobePath == "" {
		return nil
	}

	diskPath := filepath.Join(s.repo.dataDir, t.Name(), f.Path())
	result := runFFprobe(s.ffprobePath, diskPath)
	if result == nil {
		return nil
	}

	s.probeMu.Lock()
	s.probeCache[key] = result
	s.probeMu.Unlock()
	return result
}

func runFFprobe(ffprobePath, filePath string) *probeResult {
	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[ffprobe] failed on %s: %v", filePath, err)
		return nil
	}

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
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		log.Printf("[ffprobe] json parse error: %v", err)
		return nil
	}

	result := &probeResult{}
	for _, s := range probe.Streams {
		if s.CodecType == "video" && result.DurationSec == 0 {
			if d, err := strconv.ParseFloat(s.Duration, 64); err == nil {
				result.DurationSec = d
			}
		}
		if s.CodecType == "audio" {
			result.AudioTracks = append(result.AudioTracks, &pb.AudioTrack{
				Index:    int32(s.Index),
				Language: s.Tags.Language,
				Codec:    s.CodecName,
				Title:    s.Tags.Title,
			})
		}
	}
	return result
}
```

Add `"strconv"` and `"os/exec"` and `"encoding/json"` to the imports of `service.go`.

Also add `"server/internal/hls"` to imports.

- [ ] **Step 5: Fix server.go to match updated NewTorrentService signature**

In `backend/go-server/internal/grpc/server.go`, the `NewTorrentService` call will now fail because it needs new arguments. Temporarily pass `nil` and `""` to keep compilation:

```go
svc := torrent.NewTorrentService(repo, nil, "")
```

(This will be replaced in Task 8 with real values.)

- [ ] **Step 6: Run test**

```bash
cd backend/go-server
go test ./internal/torrent/... -run TestIsMKV -v
```

Expected: PASS.

- [ ] **Step 7: Verify build**

```bash
go build ./...
```

Expected: No errors.

- [ ] **Step 8: Commit**

```bash
git add backend/go-server/internal/torrent/service.go backend/go-server/internal/torrent/service_mkv_test.go backend/go-server/internal/grpc/server.go
git commit -m "feat(service): MKV detection, ffprobe cache, IsMKV helper"
```

---

## Task 7: Go — StartHLS and StopHLS gRPC handlers

**Files:**
- Modify: `backend/go-server/internal/torrent/service.go`

- [ ] **Step 1: Write failing test**

Add to `backend/go-server/internal/torrent/service_mkv_test.go`:

```go
func TestStartHLS_ReturnsNotFound_UnknownTorrent(t *testing.T) {
	repo := newTestRepository()
	svc := torrent.NewTorrentService(repo, nil, "")

	_, err := svc.StartHLS(context.Background(), &pb.HLSRequest{
		InfoHash: "nonexistent",
		FileId:   "nonexistent:0",
	})
	if err == nil {
		t.Fatal("expected error for unknown torrent, got nil")
	}
}
```

Add required imports to the test file:
```go
import (
    "context"
    "testing"
    pb "server/proto"
    "server/internal/torrent"
)
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/go-server
go test ./internal/torrent/... -run TestStartHLS -v
```

Expected: FAIL — `StartHLS` not defined on `TorrentService` (or interface not satisfied).

- [ ] **Step 3: Implement StartHLS and StopHLS**

Add these methods to `backend/go-server/internal/torrent/service.go`:

```go
func (s *TorrentService) StartHLS(ctx context.Context, req *pb.HLSRequest) (*pb.HLSResponse, error) {
	if s.transcoder == nil {
		return &pb.HLSResponse{Success: false}, status.Error(codes.Unavailable, "HLS transcoder not configured")
	}

	t, ok := s.repo.GetTorrent(req.GetInfoHash())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.GetInfoHash())
	}

	f, ok := s.repo.GetFile(req.GetInfoHash(), req.GetFileId())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "file not found: %s", req.GetFileId())
	}

	// lt.Reader implements io.ReadSeekCloser; transcoder takes ownership and closes it
	reader := f.NewReader()
	reader.SetResponsive()
	reader.SetReadahead(readahead)

	manifestURL, err := s.transcoder.Start(req.GetInfoHash(), req.GetFileId(), req.GetSeekTimeSec(), reader)
	if err != nil {
		log.Printf("[hls] StartHLS failed for %s/%s: %v", req.GetInfoHash(), req.GetFileId(), err)
		return &pb.HLSResponse{Success: false}, nil
	}

	probe := s.getOrProbe(req.GetInfoHash(), req.GetFileId(), t, f)
	resp := &pb.HLSResponse{
		ManifestUrl: manifestURL,
		Success:     true,
	}
	if probe != nil {
		resp.DurationSec = probe.DurationSec
		resp.AudioTracks = probe.AudioTracks
	}
	return resp, nil
}

func (s *TorrentService) StopHLS(ctx context.Context, req *pb.HLSRequest) (*pb.HLSResponse, error) {
	if s.transcoder != nil {
		s.transcoder.Stop(req.GetFileId())
	}
	return &pb.HLSResponse{Success: true}, nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend/go-server
go test ./internal/torrent/... -run "TestStartHLS|TestIsMKV" -v
```

Expected: All tests PASS.

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add backend/go-server/internal/torrent/service.go backend/go-server/internal/torrent/service_mkv_test.go
git commit -m "feat(service): StartHLS and StopHLS gRPC handlers"
```

---

## Task 8: Go — Wire everything in server.go

**Files:**
- Modify: `backend/go-server/internal/grpc/server.go`

- [ ] **Step 1: Update StartServer to wire FFmpeg + HLS**

Replace the entire content of `backend/go-server/internal/grpc/server.go`:

```go
package grpc_server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"server/config"
	"server/internal/ffmpeg"
	"server/internal/hls"
	"server/internal/prowlarr"
	"server/internal/torrent"
	pb "server/proto"

	"google.golang.org/grpc"
)

func StartServer(cfg config.Config) {
	// Start Prowlarr — non-fatal
	pm := prowlarr.NewManager(
		cfg.Prowlarr.BinDir,
		cfg.Prowlarr.DataDir,
		cfg.Prowlarr.Port,
		cfg.Prowlarr.SeedIndexers,
	)
	if err := pm.Start(); err != nil {
		log.Printf("[prowlarr] startup failed: %v (search will be unavailable)", err)
	}
	defer pm.Stop()

	// Ensure FFmpeg binaries — fatal if unavailable
	ffmpegBinDir := "./bin"
	ffmpegPath, err := ffmpeg.EnsureFFmpeg(cfg.FFmpegPath, ffmpegBinDir)
	if err != nil {
		log.Fatalf("[ffmpeg] binary unavailable: %v", err)
	}
	ffprobePath, err := ffmpeg.EnsureFFprobe(cfg.FFprobePath, ffmpegBinDir)
	if err != nil {
		log.Fatalf("[ffprobe] binary unavailable: %v", err)
	}
	log.Printf("[ffmpeg] using %s", ffmpegPath)
	log.Printf("[ffprobe] using %s", ffprobePath)

	// HLS base URL for manifest links returned to clients
	hlsBaseURL := fmt.Sprintf("http://localhost:%d", cfg.HLSPort)
	transcoder := hls.NewHLSTranscoder(ffmpegPath, cfg.HLSDir, hlsBaseURL)
	hlsSrv := hls.NewHLSServer(cfg.HLSPort, cfg.HLSDir)

	// Start HLS HTTP server in background
	go func() {
		if err := hlsSrv.Start(); err != nil {
			log.Printf("[hls-http] server error: %v", err)
		}
	}()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.GRPCPort, err)
	}

	client, err := torrent.NewClient(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}
	}()

	repo := torrent.NewRepository(client, cfg.DataDir, cfg.MetadataTimeout)
	svc := torrent.NewTorrentService(repo, transcoder, ffprobePath)

	grpcServer := grpc.NewServer()
	pb.RegisterTorrentServiceServer(grpcServer, svc)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server stopped: %v", err)
			stop <- syscall.SIGTERM
		}
	}()

	<-stop
	log.Println("Shutting down...")
	transcoder.StopAll()
	repo.Clearup()
	grpcServer.GracefulStop()
}
```

- [ ] **Step 2: Build and verify**

```bash
cd backend/go-server
go build ./...
```

Expected: No errors.

- [ ] **Step 3: Run all Go tests**

```bash
go test ./... -v
```

Expected: All existing tests pass. New tests from Tasks 3–7 pass.

- [ ] **Step 4: Commit**

```bash
git add backend/go-server/internal/grpc/server.go
git commit -m "feat(server): wire FFmpeg binary manager, HLS transcoder, HLS HTTP server"
```

---

## Task 9: Go — Integration test and fixtures

**Files:**
- Create: `backend/go-server/internal/hls/transcoder_integration_test.go`
- Create: `backend/go-server/internal/hls/testdata/sample_10s.mkv` (binary fixture)

- [ ] **Step 1: Generate test fixture**

Run this once (requires FFmpeg installed locally):

```bash
ffmpeg -f lavfi -i "testsrc=duration=10:size=320x240:rate=30" \
  -f lavfi -i "sine=frequency=440:duration=10" \
  -c:v libx264 -c:a aac \
  backend/go-server/internal/hls/testdata/sample_10s.mkv
```

Expected: `sample_10s.mkv` created (~500KB). Commit the binary file.

- [ ] **Step 2: Write integration test**

Create `backend/go-server/internal/hls/transcoder_integration_test.go`:

```go
package hls_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"server/internal/hls"
)

func TestHLS_TranscodesRealFile(t *testing.T) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not in PATH — skipping integration test")
	}

	tmp := t.TempDir()
	tr := hls.NewHLSTranscoder(ffmpegPath, tmp, "http://localhost:8091")

	fixture, err := os.Open("testdata/sample_10s.mkv")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}
	defer fixture.Close()

	manifestURL, err := tr.Start("testhash", "testhash:0", 0.0, fixture)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	if manifestURL == "" {
		t.Fatal("expected non-empty manifest URL")
	}

	// Verify playlist.m3u8 exists and has content
	manifestPath := filepath.Join(tmp, "testhash", "hls", "testhash:0", "playlist.m3u8")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("playlist.m3u8 not found: %v", err)
	}
	content := string(data)
	if !contains([]string{content}, "#EXTM3U") {
		t.Error("playlist.m3u8 missing #EXTM3U header")
	}

	// Verify at least 2 segments
	n := hls.CountSegmentsExported(filepath.Join(tmp, "testhash", "hls", "testhash:0"))
	if n < 2 {
		t.Fatalf("expected ≥2 segments, got %d", n)
	}
}
```

- [ ] **Step 3: Export CountSegments for test use**

Add to `backend/go-server/internal/hls/transcoder.go`:

```go
// CountSegmentsExported is the exported version of countSegments for integration tests.
func CountSegmentsExported(dir string) int {
	return countSegments(dir)
}
```

- [ ] **Step 4: Run integration test**

```bash
cd backend/go-server
go test ./internal/hls/... -run TestHLS_TranscodesRealFile -v
```

Expected: PASS if ffmpeg in PATH (SKIP if not). Fixture generates ≥2 `.ts` segments.

- [ ] **Step 5: Commit**

```bash
git add backend/go-server/internal/hls/transcoder_integration_test.go \
        backend/go-server/internal/hls/transcoder.go \
        backend/go-server/internal/hls/testdata/sample_10s.mkv
git commit -m "test(hls): integration test with 10s MKV fixture"
```

---

## Task 10: Spring — HLSStartResponse DTO and TorrentGrpcClient

**Files:**
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/HLSStartResponse.java`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/TorrentGrpcClient.java`

- [ ] **Step 1: Create HLSStartResponse DTO**

Create `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/HLSStartResponse.java`:

```java
package org.devMandali.magnetPlay.model;

import java.util.List;

public record HLSStartResponse(
    String manifestUrl,
    boolean success,
    double durationSec,
    List<AudioTrackDto> audioTracks
) {
    public record AudioTrackDto(int index, String language, String codec, String title) {}
}
```

- [ ] **Step 2: Add startHLS and stopHLS to TorrentGrpcClient**

In `TorrentGrpcClient.java`, add after the `deleteTorrent` method:

```java
// ─── StartHLS ────────────────────────────────────────────────────────────

public Mono<HLSResponse> startHLS(String infoHash, String fileId, double seekTimeSec) {
    return Mono.fromCallable(() -> torrentServiceBlockingStub
            .withDeadlineAfter(35, TimeUnit.SECONDS)
            .startHLS(HLSRequest.newBuilder()
                    .setInfoHash(infoHash)
                    .setFileId(fileId)
                    .setSeekTimeSec(seekTimeSec)
                    .build()))
            .subscribeOn(grpcScheduler)
            .doOnError(e -> logger.error("startHLS gRPC error for {}/{}", infoHash, fileId, e));
}

// ─── StopHLS ─────────────────────────────────────────────────────────────

public Mono<HLSResponse> stopHLS(String infoHash, String fileId) {
    return Mono.fromCallable(() -> torrentServiceBlockingStub
            .withDeadlineAfter(10, TimeUnit.SECONDS)
            .stopHLS(HLSRequest.newBuilder()
                    .setInfoHash(infoHash)
                    .setFileId(fileId)
                    .build()))
            .subscribeOn(grpcScheduler)
            .doOnError(e -> logger.error("stopHLS gRPC error for {}/{}", infoHash, fileId, e));
}
```

Add imports to `TorrentGrpcClient.java`:
```java
import org.devMandali.magnetPlay.HLSRequest;
import org.devMandali.magnetPlay.HLSResponse;
import org.devMandali.magnetPlay.model.HLSStartResponse;
```

- [ ] **Step 3: Build**

```bash
cd backend/mp-spring
./mvnw compile -q
```

Expected: Compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/HLSStartResponse.java \
        backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/TorrentGrpcClient.java
git commit -m "feat(spring): HLSStartResponse DTO and gRPC client stubs for HLS"
```

---

## Task 11: Spring — TorrentService and TorrentController HLS endpoints

**Files:**
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java`
- Modify: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/TorrentController.java`

- [ ] **Step 1: Add startHLS and stopHLS to TorrentService**

In `TorrentService.java`, add after the `deleteTorrent` method:

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
                            .toList()
            ))
            .doOnError(e -> logger.error("startHLS error for {}/{}", infoHash, fileId, e));
}

public Mono<String> stopHLS(String infoHash, String fileId) {
    return grpcClient.stopHLS(infoHash, fileId)
            .map(r -> r.getSuccess() ? "stopped" : "not found")
            .doOnError(e -> logger.error("stopHLS error for {}/{}", infoHash, fileId, e));
}
```

Add import: `import org.devMandali.magnetPlay.model.HLSStartResponse;`

- [ ] **Step 2: Add HLS endpoints to TorrentController**

In `TorrentController.java`, add after the `delete` endpoint:

```java
@PostMapping("/hls/{infoHash}/start")
public Mono<ResponseEntity<HLSStartResponse>> startHLS(
        @PathVariable String infoHash,
        @RequestParam String fileId,
        @RequestParam(defaultValue = "0") double t) {
    return service.startHLS(infoHash, fileId, t)
            .map(resp -> resp.success()
                    ? ResponseEntity.ok(resp)
                    : ResponseEntity.status(HttpStatus.SERVICE_UNAVAILABLE).<HLSStartResponse>build())
            .doOnError(e -> logger.error("startHLS REST error for {}/{}", infoHash, fileId, e))
            .onErrorReturn(ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build());
}

@DeleteMapping("/hls/{infoHash}/stop")
public Mono<ResponseEntity<String>> stopHLS(
        @PathVariable String infoHash,
        @RequestParam String fileId) {
    return service.stopHLS(infoHash, fileId)
            .map(ResponseEntity::ok)
            .onErrorReturn(ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build());
}
```

Add import: `import org.devMandali.magnetPlay.model.HLSStartResponse;`

- [ ] **Step 3: Build**

```bash
cd backend/mp-spring
./mvnw compile -q
```

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java \
        backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/TorrentController.java
git commit -m "feat(spring): POST /hls/{hash}/start and DELETE /hls/{hash}/stop endpoints"
```

---

## Task 12: Spring — TorrentHLSControllerTest

**Files:**
- Create: `backend/mp-spring/src/test/java/org/devMandali/magnetPlay/controller/TorrentHLSControllerTest.java`

- [ ] **Step 1: Write tests**

Create `TorrentHLSControllerTest.java` (follow the pattern from `TorrentStatsControllerTest.java`):

```java
package org.devMandali.magnetPlay.controller;

import org.devMandali.magnetPlay.model.HLSStartResponse;
import org.devMandali.magnetPlay.service.TorrentService;
import org.devMandali.magnetPlay.session.SessionManager;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.reactive.WebFluxTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.reactive.server.WebTestClient;
import reactor.core.publisher.Mono;

import java.util.List;

import static org.mockito.Mockito.when;

@WebFluxTest(TorrentController.class)
class TorrentHLSControllerTest {

    @Autowired
    private WebTestClient webTestClient;

    @MockBean
    private TorrentService torrentService;

    @MockBean
    private SessionManager sessionManager;

    @Test
    void startHLS_returns200_withManifestUrl() {
        HLSStartResponse mockResp = new HLSStartResponse(
                "http://localhost:8091/hls/abc123/abc123:0/playlist.m3u8",
                true, 7243.0, List.of());

        when(torrentService.startHLS("abc123", "abc123:0", 0.0))
                .thenReturn(Mono.just(mockResp));

        webTestClient.post()
                .uri("/v1/torrent/hls/abc123/start?fileId=abc123:0&t=0")
                .exchange()
                .expectStatus().isOk()
                .expectBody(HLSStartResponse.class)
                .value(resp -> {
                    assert resp.manifestUrl().contains("playlist.m3u8");
                    assert resp.success();
                    assert resp.durationSec() == 7243.0;
                });
    }

    @Test
    void startHLS_returns503_whenSuccessFalse() {
        HLSStartResponse failResp = new HLSStartResponse("", false, 0.0, List.of());

        when(torrentService.startHLS("abc123", "abc123:0", 0.0))
                .thenReturn(Mono.just(failResp));

        webTestClient.post()
                .uri("/v1/torrent/hls/abc123/start?fileId=abc123:0&t=0")
                .exchange()
                .expectStatus().is5xxServerError();
    }

    @Test
    void stopHLS_returns200() {
        when(torrentService.stopHLS("abc123", "abc123:0"))
                .thenReturn(Mono.just("stopped"));

        webTestClient.delete()
                .uri("/v1/torrent/hls/abc123/stop?fileId=abc123:0")
                .exchange()
                .expectStatus().isOk();
    }
}
```

- [ ] **Step 2: Run tests**

```bash
cd backend/mp-spring
./mvnw test -pl . -Dtest=TorrentHLSControllerTest -q
```

Expected: All 3 tests PASS.

- [ ] **Step 3: Run full Spring test suite**

```bash
./mvnw test -q
```

Expected: All tests pass, no regressions.

- [ ] **Step 4: Commit**

```bash
git add backend/mp-spring/src/test/java/org/devMandali/magnetPlay/controller/TorrentHLSControllerTest.java
git commit -m "test(spring): TorrentHLSControllerTest for start/stop endpoints"
```

---

## Task 13: Frontend — new types

**Files:**
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Add new types**

In `frontend/src/types/index.ts`, add the following after the `SearchResultsResponse` interface:

```typescript
export interface AudioTrack {
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
}
```

Also extend `ActivePlayer` to carry MKV metadata:

```typescript
export interface ActivePlayer {
  infoHash: string;
  fileId: string;
  mimeType: string;
  isMkv: boolean;
  manifestUrl: string;
  durationSec: number;
  audioTracks: AudioTrack[];
}
```

- [ ] **Step 2: Fix App.tsx to use updated ActivePlayer**

In `frontend/src/App.tsx`, update the `draft` and `active` state initialization:

Find:
```typescript
const [draft, setDraft] = useState({ infoHash: '', fileId: '', mimeType: 'video/mp4' });
```

Replace with:
```typescript
const [draft, setDraft] = useState({ infoHash: '', fileId: '', mimeType: 'video/mp4', fileName: '' });
```

Find `handleFetchFiles` where it sets the draft and add `fileName`:
```typescript
setDraft(d => ({ ...d, infoHash: hash, fileId: firstVideo.id, fileName: firstVideo.name }));
```

Find `handleFullReset` and update:
```typescript
setDraft({ infoHash: '', fileId: '', mimeType: 'video/mp4', fileName: '' });
```

- [ ] **Step 3: Build TypeScript**

```bash
cd frontend
npm run build 2>&1 | tail -20
```

Expected: Build succeeds (type errors from VideoPlayer will show — expected, fixed in Task 14).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/App.tsx
git commit -m "feat(types): AudioTrack, HLSStartResponse, extend ActivePlayer for MKV"
```

---

## Task 14: Frontend — VideoPlayer MKV + HLS logic

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/VideoPlayer.tsx`

- [ ] **Step 1: Update App.tsx handlePlay for MKV detection**

In `frontend/src/App.tsx`, replace the `handlePlay` function with:

```typescript
const handlePlay = async () => {
  if (!draft.infoHash.trim() || !draft.fileId.trim()) return;
  setError(null);
  setMoovStatus('idle');
  setSubtitleTracks([]);
  setConfigCollapsed(true);

  const isMkv = draft.fileName.toLowerCase().endsWith('.mkv');

  if (isMkv) {
    try {
      const res = await fetch(
        `/v1/torrent/hls/${draft.infoHash}/start?fileId=${encodeURIComponent(draft.fileId)}&t=0`,
        { method: 'POST', signal: AbortSignal.timeout(35000) }
      );
      if (!res.ok) throw new Error(`HLS start failed: ${res.status}`);
      const data: HLSStartResponse = await res.json();
      if (!data.success) throw new Error('HLS transcoder failed to start');
      setActive({
        infoHash: draft.infoHash,
        fileId: draft.fileId,
        mimeType: 'application/x-mpegURL',
        isMkv: true,
        manifestUrl: data.manifestUrl,
        durationSec: data.durationSec,
        audioTracks: data.audioTracks,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start HLS stream');
    }
  } else {
    setActive({
      infoHash: draft.infoHash,
      fileId: draft.fileId,
      mimeType: draft.mimeType,
      isMkv: false,
      manifestUrl: '',
      durationSec: 0,
      audioTracks: [],
    });
  }
};
```

Add `HLSStartResponse` to the import from `./types`.

Also update `handleReset` to stop HLS on reset — add before `setActive(null)`:

```typescript
if (active?.isMkv) {
  fetch(`/v1/torrent/hls/${active.infoHash}/stop?fileId=${encodeURIComponent(active.fileId)}`,
    { method: 'DELETE' }).catch(() => {});
}
```

Update the `VideoPlayer` usage in the JSX to pass new props:

```tsx
{active && (
  <VideoPlayer
    infoHash={active.infoHash}
    fileId={active.fileId}
    mimeType={active.mimeType}
    isMkv={active.isMkv}
    manifestUrl={active.manifestUrl}
    durationSec={active.durationSec}
    audioTracks={active.audioTracks}
    subtitleTracks={subtitleTracks}
    onError={handleError}
    onMoovStatus={setMoovStatus}
  />
)}
```

- [ ] **Step 2: Update VideoPlayer Props interface**

In `frontend/src/components/VideoPlayer.tsx`, replace the `Props` interface:

```typescript
interface Props {
  infoHash: string;
  fileId: string;
  mimeType: string;
  isMkv: boolean;
  manifestUrl: string;
  durationSec: number;
  audioTracks: AudioTrack[];
  subtitleTracks: SubtitleTrack[];
  onError?: (err: { code: number; message: string } | null) => void;
  onMoovStatus?: (status: MoovStatus) => void;
}
```

Add `AudioTrack` to the import from `../types`:
```typescript
import { SubtitleTrack, MoovStatus, AudioTrack } from '../types';
```

Update the function signature:
```typescript
export default function VideoPlayer({
  infoHash, fileId, mimeType, isMkv, manifestUrl, durationSec, audioTracks,
  subtitleTracks, onError, onMoovStatus
}: Props) {
```

- [ ] **Step 3: Add debounce ref and seek handler to VideoPlayer**

Add these refs near the top of the `VideoPlayer` component body (after existing refs):

```typescript
const seekDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
const isMkvRef = useRef(isMkv);
const infoHashRef = useRef(infoHash);
const fileIdRef = useRef(fileId);

useEffect(() => { isMkvRef.current = isMkv; }, [isMkv]);
useEffect(() => { infoHashRef.current = infoHash; }, [infoHash]);
useEffect(() => { fileIdRef.current = fileId; }, [fileId]);
```

- [ ] **Step 4: Update player init for MKV source**

In the `useEffect` that initialises the player (the big one with `videojs(...)`), replace:
```typescript
sources: [{ src: streamUrl, type: mimeType }],
```
with:
```typescript
sources: isMkv
  ? [{ src: manifestUrl, type: 'application/x-mpegURL' }]
  : [{ src: streamUrl, type: mimeType }],
```

After `player.on('error', ...)` block, add the MKV-specific setup:

```typescript
if (isMkv) {
  // Set known duration so seekbar renders
  if (durationSec > 0) {
    player.duration(durationSec);
  }

  // Seek handler: debounce 300ms, restart HLS at new timestamp
  player.on('seeked', () => {
    if (!isMkvRef.current) return;
    if (seekDebounceRef.current) clearTimeout(seekDebounceRef.current);
    seekDebounceRef.current = setTimeout(async () => {
      const t = player.currentTime() ?? 0;
      try {
        const res = await fetch(
          `/v1/torrent/hls/${infoHashRef.current}/start?fileId=${encodeURIComponent(fileIdRef.current)}&t=${t}`,
          { method: 'POST', signal: AbortSignal.timeout(35000) }
        );
        if (!res.ok) return;
        const data: HLSStartResponse = await res.json();
        if (data.success && data.manifestUrl) {
          player.src([{ src: data.manifestUrl, type: 'application/x-mpegURL' }]);
          player.play();
        }
      } catch { /* ignore */ }
    }, 300);
  });
}
```

Add `HLSStartResponse` to the import from `../types`.

- [ ] **Step 5: Unmount cleanup — stop HLS**

In the cleanup function of the player init `useEffect` (the `return () => { ... }` block), add before `player.dispose()`:

```typescript
if (seekDebounceRef.current) clearTimeout(seekDebounceRef.current);
if (isMkvRef.current) {
  fetch(
    `/v1/torrent/hls/${infoHashRef.current}/stop?fileId=${encodeURIComponent(fileIdRef.current)}`,
    { method: 'DELETE' }
  ).catch(() => {});
}
```

- [ ] **Step 6: Update src effect for MKV**

Find the `useEffect` that updates src on change:
```typescript
useEffect(() => {
  const player = playerRef.current;
  if (!player) return;
  player.src([{ src: streamUrl, type: mimeType }]);
}, [streamUrl, mimeType]);
```

Replace with:
```typescript
useEffect(() => {
  const player = playerRef.current;
  if (!player) return;
  if (isMkv) {
    player.src([{ src: manifestUrl, type: 'application/x-mpegURL' }]);
    if (durationSec > 0) player.duration(durationSec);
  } else {
    player.src([{ src: streamUrl, type: mimeType }]);
  }
}, [streamUrl, mimeType, isMkv, manifestUrl, durationSec]);
```

- [ ] **Step 7: Build and check types**

```bash
cd frontend
npm run build 2>&1 | tail -30
```

Expected: Build succeeds with no TypeScript errors.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/App.tsx frontend/src/components/VideoPlayer.tsx
git commit -m "feat(frontend): MKV detection, HLS source, seek debounce, cleanup on unmount"
```

---

## Task 15: Frontend — AudioTrackMenuButton in video.js

**Files:**
- Modify: `frontend/src/lib/videoSetup.ts`
- Modify: `frontend/src/components/VideoPlayer.tsx`

- [ ] **Step 1: Add registerAudioTrackButton to videoSetup.ts**

In `frontend/src/lib/videoSetup.ts`, add the following export after `registerSkipButtons`:

```typescript
export function registerAudioTrackButton(): void {
  if (videojs.getComponent('AudioTrackMenuButton')) return;

  const MenuButton = videojs.getComponent('MenuButton');
  const MenuItem = videojs.getComponent('MenuItem');

  class AudioTrackMenuItem extends MenuItem {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    constructor(player: ReturnType<typeof videojs>, options: any) {
      super(player, options);
      this.selectable = true;
      this.isSelected_ = options.selected ?? false;
      if (this.isSelected_) this.addClass('vjs-selected');
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    handleClick(event: any) {
      super.handleClick(event);
      const tracks = this.player().audioTracks();
      for (let i = 0; i < tracks.length; i++) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (tracks[i] as any).enabled = tracks[i].label === this.options_.label;
      }
    }
  }

  class AudioTrackMenuButton extends MenuButton {
    createEl() {
      const el = super.createEl();
      el.setAttribute('title', 'Audio Track');
      return el;
    }

    createItems() {
      const tracks = this.player().audioTracks();
      const items = [];
      for (let i = 0; i < tracks.length; i++) {
        const track = tracks[i];
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        items.push(new AudioTrackMenuItem(this.player(), {
          label: track.label || track.language || `Track ${i + 1}`,
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          selected: (track as any).enabled,
        }));
      }
      return items;
    }

    buildCSSClass() {
      return `vjs-audio-track-btn ${super.buildCSSClass()}`;
    }
  }

  videojs.registerComponent('AudioTrackMenuItem', AudioTrackMenuItem);
  videojs.registerComponent('AudioTrackMenuButton', AudioTrackMenuButton);
}
```

- [ ] **Step 2: Register in main.tsx (same pattern as registerSkipButtons)**

In `frontend/src/main.tsx`, add the import and registration call:

```typescript
import { registerSkipButtons, registerStatsButton, configureVhs, registerAudioTrackButton } from './lib/videoSetup';

registerSkipButtons();
registerStatsButton();
registerAudioTrackButton();
configureVhs();
```

- [ ] **Step 3: Add AudioTrackMenuButton to control bar only for MKV**

In the `videojs(videoEl, { ... })` call, the `controlBar.children` array currently ends with `'fullscreenToggle'`. For MKV files, insert the audio button before fullscreen:

Replace the `controlBar` configuration:
```typescript
controlBar: {
  children: [
    'playToggle',
    { name: 'SkipBackButton', seconds: 10 },
    { name: 'SkipForwardButton', seconds: 10 },
    'volumePanel',
    'currentTimeDisplay',
    'timeDivider',
    'durationDisplay',
    'progressControl',
    'remainingTimeDisplay',
    'playbackRateMenuButton',
    { name: 'StatsButton', infoHash, fileId },
    ...(isMkv && audioTracks.length > 1 ? ['AudioTrackMenuButton'] : []),
    'fullscreenToggle',
  ],
},
```

- [ ] **Step 4: Build**

```bash
cd frontend
npm run build 2>&1 | tail -20
```

Expected: No TypeScript errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/videoSetup.ts frontend/src/components/VideoPlayer.tsx
git commit -m "feat(player): AudioTrackMenuButton for MKV multi-audio track switching"
```

---

## Task 16: End-to-end manual verification

No code changes in this task — verify the full feature works.

- [ ] **Step 1: Start all services**

Terminal 1 (Go sidecar):
```bash
cd backend/go-server
go run main.go
```
Expected: `[ffmpeg] using ./bin/ffmpeg.exe`, `[hls-http] listening on :8091`, `gRPC server listening on :50051`

Terminal 2 (Spring Boot):
```bash
cd backend/mp-spring
./mvnw spring-boot:run
```
Expected: Spring starts on port 8080.

Terminal 3 (Frontend):
```bash
cd frontend
npm run dev
```
Expected: Vite starts on port 5173.

- [ ] **Step 2: Test MP4 path — no regression**

1. Open `http://localhost:5173`
2. Add an MP4 torrent magnet
3. Select an `.mp4` file, click Play
4. Expected: Plays normally with byte-range seeking — identical to pre-feature behavior

- [ ] **Step 3: Test MKV initial load**

1. Add an MKV torrent magnet
2. Select a `.mkv` file, click Play
3. Expected:
   - No errors in console
   - Video plays in browser
   - Seekbar shows full timeline (if ffprobe succeeded)
   - Source type is `application/x-mpegURL` (check in DevTools Network tab — manifest request to `:8091`)

- [ ] **Step 4: Test MKV seeking**

1. With MKV playing, click seekbar at ~50% mark
2. Expected:
   - Brief rebuffer (~1-2s)
   - Playback resumes at correct timestamp
   - Network tab shows new `POST /hls/{hash}/start?t=N` request

- [ ] **Step 5: Test multi-audio (if MKV has multiple audio tracks)**

1. With MKV playing that has multiple audio tracks
2. Expected: Audio switcher button visible in control bar
3. Click audio switcher → select second track
4. Expected: Audio switches to selected language/track

- [ ] **Step 6: Test cleanup**

1. While MKV is playing, click reset/back button
2. Expected:
   - `DELETE /hls/{hash}/stop` fired in Network tab
   - Segment directory removed: `./downloads/{hash}/hls/{fileId}/` should be gone

- [ ] **Step 7: Final commit — graphify update**

```bash
cd backend
make -f MakeFile all
```

Then update graphify:
```
/graphify . --update
```

```bash
git add -A
git commit -m "feat(mkv-hls): MKV streaming via FFmpeg HLS — full implementation

- FFmpeg/ffprobe auto-download (BtbN/FFmpeg-Builds)
- HLS transcoder: per-fileId FFmpeg, audio copy→AAC fallback, 4s segments
- HLS HTTP server on :8091 with CORS
- gRPC: StartHLS / StopHLS RPCs
- Spring: POST /hls/{hash}/start, DELETE /hls/{hash}/stop
- Frontend: MKV detection, VHS source, seek debounce, AudioTrackMenuButton
- MP4 path completely unchanged"
```
