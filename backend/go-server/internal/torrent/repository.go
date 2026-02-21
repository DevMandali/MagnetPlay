package torrent

import (
	"fmt"
	"sync"
	"time"

	lt "github.com/anacrolix/torrent"
)

type Repository struct {
	client          *lt.Client
	torrents        map[string]*lt.Torrent
	mu              sync.RWMutex
	metadataTimeout time.Duration
}

func NewRepository(client *lt.Client, metadataTimeout time.Duration) *Repository {
	return &Repository{
		client:          client,
		torrents:        make(map[string]*lt.Torrent),
		metadataTimeout: metadataTimeout,
	}
}

// GetOrAdd returns a cached torrent by infoHash, or adds it via magnet and waits for metadata.
func (r *Repository) GetOrAdd(magnetURL string, infoHash string) (*lt.Torrent, error) {
	if infoHash != "" {
		r.mu.RLock()
		t, exists := r.torrents[infoHash]
		r.mu.RUnlock()
		if exists {
			return t, nil
		}
	}

	t, err := r.client.AddMagnet(magnetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to add magnet: %w", err)
	}

	select {
	case <-t.GotInfo():
	case <-time.After(r.metadataTimeout):
		return nil, fmt.Errorf("timeout waiting for torrent metadata")
	}

	r.mu.Lock()
	r.torrents[t.InfoHash().HexString()] = t
	r.mu.Unlock()

	return t, nil
}
