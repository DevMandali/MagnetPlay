package torrent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"mime"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"server/internal/hls"
	pb "server/proto"

	lt "github.com/anacrolix/torrent"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

const (
	// chunkSize controls how many bytes are sent per gRPC message.
	// 256 KB is a good balance for streaming video - it's small enough to keep latency low, but large enough
	// to be efficient and not overwhelm the network with too many small messages.
	chunkSize = 256 * 1024

	// readahead tells anacrolix how far ahead to pre-fetch torrent pieces.
	// 20 MB is a good starting point for 1080p video streaming - it provides enough buffer to handle
	// network variability without consuming too much memory.
	readahead = 20 * 1024 * 1024

	// torrentInfoTimeout is how long we wait for magnet metadata to resolve.
	torrentInfoTimeout = 180 * time.Second
)

type probeResult struct {
	DurationSec float64
	AudioTracks []*pb.AudioTrack
	VideoCodec  string // e.g. "h264", "hevc", "av1"
	AudioCodec  string // first audio track codec, e.g. "aac", "eac3", "ac3"
	Subtitles   []SubtitleTrack
}

// IsMKV returns true if the file path has a .mkv extension (case-insensitive).
func IsMKV(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".mkv"
}

type SubtitleTrack struct {
	Index    int
	Language string
	Codec    string
	Title    string
}

type TorrentService struct {
	pb.UnimplementedTorrentServiceServer
	repo           *Repository
	speedTrackers  map[string]*SpeedTracker
	trackerMu      sync.Mutex
	remuxHandler   *hls.RemuxHandler
	ffprobePath    string
	hlsFileBaseURL string // e.g. "http://localhost:8091/rawfile"
	probeCache     map[string]*probeResult
	probeMu        sync.RWMutex
}

func NewTorrentService(repo *Repository, remuxHandler *hls.RemuxHandler, ffprobePath string, hlsFileBaseURL string) *TorrentService {
	return &TorrentService{
		repo:           repo,
		speedTrackers:  make(map[string]*SpeedTracker),
		remuxHandler:   remuxHandler,
		ffprobePath:    ffprobePath,
		hlsFileBaseURL: hlsFileBaseURL,
		probeCache:     make(map[string]*probeResult),
	}
}

func (s *TorrentService) AddTorrent(ctx context.Context, req *pb.TorrentRequest) (*pb.TorrentResponse, error) {
	magnetUrl := req.GetMagnetUrl()
	if magnetUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "magnet URL is required")
	}
	t, err := s.repo.GetOrAdd(ctx, magnetUrl, "")
	if err != nil {
		log.Printf("Failed to get torrent: %v", err)
		return &pb.TorrentResponse{Status: pb.TorrentStatus_NOT_FOUND}, nil
	}

	info := t.Info()
	if info == nil {
		return nil, status.Error(codes.Unavailable, "") // TODO need to rectify more gracefully, maybe return an error instead of nil
	}
	log.Printf("Torrent ready: %s", info.Name)
	infoHash := t.InfoHash().String()

	var totalSize int64
	for _, f := range t.Files() {
		totalSize += f.Length()
	}
	log.Printf("[torrent] name=%s infoHash=%s files=%d totalSize=%d",
		info.Name, infoHash, len(t.Files()), totalSize)

	var fileInfo []*pb.FileInfo = toFileInfoList(infoHash, t.Files(), s.repo)

	return &pb.TorrentResponse{
		TorrentId: infoHash,
		Name:      info.Name,
		Files:     fileInfo,
		Status:    pb.TorrentStatus_MULTI_FILE,
	}, nil
}

func (s *TorrentService) GetFileInfo(ctx context.Context, req *pb.FileInfoRequest) (*pb.FileInfoResponse, error) {
	t, exists := s.repo.GetTorrent(req.GetInfoHash())
	if !exists {
		return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.GetInfoHash())
	}

	// Ensure the torrent has metadata and files are available
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		return nil, status.FromContextError(ctx.Err()).Err()
	}

	f, filePrsent := s.repo.GetFile(req.GetInfoHash(), req.GetFileId())
	if !filePrsent {
		return nil, status.Errorf(codes.NotFound, "file not found: %s", req.GetFileId())
	}

	// Prioritize this file for download since the client is requesting info about it -
	// likely to stream soon. This helps ensure the file will be ready faster,
	// especially for large multi-file torrents where pieces are shared across files.
	f.Torrent().SetMaxEstablishedConns(80)
	f.SetPriority(lt.PiecePriorityReadahead) // Optional: prioritize pieces for this file to speed up availability

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

	return resp, nil
}

