package main

import (
	"io"
	"log"
	"os"

	"gopkg.in/lumberjack.v2"

	"server/config"
	server "server/internal/grpc"
)

func main() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatalf("cannot create logs dir: %v", err)
	}
	roller := &lumberjack.Logger{
		Filename:   "logs/go-server.log",
		MaxSize:    20,   // MB before rolling
		MaxBackups: 7,    // rotated files to keep
		MaxAge:     14,   // days
		Compress:   true,
	}
	log.SetOutput(io.MultiWriter(os.Stdout, roller))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("[startup] logging to stdout + logs/go-server.log")

	server.StartServer(config.FromEnv())
}
