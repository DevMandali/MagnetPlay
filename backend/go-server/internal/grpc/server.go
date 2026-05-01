package grpc_server

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"server/config"
	"server/internal/ffmpeg"
	"server/internal/hls"
	"server/internal/prowlarr"
	"server/internal/torrent"
	pb "server/proto"

	"google.golang.org/grpc"
)

func StartServer(cfg config.Config) {
	// Start Prowlarr — non-fatal
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

	// Ensure FFmpeg binaries — fatal if unavailable
	ffmpegBinDir := "./bin"
	ffmpegPath, err := ffmpeg.EnsureFFmpeg(cfg.FFmpegPath, ffmpegBinDir)
	if err != nil {
		log.Fatalf("[ffmpeg] binary unavailable: %v", err)
	}
	ffprobePath, err := ffmpeg.EnsureFFprobe(cfg.FFprobePath, ffmpegBinDir)
	if err != nil {
		log.Fatalf("[ffprobe] binary unavailable: %v", err)
	}
	log.Printf("[ffmpeg] using %s", ffmpegPath)
	log.Printf("[ffprobe] using %s", ffprobePath)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.GRPCPort, err)
	}

	client, err := torrent.NewClient(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}

	repo := torrent.NewRepository(client, cfg.DataDir, cfg.MetadataTimeout)

	defer repo.CleanupDataDir(cfg.DataDir)   // registered first → runs LAST (after client.Close)

	defer func() {                            // registered second → runs 2nd-to-last
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}
	}()

	// fileOpener serves torrent file bytes over HTTP with Range support.
	// FFmpeg uses this to seek directly to any byte position without reading
	// all preceding data (fixes seek latency on large MKV files).
	fileOpener := hls.FileOpener(func(infoHash, fileId string) (io.ReadSeekCloser, int64, error) {
		f, ok := repo.GetFile(infoHash, fileId)
		if !ok {
			return nil, 0, fmt.Errorf("file not found: %s/%s", infoHash, fileId)
		}
		r := f.NewReader()
		r.SetResponsive()
		r.SetReadahead(64 * 1024 * 1024) // 64 MB — larger readahead reduces per-segment stalls
		return r, f.Length(), nil
	})

	// filePrioritize is called by FFmpeg when it requests a byte range. We prioritize the pieces that overlap
	// the requested range to ensure FFmpeg gets the data as quickly as possible.
	filePrioritizer := func(infoHash, fileId string, start, end int64) {
		t, ok := repo.GetTorrent(infoHash)
		if !ok {
			log.Printf("torrent not found for prioritization: %s", infoHash)
			return
		}
		f, ok := repo.GetFile(infoHash, fileId)
		if !ok {
			log.Printf("file not found for prioritization: %s/%s", infoHash, fileId)
			return
		}
		torrent.Prioritize(t, f, start, end)
	}

	hlsBaseURL := fmt.Sprintf("http://localhost:%d", cfg.HLSPort)
	hlsFileBaseURL := fmt.Sprintf("http://localhost:%d/rawfile", cfg.HLSPort)

	remuxHandler := hls.NewRemuxHandler(ffmpegPath, hlsFileBaseURL, hlsBaseURL)
	subtitleHandler := hls.NewSubtitleHandler(ffmpegPath, hlsFileBaseURL)
	hlsSrv := hls.NewHLSServer(cfg.HLSPort, fileOpener, filePrioritizer, remuxHandler, subtitleHandler)

	go func() {
		if err := hlsSrv.Start(); err != nil {
			log.Printf("[media-http] server error: %v", err)
		}
	}()

	svc := torrent.NewTorrentService(repo, remuxHandler, ffprobePath, hlsFileBaseURL)

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
	remuxHandler.StopAll()
	grpcServer.GracefulStop()
	repo.Clearup()
}
