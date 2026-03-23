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

func toFileInfoList(infoHash string, tFiles []*lt.File, repo *Repository) []*pb.FileInfo {
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
		// If the MIME type is empty, it means we couldn't determine the type based on the extension,
		// we can log a warning and skip it
		if contentType == "" {
			f.SetPriority(lt.PiecePriorityNone) // Skip this file by setting its priority to none
			log.Printf("Warning: Unknown MIME type for file %s, skipping", f.DisplayPath())
			continue
		}

		// We want to filter out non-video files, we can do this by checking the MIME type of the file based on its extension,
		// if it doesn't start with "video/" and it's not in our list of other video types, we skip it
		if !strings.HasPrefix(contentType, "video/") && !slices.Contains(otherVideoTypes, contentType) {
			f.SetPriority(lt.PiecePriorityNone) // Skip this file by setting its priority to none
			log.Printf("Skipping and non-video file: %s, MIME type: %s", f.DisplayPath(), contentType)
			continue
		}
		log.Printf("File: %s, Extension: %s, Content-Type: %s", f.DisplayPath(), ext, contentType)

		fileId := fmt.Sprintf("%s:%d", infoHash, i)
		// We also want to ensure that we only add each file to the repository once,
		// so we can check if the file ID already exists in the repository's torrent info before adding it.
		// If it doesn't exist, we add it to the repository's torrent info
		if _, exists := repo.torrents[infoHash].files[fileId]; !exists {
			repo.torrents[infoHash].files[fileId] = f
		}

		// Finally, we create a FileInfo object for the file and add it to our list of files to return to the client
		files = append(files, &pb.FileInfo{
			Id:   fileId,
			Name: f.DisplayPath(),
			Size: f.Length(),
		})
	}
	return files
}
