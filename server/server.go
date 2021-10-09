package main

import (
	"log"
	"net"
	"time"

	api "github.com/spirifoxy/teleworker/internal/api/v1"
	cg "github.com/spirifoxy/teleworker/pkg/cgroup"
	tw "github.com/spirifoxy/teleworker/pkg/teleworker"
	"github.com/spirifoxy/teleworker/server/internal/storage"
	"google.golang.org/grpc"
)

type TWServer struct {
	api.UnimplementedTeleWorkerServer

	store  storage.Storage
	cgroup cg.Cgroup
}

func NewTWServer() (*TWServer, error) {
	const defaultTTL = 5 * time.Minute

	cgroup, err := cg.NewV1Runner()
	if err != nil {
		return nil, err
	}

	return &TWServer{
		store: storage.NewMemStorage(
			storage.WithTTL(defaultTTL),
		),
		cgroup: cgroup,
	}, nil
}

func main() {
	tw.InternalCallHandle()

	port := ":50051"

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to create listener on port %s: %s", port, err)
	}

	twServer, err := NewTWServer()
	if err != nil {
		log.Fatalf("error registering internal services: %s", err)
	}
	grpcServer := grpc.NewServer()
	api.RegisterTeleWorkerServer(grpcServer, twServer)

	grpcServer.Serve(listener)
}
