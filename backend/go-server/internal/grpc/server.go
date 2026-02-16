package grpc_server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

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
	defer client.Close()

	pb.RegisterTorrentServiceServer(grpcServer, torrent.NewTorrentService(client))

	log.Printf("Go gRPC server listening on port %d", port)

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	// Start serving requests (blocking call)
	if err := grpcServer.Serve(listener); err != nil {
		log.Printf("Server stopped: %v", err)
	}
}
