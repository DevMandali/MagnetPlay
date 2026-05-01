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
	req := httptest.NewRequest("GET", "/subtitle/embedded/../secret/dGVzdA/0", nil)
	w := httptest.NewRecorder()
	h.ServeEmbedded(w, req)
	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for path traversal attempt")
	}
}

func TestSubtitleFile_ContentTypeHeader(t *testing.T) {
	h := hls.NewSubtitleHandler("ffmpeg", "http://localhost:8091/rawfile")
	req := httptest.NewRequest("OPTIONS", "/subtitle/file/abc/dGVzdA", nil)
	w := httptest.NewRecorder()
	h.ServeFile(w, req)
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Origin"), "*") {
		t.Error("expected CORS * header")
	}
}
