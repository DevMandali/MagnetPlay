package torrent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	lt "github.com/anacrolix/torrent"
	"google.golang.org/grpc/status"
)

type Repository struct {
	client          *lt.Client
	torrents        map[string]*TorrentInfo
	mu              sync.RWMutex
	metadataTimeout time.Duration
}

type TorrentInfo struct {
	torrent *lt.Torrent
	files   map[string]*lt.File
}

func NewRepository(client *lt.Client, metadataTimeout time.Duration) *Repository {
	return &Repository{
		client:          client,
		torrents:        make(map[string]*TorrentInfo),
		metadataTimeout: metadataTimeout,
	}
}

func (r *Repository) GetOrAdd(ctx context.Context, magnetURL string, infoHash string) (*lt.Torrent, error) {
	normalizedHash := strings.ToLower(infoHash)
	if infoHash != "" {
		r.mu.RLock()
		tInfo, exists := r.torrents[normalizedHash]
		r.mu.RUnlock()
		if exists {
			return tInfo.torrent, nil
		}
	}

	t, err := r.client.AddMagnet(magnetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to add magnet: %w", err)
	}

	// Wait for torrent info (magnet links need peer handshake to resolve metadata)
	infoCtx, cancel := context.WithTimeout(ctx, torrentInfoTimeout)
	defer cancel()

	select {
	case <-t.GotInfo():
		// metadata resolved
	case <-time.After(r.metadataTimeout):
		t.Drop()
		return nil, fmt.Errorf("timeout waiting for torrent metadata")
	case <-infoCtx.Done():
		t.Drop()
		return nil, fmt.Errorf("context cancelled while waiting for torrent metadata: %w", infoCtx.Err())
	case <-ctx.Done():
		t.Drop()
		return nil, status.FromContextError(ctx.Err()).Err()
	}

	key := t.InfoHash().HexString()
	r.mu.Lock()
	// Double-check in case another goroutine added it while we were fetching metadata
	if existingInfo, exists := r.torrents[key]; exists {
		r.mu.Unlock()
		t.Drop() // Drop duplicate
		return existingInfo.torrent, nil
	}
	r.torrents[key] = &TorrentInfo{torrent: t, files: make(map[string]*lt.File)}
	r.mu.Unlock()

	return t, nil
}

func (r *Repository) GetTorrent(infoHash string) (*lt.Torrent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tInfo, exists := r.torrents[strings.ToLower(infoHash)]
	if !exists {
		return nil, false
	}
	return tInfo.torrent, true
}

func (r *Repository) GetFile(infoHash, fileId string) (*lt.File, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tInfo, exists := r.torrents[strings.ToLower(infoHash)]
	if !exists {
		return nil, false
	}
	f, exists := tInfo.files[fileId]
	return f, exists
}

func (r *Repository) Clearup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, tInfo := range r.torrents {
		tInfo.torrent.Drop()
		delete(r.torrents, key)
	}
}
