package main

import (
	"context"
	"fmt"
	"log"
	"net"
	pb "server/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) SayHello(ctx context.Context, request *pb.HelloRequest) (*pb.HelloResponse, error) {
	name := request.GetName()
	// time.Sleep(time.Second * 10)
	return &pb.HelloResponse{
		Message: name + " Welcome",
	}, nil
}

func main() {
	port := 50051

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterGreeterServer(grpcServer, &server{})

	log.Printf("Go gRPC server listening on port %d", port)

	// Start serving requests (blocking call)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
