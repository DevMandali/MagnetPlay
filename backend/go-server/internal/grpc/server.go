package grpc_server

import (
	"fmt"
	"log"
	"net"

	"server/config"
	"server/internal/torrent"
	pb "server/proto"

	"google.golang.org/grpc"
)

func StartServer(cfg config.Config) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.GRPCPort, err)
	}

	client, err := torrent.NewClient(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}
	defer client.Close()

	repo := torrent.NewRepository(client, cfg.MetadataTimeout)
	svc := torrent.NewTorrentService(repo)

	grpcServer := grpc.NewServer()
	pb.RegisterTorrentServiceServer(grpcServer, svc)

	log.Printf("gRPC server listening on :%d", cfg.GRPCPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
