package grpc_server

import (
	"fmt"
	"log"
	"net"

	"server/internal/torrent"
	pb "server/proto"

	"google.golang.org/grpc"
)

func StartServer() {
	port := 50051
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	// Initialize the torrent client
	// One client instance can manage multiple torrents, so we create it once and pass it to the gRPC server
	client, err := torrent.InitializeTorrentClient()
	if err != nil {
		log.Fatalf("Failed to create libtorrent client: %v", err)
	}

	pb.RegisterTorrentServiceServer(grpcServer, torrent.NewTorrentService(client))

	log.Printf("Go gRPC server listening on port %d", port)

	// Start serving requests (blocking call)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
	client.Close()
}
