package config

import (
	"os"
	"time"
)

type ProwlarrConfig struct {
	DataDir      string
	BinDir       string
	Port         int
	SeedIndexers bool
}

type Config struct {
	GRPCPort        int
	DataDir         string
	MetadataTimeout time.Duration
	Prowlarr        ProwlarrConfig
	FFmpegPath      string
	FFprobePath     string
	FFmpegBinDir    string
	HLSPort         int
}

func Default() Config {
	return Config{
		GRPCPort:        50051,
		DataDir:         "./downloads",
		MetadataTimeout: 60 * time.Second,
		Prowlarr: ProwlarrConfig{
			DataDir:      "./prowlarr-data",
			BinDir:       "./prowlarr",
			Port:         9696,
			SeedIndexers: true,
		},
		FFmpegBinDir: "./bin",
		HLSPort:      8091,
	}
}

// FromEnv returns Default() with path overrides from environment variables.
// Electron sets these to subdirectories of app.getPath('userData').
func FromEnv() Config {
	cfg := Default()
	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("PROWLARR_DATA_DIR"); v != "" {
		cfg.Prowlarr.DataDir = v
	}
	if v := os.Getenv("PROWLARR_BIN_DIR"); v != "" {
		cfg.Prowlarr.BinDir = v
	}
	if v := os.Getenv("FFMPEG_BIN_DIR"); v != "" {
		cfg.FFmpegBinDir = v
	}
	return cfg
}
