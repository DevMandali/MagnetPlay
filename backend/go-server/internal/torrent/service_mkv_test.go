package torrent_test

import (
	"context"
	"testing"

	"server/internal/torrent"
	pb "server/proto"
)

func TestIsMKV(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"video.mkv", true},
		{"video.MKV", true},
		{"video.mp4", false},
		{"video.avi", false},
		{"", false},
	}
	for _, tc := range cases {
		got := torrent.IsMKV(tc.path)
		if got != tc.want {
			t.Errorf("IsMKV(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestStartRemux_ReturnsNotFound_UnknownTorrent(t *testing.T) {
	repo := torrent.NewTestRepository()
	svc := torrent.NewTorrentService(repo, nil, "", "")

	_, err := svc.StartRemux(context.Background(), &pb.HLSRequest{
		InfoHash: "nonexistent",
		FileId:   "nonexistent:0",
	})
	if err == nil {
		t.Fatal("expected error for unknown torrent, got nil")
	}
}
