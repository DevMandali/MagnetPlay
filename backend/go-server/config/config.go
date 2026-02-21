package config

import "time"

type Config struct {
	GRPCPort        int
	DataDir         string
	MetadataTimeout time.Duration
}

func Default() Config {
	return Config{
		GRPCPort:        50051,
		DataDir:         "./downloads",
		MetadataTimeout: 60 * time.Second,
	}
}
