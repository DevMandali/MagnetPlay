package grpc_server

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"server/config"
	"server/internal/ffmpeg"
	"server/internal/hls"
	"server/internal/prowlarr"
	"server/internal/torrent"
	pb "server/proto"

	"google.golang.org/grpc"
)

func StartServer(cfg config.Config) {
	// Open the gRPC listener immediately so Electron's health check passes
	// before any optional-binary downloads begin.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.GRPCPort, err)
	}

	client, err := torrent.NewClient(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}

	repo := torrent.NewRepository(client, cfg.DataDir, cfg.MetadataTimeout)

	defer repo.CleanupDataDir(cfg.DataDir) // registered first → runs LAST (after client.Close)

	defer func() { // registered second → runs 2nd-to-last
		// 1. Drop all torrents first — releases piece storage / file handles
		for _, t := range client.Torrents() {
			t.Drop()
		}

		// 2. Give background goroutines time to release OS handles
		//    (client.Close is asynchronous internally)
		client.WaitAll() // blocks until all torrents are fully stopped

		// 3. Now safe to close
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}

		// 4. Force GC to flush any finalizers holding file descriptors
		runtime.GC()
		time.Sleep(500 * time.Millisecond) // let OS catch up
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

	// Resolve existing FFmpeg/FFprobe binaries without downloading.
	// If not present, handlers return 503 until the background download completes.
	ffmpegPath, _ := ffmpeg.ResolvePath(cfg.FFmpegPath, cfg.FFmpegBinDir)
	ffprobePath, _ := ffmpeg.ResolveFFprobePath(cfg.FFprobePath, cfg.FFmpegBinDir)

	remuxHandler := hls.NewRemuxHandler(ffmpegPath, hlsFileBaseURL, hlsBaseURL)
	subtitleHandler := hls.NewSubtitleHandler(ffmpegPath, hlsFileBaseURL)

	svc := torrent.NewTorrentService(repo, remuxHandler, ffprobePath, hlsFileBaseURL)

	// Download FFmpeg in the background on first run.
	if ffmpegPath == "" || ffprobePath == "" {
		log.Printf("[ffmpeg] binary not found — downloading in background (remux/subtitle unavailable until done)")
		go func() {
			p, err := ffmpeg.EnsureFFmpeg(cfg.FFmpegPath, cfg.FFmpegBinDir)
			if err != nil {
				log.Printf("[ffmpeg] download failed: %v", err)
				return
			}
			pp, err := ffmpeg.EnsureFFprobe(cfg.FFprobePath, cfg.FFmpegBinDir)
			if err != nil {
				log.Printf("[ffprobe] download failed: %v", err)
				return
			}
			remuxHandler.SetFFmpegPath(p)
			subtitleHandler.SetFFmpegPath(p)
			svc.SetFFprobePath(pp)
			log.Printf("[ffmpeg] ready at %s", p)
			log.Printf("[ffprobe] ready at %s", pp)
		}()
	} else {
		log.Printf("[ffmpeg] using %s", ffmpegPath)
		log.Printf("[ffprobe] using %s", ffprobePath)
	}

	// Start Prowlarr in the background — non-fatal, search unavailable until done.
	pm := prowlarr.NewManager(
		cfg.Prowlarr.BinDir,
		cfg.Prowlarr.DataDir,
		cfg.Prowlarr.Port,
		cfg.Prowlarr.SeedIndexers,
	)
	go func() {
		if err := pm.Start(); err != nil {
			log.Printf("[prowlarr] startup failed: %v (search will be unavailable)", err)
		}
	}()
	defer pm.Stop()

	hlsSrv := hls.NewHLSServer(cfg.HLSPort, fileOpener, filePrioritizer, remuxHandler, subtitleHandler)

	go func() {
		if err := hlsSrv.Start(); err != nil {
			log.Printf("[media-http] server error: %v", err)
		}
	}()

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
