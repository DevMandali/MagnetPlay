package hls_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"server/internal/hls"
)

func TestRemux_ProducesMP4Stream(t *testing.T) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		localBin := filepath.Join("..", "..", "..", "bin", "ffmpeg.exe")
		if _, statErr := os.Stat(localBin); statErr == nil {
			ffmpegPath = localBin
		} else {
			t.Skip("ffmpeg not in PATH — skipping integration test")
		}
	}

	fixturePath := "testdata/sample_10s.mkv"
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	// Serve fixture over HTTP (simulates the rawfile server)
	fixtureSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, fixturePath)
	}))
	defer fixtureSrv.Close()

	time.Sleep(10 * time.Millisecond)

	h := hls.NewRemuxHandler(ffmpegPath, fixtureSrv.URL, "http://localhost:8091")

	// Build args the same way ServeHTTP would
	fileURL := fixtureSrv.URL + "/sample_10s.mkv"
	args := hls.BuildRemuxArgs(0.0, fileURL, "copy", "aac")

	cmd := exec.Command(ffmpegPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	buf := make([]byte, 8)
	if _, err := io.ReadFull(stdout, buf); err != nil {
		t.Fatalf("read first 8 bytes: %v", err)
	}

	// fMP4 starts with ftyp box: size(4 bytes) + "ftyp"
	if string(buf[4:8]) != "ftyp" {
		t.Errorf("expected ftyp box at offset 4, got %q (full header: %x)", buf[4:8], buf)
	}

	cmd.Process.Kill()
	cmd.Wait()

	// RemuxHandler itself should construct valid URLs
	url := h.GetRemuxBaseURL("abc", "abc:0", "copy", "aac")
	if url == "" {
		t.Error("GetRemuxBaseURL returned empty string")
	}
}
