package torrent

// NewTestRepository returns an empty Repository for use in unit tests.
// It does not start a torrent client.
func NewTestRepository() *Repository {
	return &Repository{
		torrents: make(map[string]*TorrentInfo),
		dataDir:  "",
	}
}
