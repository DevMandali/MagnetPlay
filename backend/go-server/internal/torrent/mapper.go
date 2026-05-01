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

var subtitleExts = map[string]bool{
	".srt": true, ".vtt": true, ".ass": true, ".ssa": true, ".sub": true,
}

func toFileInfoList(infoHash string, tFiles []*lt.File, repo *Repository) []*pb.FileInfo {
	files := make([]*pb.FileInfo, 0, len(tFiles))

	otherVideoTypes := []string{
		"application/octet-stream",
		"application/ogg",
		"application/mp4",
		"application/x-mpegURL",
		"application/vnd.apple.mpegurl",
	}

	for i, f := range tFiles {
		ext := strings.ToLower(filepath.Ext(f.DisplayPath()))

		// Subtitle files — allow through with SUBTITLE tag
		if subtitleExts[ext] {
			fileId := fmt.Sprintf("%s:%d", infoHash, i)
			if _, exists := repo.torrents[infoHash].files[fileId]; !exists {
				repo.torrents[infoHash].files[fileId] = f
			}
			files = append(files, &pb.FileInfo{
				Id:       fileId,
				Name:     f.DisplayPath(),
				Size:     f.Length(),
				FileType: pb.FileType_SUBTITLE,
			})
			log.Printf("[mapper] subtitle file: %s ext=%s", f.DisplayPath(), ext)
			continue
		}

		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			f.SetPriority(lt.PiecePriorityNone)
			log.Printf("[mapper] unknown MIME type for %s, skipping", f.DisplayPath())
			continue
		}

		if !strings.HasPrefix(contentType, "video/") && !slices.Contains(otherVideoTypes, contentType) {
			f.SetPriority(lt.PiecePriorityNone)
			log.Printf("[mapper] non-video file skipped: %s MIME=%s", f.DisplayPath(), contentType)
			continue
		}

		log.Printf("[mapper] video file: %s ext=%s MIME=%s", f.DisplayPath(), ext, contentType)
		fileId := fmt.Sprintf("%s:%d", infoHash, i)
		if _, exists := repo.torrents[infoHash].files[fileId]; !exists {
			repo.torrents[infoHash].files[fileId] = f
		}
		files = append(files, &pb.FileInfo{
			Id:       fileId,
			Name:     f.DisplayPath(),
			Size:     f.Length(),
			FileType: pb.FileType_VIDEO,
		})
	}
	return files
}
