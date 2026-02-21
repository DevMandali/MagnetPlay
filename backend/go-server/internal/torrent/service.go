package torrent

import (
	"context"
	"log"

	pb "server/proto"
)

type TorrentService struct {
	pb.UnimplementedTorrentServiceServer
	repo *Repository
}

func NewTorrentService(repo *Repository) *TorrentService {
	return &TorrentService{repo: repo}
}

func (s *TorrentService) AddTorrent(ctx context.Context, req *pb.TorrentRequest) (*pb.TorrentResponse, error) {
	t, err := s.repo.GetOrAdd(req.GetMagnetURL(), "")
	if err != nil {
		log.Printf("Failed to get torrent: %v", err)
		return &pb.TorrentResponse{Status: pb.TorrentStatus_NOT_FOUND}, nil
	}

	log.Printf("Torrent ready: %s", t.Info().Name)
	return toResponse(t), nil
}
