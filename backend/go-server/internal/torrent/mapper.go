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

func toResponse(t *lt.Torrent) *pb.TorrentResponse {
	info := t.Info()
	if info == nil {
		return nil // TODO need to rectify more gracefully, maybe return an error instead of nil
	}
	infoHash := t.InfoHash().String()

	if len(t.Files()) == 0 {
		return &pb.TorrentResponse{
			TorrentId: infoHash,
			Name:      info.Name,
			Status:    pb.TorrentStatus_SINGLE_FILE,
		}
	}

	return &pb.TorrentResponse{
		TorrentId: infoHash,
		Name:      info.Name,
		Files:     toFileInfoList(infoHash, t.Files()),
		Status:    pb.TorrentStatus_MULTI_FILE,
	}
}

func toFileInfoList(infoHash string, tFiles []*lt.File) []*pb.FileInfo {
	files := make([]*pb.FileInfo, 0, len(tFiles))

	otherVideoTypes := []string{
		"application/octet-stream",      //often used for video files without a recognized extension
		"application/ogg",               //legacy or highly specialized formats
		"application/mp4",               //legacy or highly specialized formats
		"application/x-mpegURL",         //Streaming Manifests
		"application/vnd.apple.mpegurl", //Streaming Manifests
	}

	for i, f := range tFiles {
		// Get extension (e.g., .mp4, .mkv)
		ext := filepath.Ext(f.DisplayPath())

		contentType := mime.TypeByExtension(ext)
		// If the MIME type is empty, it means we couldn't determine the type based on the extension, we can log a warning and skip it
		if contentType == "" {
			log.Printf("Warning: Unknown MIME type for file %s, skipping", f.DisplayPath())
			continue
		}

		// We want to filter out non-video files, we can do this by checking the MIME type of the file based on its extension, if it doesn't start with "video/" and it's not in our list of other video types, we skip it
		if !strings.HasPrefix(contentType, "video/") && !slices.Contains(otherVideoTypes, contentType) {
			log.Printf("Skipping non-video file: %s, MIME type: %s", f.DisplayPath(), contentType)
			continue
		}

		log.Printf("File: %s, Extension: %s, Content-Type: %s", f.DisplayPath(), ext, contentType)

		files = append(files, &pb.FileInfo{
			Id:   fmt.Sprintf("%s:%d", infoHash, i),
			Name: f.DisplayPath(),
			Size: f.Length(),
		})
	}
	return files
}