func (s *TorrentService) StreamFile(req *pb.StreamRequest, stream pb.TorrentService_StreamFileServer) error {
	log.Println("[STREAMING]----------------------------------")
	ctx := stream.Context()

	t, isTorrentPresent := s.repo.GetTorrent(req.GetTorrentId())
	if !isTorrentPresent {
		log.Println("[STREAMING] closed with error ----------------------------------")
		return status.Errorf(codes.NotFound, "torrent not found: %s", req.GetTorrentId())
	}

	// Ensure the torrent has metadata and files are available
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		return status.FromContextError(ctx.Err()).Err()
	}

	f, isFilePresent := s.repo.GetFile(req.GetTorrentId(), req.GetFileId())
	if !isFilePresent {
		log.Println("[STREAMING] closed with error ----------------------------------")
		return status.Errorf(codes.NotFound, "file not found: %s", req.GetFileId())
	}

	// ── Set up anacrolix reader ───────────────────────────────────────────────
	reader := f.NewReader()
	defer reader.Close()

	// SetResponsive tells anacrolix to prioritize downloading the pieces
	// that the reader currently needs, rather than sequential pre-fetch.
	// This is critical for seek support — without it, seeking to the middle
	// of a file would stall until all preceding pieces are downloaded.
	reader.SetResponsive()

	// SetReadahead tells anacrolix how far ahead to pre-fetch pieces beyond
	// the current read position. 20 MB ≈ 5-10 seconds of 1080p video at typical bitrates, providing a good buffer
	// for smooth playback without consuming too much memory.
	reader.SetReadahead(readahead)

	log.Print("[READER] responsive + readahead=20MB configured for optimal streaming performance")

	// ── Resolve byte range ────────────────────────────────────────────────────
	startByte := req.GetStartByte()
	endByte := req.GetEndByte()
	if endByte == -1 || endByte == 0 {
		endByte = f.Length()
	}

	if startByte < 0 || startByte >= f.Length() {
		log.Println("[STREAMING] closed with error ----------------------------------")
		return status.Errorf(codes.InvalidArgument,
			"start_byte %d out of range [0, %d)", startByte, f.Length())
	}

	if endByte > f.Length() {
		endByte = f.Length()
	}

	// Seek to requested start position (handles video seeking)
	if startByte > 0 {
		if _, err := reader.Seek(startByte, io.SeekStart); err != nil {
			log.Println("[STREAMING] closed with error ----------------------------------")
			return status.Errorf(codes.Internal, "seek to %d failed: %v", startByte, err)
		}
	}

	// ── Prioritize pieces for requested byte range ─────────────────────────────
	log.Println("[PRIORITY] prioritizing pieces for requested byte range...")
	Prioritize(t, f, startByte, endByte)
	log.Printf("streaming %s | range [%d, %d] | total %d bytes",
		f.DisplayPath(), startByte, endByte, f.Length())

	// ── Stream file content in chunks ─────────────────────────────────────────
	buf := make([]byte, chunkSize)
	offset := startByte

	for offset < endByte {
		// Respect client cancelllation - user closed player, seeked away, etc.
		if err := ctx.Err(); err != nil {
			log.Println("[STREAMING] closed with client cancelled ----------------------------------")
			log.Printf("client cancelled stream at offset %d: %v", offset, err)
			return nil
		}

		// Read up to chunkSize bytes, but dont exceed the requested endByte
		remaining := endByte - offset
		toRead := int64(chunkSize)
		if remaining < toRead {
			toRead = remaining
		}

		// n, readErr := reader.Read(buf[:toRead])
		n, readErr := io.ReadFull(reader, buf[:toRead])
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			if strings.Contains(readErr.Error(), "resync") {
				time.Sleep(500 * time.Millisecond) // anacrolix may return a "resync" error if the piece isn't fully available yet - this is expected during streaming as pieces are being downloaded.
				continue                           // retry
			}
			//Distinguish between client disconnect and actualky read errors
			if ctx.Err() != nil {
				return nil
			}
			return status.Errorf(codes.Internal,
				"read error at offset %d: %v", offset, readErr)
		}

		if n > 0 {
			if bytes.Contains(buf[:n], []byte("moov")) {
				log.Printf("[MOOV] found at offset %d | bytes: %d | range: [%d, %d]", offset, n, startByte, endByte)
			}
		}

		log.Printf("read %d bytes at offset %d (requested range [%d, %d])", n, offset, startByte, endByte)
		if n > 0 {
			isLast := offset+int64(n) >= endByte
			if sendErr := stream.Send(&pb.FileChunk{
				Data:   buf[:n],
				Offset: offset,
				IsLast: isLast,
			}); sendErr != nil {
				// Client disconnected mid-stream - normal during seeks
				log.Printf("send error at offset %d (client likely disconnected): %v", offset, sendErr)
				return nil
			}
			offset += int64(n)
		}
	}
	log.Printf("[COMPLETE] stream complete for %s | sent %d bytes", f.DisplayPath(), offset-startByte)
	return nil
}

