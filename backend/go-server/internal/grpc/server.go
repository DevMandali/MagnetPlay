package grpc_server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

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
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}
	}()

	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}

	repo := torrent.NewRepository(client, cfg.MetadataTimeout)
	svc := torrent.NewTorrentService(repo)

	grpcServer := grpc.NewServer()
	pb.RegisterTorrentServiceServer(grpcServer, svc)

	// Channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Run the gRPC server in a goroutine
	go func() {
		log.Printf("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server stopped with error: %v", err)
			stop <- syscall.SIGTERM // Trigger shutdown on server error
		}
	}()

	// Wait for shutdown signal
	<-stop
	log.Println("Shutting down server...")

	// Gracefully stop the gRPC server
	grpcServer.GracefulStop()
}
