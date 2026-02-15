package main

import (
	"context"
	"fmt"
	"log"
	"net"
	pb "server/proto"
	"time"

	lt "github.com/anacrolix/torrent"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedTorrentServiceServer
}

func createTorrentClient(magnetURL string) (string, error) {

	// Initialize libtorrent session
	client, err := lt.NewClient(lt.NewDefaultClientConfig())
	if err != nil {
		log.Fatalf("Failed to create libtorrent client: %v", err)
		return "", err
	}
	torrent, addMagnetError := client.AddMagnet(magnetURL)
	if addMagnetError != nil {
		log.Fatalf("Failed to download Torrent: %v", err)
		return "", err
	}
	select {
	case <-torrent.GotInfo():
		// Metadata received
	case <-time.After(60 * time.Second):
		return "", fmt.Errorf("timeout waiting for torrent metadata")
	}

	info := torrent.Info()
	if info == nil {
		return "", fmt.Errorf("torrent info is nil")
	}

	name := info.Name
	log.Printf("Torrent name: %s", name)

	return name, nil
}

func (s *server) AddTorrent(ctx context.Context, request *pb.TorrentRequest) (*pb.SessionInfo, error) {
	magnetURL := request.GetMagnetURL()

	name, err := createTorrentClient(magnetURL)

	if err != nil {
		return &pb.SessionInfo{
			Message: "Failed to add torrent",
			Status:  pb.Status_STATUS_FAILED,
		}, nil
	}
	return &pb.SessionInfo{
		Message: "Torrent added successfully " + name,
		Status:  pb.Status_STATUS_PENDING,
	}, nil
}

func main() {
	port := 50051
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterTorrentServiceServer(grpcServer, &server{})

	log.Printf("Go gRPC server listening on port %d", port)

	// Start serving requests (blocking call)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