func (s *TorrentService) GetTorrentStats(ctx context.Context, req *pb.GetTorrentStatsRequest) (*pb.GetTorrentStatsResponse, error) {
	f, ok := s.repo.GetFile(req.GetInfoHash(), req.GetFileId())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "file not found: %s/%s", req.GetInfoHash(), req.GetFileId())
	}

	t, ok := s.repo.GetTorrent(req.GetInfoHash())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.GetInfoHash())
	}

	downloaded := f.BytesCompleted()
	total := f.Length()
	var pct float64
	if total > 0 {
		pct = float64(downloaded) / float64(total) * 100
	}

	key := req.GetInfoHash() + "|" + req.GetFileId()
	s.trackerMu.Lock()
	tr, ok := s.speedTrackers[key]
	if !ok {
		tr = NewSpeedTracker(5)
		s.speedTrackers[key] = tr
	}
	s.trackerMu.Unlock()
	tr.Record(downloaded, time.Now())

	ts := t.Stats()
	mi := t.Metainfo()
	trackerCount := 0
	for _, tier := range mi.AnnounceList {
		trackerCount += len(tier)
	}

	return &pb.GetTorrentStatsResponse{
		Stats: &pb.TorrentFileStats{
			FileId:           req.GetFileId(),
			TotalSize:        total,
			DownloadedBytes:  downloaded,
			DownloadSpeedBps: tr.SpeedBps(),
			CompletionPct:    pct,
			Seeders:          int32(ts.ConnectedSeeders),
			Peers:            int32(ts.ActivePeers),
			Trackers:         int32(trackerCount),
		},
	}, nil
}

func (s *TorrentService) ListTorrents(ctx context.Context, req *pb.ListTorrentsRequest) (*pb.ListTorrentsResponse, error) {
	all := s.repo.ListTorrents()
	items := make([]*pb.TorrentListItem, 0, len(all))
	for _, info := range all {
		t := info.Torrent()
		var totalSize, downloaded int64
		var files []*pb.FileInfo
		for id, f := range info.Files() {
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
		infoHash := t.InfoHash().HexString()
		s.trackerMu.Lock()
		tr, ok := s.speedTrackers[infoHash]
		if !ok {
			tr = NewSpeedTracker(5)
			s.speedTrackers[infoHash] = tr
		}
		s.trackerMu.Unlock()
		tr.Record(downloaded, time.Now())
		items = append(items, &pb.TorrentListItem{
			TorrentId:        infoHash,
			Name:             t.Name(),
			State:            state,
			TotalSize:        totalSize,
			DownloadedBytes:  downloaded,
			CompletionPct:    pct,
			DownloadSpeedBps: tr.SpeedBps(),
			Files:            files,
		})
	}
	return &pb.ListTorrentsResponse{Torrents: items}, nil
}

func (s *TorrentService) PauseTorrent(ctx context.Context, req *pb.PauseTorrentRequest) (*pb.PauseTorrentResponse, error) {
	info, err := s.repo.GetTorrentInfo(req.GetInfoHash())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.GetInfoHash())
	}
	if info.Paused {
		return &pb.PauseTorrentResponse{Success: true, Message: "already paused"}, nil
	}
	for _, f := range info.Files() {
		f.SetPriority(lt.PiecePriorityNone)
	}
	// Drop all peer connections — SetPriority alone doesn't stop in-flight downloads.
	// Peers keep pushing pieces they've already been asked for until disconnected.
	info.Torrent().SetMaxEstablishedConns(0)
	s.repo.SetPaused(req.GetInfoHash(), true)
	return &pb.PauseTorrentResponse{Success: true, Message: "paused"}, nil
}

func (s *TorrentService) ResumeTorrent(ctx context.Context, req *pb.ResumeTorrentRequest) (*pb.ResumeTorrentResponse, error) {
	info, err := s.repo.GetTorrentInfo(req.GetInfoHash())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.GetInfoHash())
	}
	if !info.Paused {
		return &pb.ResumeTorrentResponse{Success: true, Message: "already active"}, nil
	}
	// Restore peer connections before re-enabling piece downloads.
	info.Torrent().SetMaxEstablishedConns(80)
	for _, f := range info.Files() {
		f.Download()
	}
	s.repo.SetPaused(req.GetInfoHash(), false)
	return &pb.ResumeTorrentResponse{Success: true, Message: "resumed"}, nil
}

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

