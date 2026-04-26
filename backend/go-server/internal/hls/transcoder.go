package hls

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

type activeJob struct {
	cancel context.CancelFunc
}

// RemuxHandler spawns FFmpeg to remux a torrent file (container-only when possible)
// and streams the result as fragmented MP4 directly to the HTTP client.
// Each new request for a fileId cancels any prior stream for that fileId.
type RemuxHandler struct {
	ffmpegPath  string
	fileBaseURL string // rawfile base URL, e.g. "http://localhost:8091/rawfile"
	baseURL     string // public base URL for this server, e.g. "http://localhost:8091"
	mu          sync.Mutex
	activeJobs  map[string]*activeJob // fileId → current job
}

func NewRemuxHandler(ffmpegPath, fileBaseURL, baseURL string) *RemuxHandler {
	return &RemuxHandler{
		ffmpegPath:  ffmpegPath,
		fileBaseURL: fileBaseURL,
		baseURL:     baseURL,
		activeJobs:  make(map[string]*activeJob),
	}
}

// GetRemuxBaseURL returns the remux URL for a file without the ?t= seek param.
// The caller appends &t={seekSec} for each play/seek request.
func (h *RemuxHandler) GetRemuxBaseURL(infoHash, fileId, videoCodec, audioCodec string) string {
	encodedFileId := base64.RawURLEncoding.EncodeToString([]byte(fileId))
	return fmt.Sprintf("%s/remux/%s/%s?vc=%s&ac=%s", h.baseURL, infoHash, encodedFileId, videoCodec, audioCodec)
}

// StopAll cancels all active remux jobs (called on shutdown).
func (h *RemuxHandler) StopAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for fileId, job := range h.activeJobs {
		job.cancel()
		log.Printf("[remux] stopped job fileId=%s", fileId)
	}
	h.activeJobs = make(map[string]*activeJob)
}

// ServeHTTP handles GET /remux/{infoHash}/{fileIdB64}?vc=...&ac=...&t={seekSec}
func (h *RemuxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/remux/")
	slashIdx := strings.Index(trimmed, "/")
	if slashIdx < 0 {
		http.NotFound(w, r)
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
		http.Error(w, "bad file id", http.StatusBadRequest)
		return
	}
	fileId := string(fileIdBytes)

	q := r.URL.Query()
	var seekSec float64
	fmt.Sscanf(q.Get("t"), "%f", &seekSec)
	videoCodec := q.Get("vc")
	audioCodec := q.Get("ac")
	if videoCodec == "" {
		videoCodec = "libx264"
	}
	if audioCodec == "" {
		audioCodec = "aac"
	}

	fileURL := fmt.Sprintf("%s/%s/%s", h.fileBaseURL, infoHash, encodedFileId)
	log.Printf("[remux] request hash=%s fileId=%s seek=%.1fs vc=%s ac=%s", infoHash, fileId, seekSec, videoCodec, audioCodec)

	ctx, cancel := context.WithCancel(r.Context())
	job := &activeJob{cancel: cancel}

	h.mu.Lock()
	if old, ok := h.activeJobs[fileId]; ok {
		old.cancel()
	}
	h.activeJobs[fileId] = job
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		if h.activeJobs[fileId] == job {
			delete(h.activeJobs, fileId)
		}
		h.mu.Unlock()
		cancel()
	}()

	args := BuildRemuxArgs(seekSec, fileURL, videoCodec, audioCodec)
	cmd := exec.CommandContext(ctx, h.ffmpegPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[remux] stdout pipe error: %v", err)
		http.Error(w, "pipe error", http.StatusInternalServerError)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[remux] ffmpeg start error: %v", err)
		http.Error(w, "ffmpeg start failed", http.StatusInternalServerError)
		return
	}
	log.Printf("[remux] ffmpeg spawned hash=%s fileId=%s seek=%.1fs", infoHash, fileId, seekSec)

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.WriteHeader(http.StatusOK)

	buf := make([]byte, 256*1024)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				log.Printf("[remux] client disconnected fileId=%s: %v", fileId, writeErr)
				break
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Printf("[remux] pipe read error fileId=%s: %v", fileId, readErr)
			}
			break
		}
	}

	cancel()
	cmd.Wait()
	log.Printf("[remux] stream ended fileId=%s", fileId)
}

// BuildRemuxArgs constructs the FFmpeg argument slice for fMP4 pipe output.
// videoCodec: "copy" or "libx264". audioCodec: "copy" or "aac".
func BuildRemuxArgs(seekSec float64, fileURL, videoCodec, audioCodec string) []string {
	args := []string{"-y"}
	if seekSec > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", seekSec))
	}
	args = append(args, "-i", fileURL, "-map", "0:v:0", "-map", "0:a")

	if videoCodec == "copy" {
		args = append(args, "-c:v", "copy")
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "veryfast", "-crf", "22")
	}

	if audioCodec == "copy" {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac")
	}

	args = append(args,
		"-movflags", "frag_keyframe+empty_moov",
		"-f", "mp4",
		"pipe:1",
	)
	log.Printf("[remux] FFmpeg args: %v", args)
	return args
}
