package torrent

import (
	"bytes"
	"context"
	"io"
	"log"
	"mime"
	"path/filepath"
	"strings"
	"time"

	pb "server/proto"

	lt "github.com/anacrolix/torrent"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

type TorrentService struct {
	pb.UnimplementedTorrentServiceServer
	repo *Repository
}

func NewTorrentService(repo *Repository) *TorrentService {
	return &TorrentService{repo: repo}
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

	log.Printf("Torrent ready: %s", t.Info().Name)

	info := t.Info()
	if info == nil {
		return nil, status.Error(codes.Unavailable, "") // TODO need to rectify more gracefully, maybe return an error instead of nil
	}
	infoHash := t.InfoHash().String()

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

	return &pb.FileInfoResponse{
		FileName:  filepath.Base(f.DisplayPath()),
		FilePath:  f.DisplayPath(),
		TotalSize: f.Length(),
		MimeType:  mime.TypeByExtension(filepath.Ext(f.DisplayPath())),
		IsReady:   true,
	}, nil
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
	prioritize(t, f, req.GetStartByte(), req.GetEndByte())

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

func prioritize(t *lt.Torrent, f *lt.File, start, end int64) {
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
