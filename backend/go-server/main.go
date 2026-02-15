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
	client *lt.Client
	pb.UnimplementedTorrentServiceServer
}

func createTorrentClient(client *lt.Client, magnetURL string) (string, error) {
	torrent, addMagnetError := client.AddMagnet(magnetURL)
	if addMagnetError != nil {
		fmt.Errorf("Failed to download Torrent: %v", addMagnetError)
		return "", addMagnetError
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

	// tReader := torrent.NewReader()

	// Returns the total number of pieces in the torrent. This is a fixed value that does not change as pieces are downloaded.
	log.Printf("%v", torrent.NumPieces())

	// Returns the length of each piece in bytes. This is a fixed value that does not change as pieces are downloaded.
	log.Printf("%v", torrent.Info().PieceLength)

	// torrent.AllowDataDownload()

	// torrent.DownloadPieces(0, 5) // Start downloading all pieces of the torrent

	// defer tReader.Close()

	// tReader.SetReadahead(torrent.Info().PieceLength * 5) // Set readahead to 10 MB;

	// buffer := make([]byte, torrent.Info().PieceLength) // 128 KB buffer

	// for i := 0; i < 5; i++ {
	// 	n, err := tReader.Read(buffer)
	// 	if err != nil {
	// 		log.Printf("Error reading torrent data: %v", err)
	// 		break
	// 	}
	// 	log.Printf("Read %d bytes of torrent data", n)
	// }

	files := []string{}
	for _, file := range torrent.Files() {
		// file.Torrent().DownloadPieces(int(file.Offset()/torrent.Info().PieceLength), int((file.Offset()+file.Length())/torrent.Info().PieceLength))
		files = append(files, file.Path())
		log.Printf("File: %s, Size: %d bytes", file.Path(), file.Length())
	}

	name := info.Name
	log.Printf("Torrent name: %s", name)

	return name, nil
}

func (s *server) AddTorrent(ctx context.Context, request *pb.TorrentRequest) (*pb.SessionInfo, error) {
	magnetURL := request.GetMagnetURL()

	name, err := createTorrentClient(s.client, magnetURL)

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

	// Initialize libtorrent session
	clientConfig := lt.NewDefaultClientConfig()
	clientConfig.DataDir = "./downloads" // Set the directory where torrents will be downloaded
	client, err := lt.NewClient(clientConfig)
	if err != nil {
		log.Fatalf("Failed to create libtorrent client: %v", err)
	}

	pb.RegisterTorrentServiceServer(grpcServer, &server{client: client})

	log.Printf("Go gRPC server listening on port %d", port)

	// Start serving requests (blocking call)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