func (s *TorrentService) getOrProbe(infoHash, fileId string, _ *lt.Torrent, f *lt.File) *probeResult {
	key := infoHash + ":" + fileId
	s.probeMu.RLock()
	if cached, ok := s.probeCache[key]; ok {
		s.probeMu.RUnlock()
		log.Printf("[ffprobe] cache hit for fileId=%s duration=%.1fs audio=%d", fileId, cached.DurationSec, len(cached.AudioTracks))
		return cached
	}
	s.probeMu.RUnlock()

	if s.ffprobePath == "" {
		log.Printf("[ffprobe] skipped — ffprobePath not configured")
		return nil
	}

	log.Printf("[ffprobe] probing fileId=%s file=%s", fileId, filepath.Base(f.DisplayPath()))
	reader := f.NewReader()
	reader.SetResponsive()
	reader.SetReadahead(readahead)
	defer reader.Close()

	result := runFFprobe(s.ffprobePath, reader)
	if result == nil {
		log.Printf("[ffprobe] probe failed for fileId=%s", fileId)
		return nil
	}

	log.Printf("[ffprobe] probe ok fileId=%s duration=%.1fs audio_tracks=%d", fileId, result.DurationSec, len(result.AudioTracks))
	s.probeMu.Lock()
	s.probeCache[key] = result
	s.probeMu.Unlock()
	return result
}

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

func (s *TorrentService) StartRemux(ctx context.Context, req *pb.HLSRequest) (*pb.HLSResponse, error) {
	log.Printf("[remux] StartRemux hash=%s fileId=%s seek=%.1fs", req.GetInfoHash(), req.GetFileId(), req.GetSeekTimeSec())
	if s.remuxHandler == nil {
		return &pb.HLSResponse{Success: false}, status.Error(codes.Unavailable, "remux handler not configured")
	}

	t, ok := s.repo.GetTorrent(req.GetInfoHash())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "torrent not found: %s", req.GetInfoHash())
	}

	f, ok := s.repo.GetFile(req.GetInfoHash(), req.GetFileId())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "file not found: %s", req.GetFileId())
	}

	probe := s.getOrProbe(req.GetInfoHash(), req.GetFileId(), t, f)

	// Prefer stream copy when source codec is browser-safe; otherwise transcode.
	videoCodec := "libx264"
	audioCodec := "aac"
	if probe != nil {
		if strings.EqualFold(probe.VideoCodec, "h264") {
			videoCodec = "copy"
		}
		if strings.EqualFold(probe.AudioCodec, "aac") {
			audioCodec = "copy"
		}
		log.Printf("[remux] codec decision vc=%s ac=%s (src video=%s audio=%s)",
			videoCodec, audioCodec, probe.VideoCodec, probe.AudioCodec)
	}

	streamURL := s.remuxHandler.GetRemuxBaseURL(req.GetInfoHash(), req.GetFileId(), videoCodec, audioCodec)
	log.Printf("[remux] StartRemux ok streamUrl=%s", streamURL)

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
}

func Prioritize(t *lt.Torrent, f *lt.File, start, end int64) {
	// anacrolix doesn't have a built-in way to prioritize specific byte ranges,
	// but we can achieve this by prioritizing the pieces that overlap the range.
	pieceLen := t.Info().PieceLength

	fileStart := f.Offset()
	fileEnd := fileStart + f.Length()

	startPiece := int((fileStart + start) / pieceLen)
	endPiece := int((fileStart + end) / pieceLen)

	log.Printf("[PRIORITY] starting piece: %d | ending piece: %d", startPiece, endPiece)

	// 🔥 Current Playback
	// Prioritize pieces overlapping the requested byte range as "NOW" to ensure they are downloaded immediately.
	// This is critical for smooth playback, especially when seeking to a new position in the video.
	for i := startPiece; i <= endPiece; i++ {
		t.Piece(i).SetPriority(lt.PiecePriorityNext)
		// log.Printf("[PRIORITY] piece=%d -> NOW", i)
	}

	// ⚡ Beginning of file (moov atom for mp4) - needed for playback to start
	for i := 0; i < 5; i++ {
		t.Piece(i).SetPriority(lt.PiecePriorityNow)
		// log.Printf("[PRIORITY] piece=%d -> NEXT (start)/moov", i)
	}

	// ⚡ End of file - often contains metadata needed for playback, especially for formats like MP4.
	// Prioritizing the last few pieces can help ensure that the player can access necessary metadata
	// for seeking and smooth playback, even if the user jumps to the end of the video.
	lastPiece := int(fileEnd / pieceLen)
	for i := lastPiece - 5; i <= lastPiece; i++ {
		if i >= 0 {
			t.Piece(i).SetPriority(lt.PiecePriorityNow)
			// log.Printf("[PRIORITY] piece=%d -> NEXT (end/moov)", i)
		}
	}
}
