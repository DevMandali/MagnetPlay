package torrent

import (
	"context"
	"testing"
	"time"

	pb "server/proto"
)

// newTestRepository returns a Repository with a nil torrent client.
// Safe for tests that never call GetOrAdd (i.e. no magnet fetching).
func newTestRepository() *Repository {
	return &Repository{
		client:          nil,
		torrents:        make(map[string]*TorrentInfo),
		metadataTimeout: 5 * time.Second,
	}
}

func TestGetTorrentStats_UnknownHash(t *testing.T) {
	svc := NewTorrentService(newTestRepository())
	_, err := svc.GetTorrentStats(context.Background(), &pb.GetTorrentStatsRequest{
		InfoHash: "nonexistent",
		FileId:   "nonexistent:0",
	})
	if err == nil {
		t.Error("want error for unknown torrent, got nil")
	}
}

func TestListTorrents_EmptyRepo(t *testing.T) {
	svc := NewTorrentService(newTestRepository())
	resp, err := svc.ListTorrents(context.Background(), &pb.ListTorrentsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Torrents) != 0 {
		t.Errorf("want 0 torrents, got %d", len(resp.Torrents))
	}
}
