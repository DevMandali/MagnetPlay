package main

import (
	"server/config"
	server "server/internal/grpc"
)

func main() {
	server.StartServer(config.Default())
}
