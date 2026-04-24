package grpc_server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"server/config"
	"server/internal/prowlarr"
	"server/internal/torrent"
	pb "server/proto"

	"google.golang.org/grpc"
)

func StartServer(cfg config.Config) {
	// Start Prowlarr — non-fatal: search unavailable if it fails
	pm := prowlarr.NewManager(
		cfg.Prowlarr.BinDir,
		cfg.Prowlarr.DataDir,
		cfg.Prowlarr.Port,
		cfg.Prowlarr.SeedIndexers,
	)
	if err := pm.Start(); err != nil {
		log.Printf("[prowlarr] startup failed: %v (search will be unavailable)", err)
	}
	defer pm.Stop()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.GRPCPort, err)
	}

	client, err := torrent.NewClient(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}
	}()

	repo := torrent.NewRepository(client, cfg.DataDir, cfg.MetadataTimeout)
	svc := torrent.NewTorrentService(repo)

	grpcServer := grpc.NewServer()
	pb.RegisterTorrentServiceServer(grpcServer, svc)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server stopped: %v", err)
			stop <- syscall.SIGTERM
		}
	}()

	<-stop
	log.Println("Shutting down...")
	repo.Clearup()
	grpcServer.GracefulStop()
}
