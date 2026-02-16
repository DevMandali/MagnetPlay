package torrent

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "server/proto"

	lt "github.com/anacrolix/torrent"
)

func InitializeTorrentClient() (*lt.Client, error) {
	clientConfig := lt.NewDefaultClientConfig()
	clientConfig.DataDir = "./downloads"
	client, err := lt.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create libtorrent client: %v", err)
	}
	return client, nil
}

type TorrentService struct {
	client *lt.Client
	pb.UnimplementedTorrentServiceServer
}

func NewTorrentService(client *lt.Client) *TorrentService {
	return &TorrentService{
		client: client,
	}
}

func (s *TorrentService) AddTorrent(ctx context.Context, request *pb.TorrentRequest) (*pb.TorrentResponse, error) {
	magnetURL := request.GetMagnetURL()

	torrent, addMagnetError := s.client.AddMagnet(magnetURL)
	if addMagnetError != nil {
		return nil, fmt.Errorf("Failed to download Torrent: %v", addMagnetError)
	}
	select {
	case <-torrent.GotInfo():
		// Metadata received
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("timeout waiting for torrent metadata")
	}

	info := torrent.Info()

	// Get torrent name
	name := info.Name
	log.Printf("Torrent name: %s", name)

	// Get list of files in the torrent and their sizes
	torrentFiles := info.Files

	// If fiels are empty, it might be a single-file torrent
	if len(torrentFiles) == 0 {
		return &pb.TorrentResponse{
			Name:   name,
			Status: pb.TorrentStatus_SINGLE_FILE,
		}, nil
	}

	// If there are files, it's a multi-file torrent. We need to extract the file paths and sizes
	fileInfoList := []*pb.FileInfo{}
	for _, file := range torrentFiles {
		fileInfoList = append(fileInfoList, &pb.FileInfo{
			Name: file.DisplayPath(info),
			Size: file.Length,
		})
		log.Printf("File: %s, Size: %d bytes", file.Path, file.Length)
	}

	return &pb.TorrentResponse{
		Name:   name,
		Files:  fileInfoList,
		Status: pb.TorrentStatus_MULTI_FILE,
	}, nil
}
