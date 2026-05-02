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
	"sync/atomic"
	"time"
)

// SubtitleHandler serves subtitle tracks as WebVTT via FFmpeg conversion.
// Two routes:
//
//	GET /subtitle/embedded/{infoHash}/{b64fileId}/{streamIndex}  — extract embedded MKV sub stream
//	GET /subtitle/file/{infoHash}/{b64fileId}                   — convert torrent subtitle file to WebVTT
type SubtitleHandler struct {
	ffmpegPath  atomic.Value // stores string; empty until binary is available
	fileBaseURL string       // e.g. "http://localhost:8091/rawfile"
}

func NewSubtitleHandler(ffmpegPath, fileBaseURL string) *SubtitleHandler {
	h := &SubtitleHandler{fileBaseURL: fileBaseURL}
	if ffmpegPath != "" {
		h.ffmpegPath.Store(ffmpegPath)
	}
	return h
}

// SetFFmpegPath updates the FFmpeg binary path after an async download completes.
func (h *SubtitleHandler) SetFFmpegPath(p string) { h.ffmpegPath.Store(p) }

// ServeEmbedded handles GET /subtitle/embedded/{infoHash}/{b64fileId}/{streamIndex}
func (h *SubtitleHandler) ServeEmbedded(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

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

	_ = filepath.Ext(fileId) // keep filepath import used

	h.runFFmpegToWebVTT(w, []string{
		"-i", fileURL,
		"-f", "webvtt",
		"pipe:1",
	})
}

func (h *SubtitleHandler) runFFmpegToWebVTT(w http.ResponseWriter, args []string) {
	fp, _ := h.ffmpegPath.Load().(string)
	if fp == "" {
		http.Error(w, "FFmpeg not yet available — downloading in background, please retry shortly", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fullArgs := append([]string{"-y"}, args...)
	cmd := exec.CommandContext(ctx, fp, fullArgs...)

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
