package hls_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"server/internal/hls"
)

func TestRawFileHandler_NotFound_NoOpener(t *testing.T) {
	srv := hls.NewHLSServer(8091, nil, nil, hls.NewRemuxHandler("ffmpeg", "", ""), nil)
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/rawfile/abc123/dGVzdA", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when fileOpener nil, got %d", w.Code)
	}
}

func TestRawFileHandler_CORS(t *testing.T) {
	srv := hls.NewHLSServer(8091, nil, nil, hls.NewRemuxHandler("ffmpeg", "", ""), nil)
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/rawfile/abc123/dGVzdA", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS * header on /rawfile/")
	}
}

func TestRemuxRoute_OptionsPreflightOK(t *testing.T) {
	h := hls.NewRemuxHandler("ffmpeg", "", "http://localhost:8091")
	srv := hls.NewHLSServer(8091, nil, nil, h, nil)
	handler := srv.Handler()

	req := httptest.NewRequest("OPTIONS", "/remux/abc123/dGVzdA", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS preflight, got %d", w.Code)
	}
}

func TestRemuxRoute_BadPath(t *testing.T) {
	h := hls.NewRemuxHandler("ffmpeg", "", "http://localhost:8091")
	srv := hls.NewHLSServer(8091, nil, nil, h, nil)
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/remux/abc123/not-valid-base64!!!", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad base64 path, got %d", w.Code)
	}
}
