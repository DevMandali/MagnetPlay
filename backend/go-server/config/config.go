package config

import "time"

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
	}
}
