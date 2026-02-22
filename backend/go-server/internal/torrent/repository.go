package torrent

import (
	"context"
	"fmt"
	"strings"
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

func (r *Repository) GetOrAdd(ctx context.Context, magnetURL string, infoHash string) (*lt.Torrent, error) {
	normalizedHash := strings.ToLower(infoHash)
	if infoHash != "" {
		r.mu.RLock()
		t, exists := r.torrents[normalizedHash]
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
		t.Drop()
		return nil, fmt.Errorf("timeout waiting for torrent metadata")
	case <-ctx.Done():
		t.Drop()
		return nil, fmt.Errorf("context cancelled while waiting for torrent metadata: %w", ctx.Err())
	}

	key := t.InfoHash().HexString()
	r.mu.Lock()
	// Double-check in case another goroutine added it while we were fetching metadata
	if existing, exists := r.torrents[key]; exists {
		r.mu.Unlock()
		t.Drop() // Drop duplicate
		return existing, nil
	}
	r.torrents[key] = t
	r.mu.Unlock()

	return t, nil
}
