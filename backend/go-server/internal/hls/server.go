package hls

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// FileOpener returns a seekable reader and total byte size for a torrent file.
// Each call creates a fresh reader; the caller is responsible for closing it.
type FileOpener func(infoHash, fileId string) (io.ReadSeekCloser, int64, error)

// FilePrioritize is a callback to prioritize torrent pieces for a specific byte range.
// This is called by FFmpeg when it requests a byte range. We prioritize the pieces that overlap
// the requested range to ensure FFmpeg gets the data as quickly as possible.
type FilePrioritizer func(infoHash, fileId string, start, end int64)

// HLSServer is the internal media HTTP server.
// It exposes two routes:
//   - /remux/  handled by RemuxHandler (fMP4 pipe streaming)
//   - /rawfile/ serves raw torrent bytes with Range support (FFmpeg input)
type HLSServer struct {
	port            int
	fileOpener      FileOpener
	filePrioritizer FilePrioritizer
	remuxHandler    *RemuxHandler
}

func NewHLSServer(port int, fo FileOpener, fprio FilePrioritizer, remuxHandler *RemuxHandler) *HLSServer {
	return &HLSServer{port: port, fileOpener: fo, filePrioritizer: fprio, remuxHandler: remuxHandler}
}

func (s *HLSServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/remux/", s.remuxHandler)
	mux.HandleFunc("/rawfile/", s.serveRawFile)
	return mux
}

func (s *HLSServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("[media-http] listening on %s", addr)
	return http.ListenAndServe(addr, s.Handler())
}

// serveRawFile handles:
//
//	GET /rawfile/{infoHash}/{base64RawURLFileId}
//
// Supports byte-range requests so FFmpeg can seek directly to any position
// without reading preceding data (fixes seek latency for large MKV files).
func (s *HLSServer) serveRawFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if s.fileOpener == nil {
		http.Error(w, "file serving not configured", http.StatusNotImplemented)
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/rawfile/")
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
		http.Error(w, "bad file id encoding", http.StatusBadRequest)
		return
	}
	fileId := string(fileIdBytes)

	reader, size, err := s.fileOpener(infoHash, fileId)
	if err != nil {
		log.Printf("[rawfile] open failed hash=%s fileId=%s: %v", infoHash, fileId, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer reader.Close()
	log.Printf("[rawfile] hash=%s Range=%q size=%d", infoHash, r.Header.Get("Range"), size)

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", "video/x-matroska")

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		io.CopyN(w, reader, 256*1024)
		return
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	dashIdx := strings.Index(rangeSpec, "-")
	if dashIdx < 0 {
		http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	var start, end int64
	startStr := rangeSpec[:dashIdx]
	endStr := rangeSpec[dashIdx+1:]

	if startStr != "" {
		start, _ = strconv.ParseInt(startStr, 10, 64)
	}
	if endStr != "" {
		end, _ = strconv.ParseInt(endStr, 10, 64)
	} else {
		end = size - 1
	}

	if start < 0 || end >= size || start > end {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
		http.Error(w, "range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLen := end - start + 1

	type readaheadSetter interface{ SetReadahead(int64) }
	if rs, ok := reader.(readaheadSetter); ok {
		if contentLen < 2*1024*1024 {
			rs.SetReadahead(2 * 1024 * 1024)
		}
	}

	if _, err := reader.Seek(start, io.SeekStart); err != nil {
		http.Error(w, "seek failed", http.StatusInternalServerError)
		return
	}

	// Prioritize the pieces that overlap the requested range to ensure FFmpeg gets the data as quickly as possible.
	if s.filePrioritizer != nil {
		s.filePrioritizer(infoHash, fileId, start, end)
	}

	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLen, 10))
	w.WriteHeader(http.StatusPartialContent)
	io.CopyN(w, reader, contentLen)
}
