package torrent

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	lt "github.com/anacrolix/torrent"
	"google.golang.org/grpc/status"
)

type Repository struct {
	client          *lt.Client
	dataDir         string
	torrents        map[string]*TorrentInfo
	mu              sync.RWMutex
	metadataTimeout time.Duration
	pendingDelete   []string // dirs that failed runtime delete; removed on shutdown
	deleteMu        sync.Mutex
}

type TorrentInfo struct {
	torrent *lt.Torrent
	files   map[string]*lt.File
	Paused  bool // true when all file priorities set to None
}

// Torrent returns the underlying anacrolix torrent handle.
func (ti *TorrentInfo) Torrent() *lt.Torrent { return ti.torrent }

// Files returns a snapshot copy of the file map (safe for iteration outside lock).
func (ti *TorrentInfo) Files() map[string]*lt.File {
	cp := make(map[string]*lt.File, len(ti.files))
	for k, v := range ti.files {
		cp[k] = v
	}
	return cp
}
func NewRepository(client *lt.Client, dataDir string, metadataTimeout time.Duration) *Repository {
	return &Repository{
		client:          client,
		dataDir:         dataDir,
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
		// t.Drop()
		return nil, fmt.Errorf("timeout waiting for torrent metadata")
	case <-infoCtx.Done():
		// t.Drop()
		return nil, fmt.Errorf("context cancelled while waiting for torrent metadata: %w", infoCtx.Err())
	case <-ctx.Done():
		// t.Drop()
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

// QueueDelete schedules a directory to be removed on shutdown, used when
// runtime deletion fails due to Windows file handle locks on .part files.
func (r *Repository) QueueDelete(dir string) {
	r.deleteMu.Lock()
	r.pendingDelete = append(r.pendingDelete, dir)
	r.deleteMu.Unlock()
	log.Printf("[delete] queued for shutdown cleanup: %s", dir)
}

func (r *Repository) Clearup() {
	r.mu.Lock()
	for key, tInfo := range r.torrents {
		tInfo.torrent.Drop()
		delete(r.torrents, key)
	}
	r.mu.Unlock()

	// All torrents are now dropped — OS file handles on .part files are released.
	// Safe to delete any dirs that failed during runtime.
	r.deleteMu.Lock()
	pending := r.pendingDelete
	r.pendingDelete = nil
	r.deleteMu.Unlock()

	for _, dir := range pending {
		log.Printf("[shutdown] removing deferred %s", dir)
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("[shutdown] failed to remove %s: %v", dir, err)
		} else {
			log.Printf("[shutdown] removed %s", dir)
		}
	}
}

// GetTorrentInfo returns the full TorrentInfo for an info hash, or an error if not found.
func (r *Repository) GetTorrentInfo(infoHash string) (*TorrentInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tInfo, exists := r.torrents[strings.ToLower(infoHash)]
	if !exists {
		return nil, fmt.Errorf("torrent not found: %s", infoHash)
	}
	return tInfo, nil
}

// ListTorrents returns a snapshot of all tracked torrents.
func (r *Repository) ListTorrents() []*TorrentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*TorrentInfo, 0, len(r.torrents))
	for _, info := range r.torrents {
		result = append(result, info)
	}
	return result
}

// SetPaused updates the Paused flag for a torrent.
func (r *Repository) SetPaused(infoHash string, paused bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.torrents[strings.ToLower(infoHash)]; ok {
		info.Paused = paused
	}
}

// Remove drops tracking of a torrent (does NOT call Drop on the torrent itself).
func (r *Repository) Remove(infoHash string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.torrents, strings.ToLower(infoHash))
}
