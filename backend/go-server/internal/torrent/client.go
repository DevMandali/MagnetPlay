package torrent

import (
	"fmt"

	lt "github.com/anacrolix/torrent"
)

func NewClient(dataDir string) (*lt.Client, error) {
	cfg := lt.NewDefaultClientConfig()
	cfg.DataDir = dataDir
	client, err := lt.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent client: %w", err)
	}
	return client, nil
}
