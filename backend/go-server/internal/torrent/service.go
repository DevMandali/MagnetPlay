package torrent

import (
	"context"
	"log"

	pb "server/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	return toResponse(t), nil
}
