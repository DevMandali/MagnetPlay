package torrent

import (
	"testing"
)

func TestTorrentInfo_PauseResume(t *testing.T) {
	info := &TorrentInfo{Paused: false}
	if info.Paused {
		t.Error("want not paused at start")
	}
	info.Paused = true
	if !info.Paused {
		t.Error("want paused after set")
	}
}
