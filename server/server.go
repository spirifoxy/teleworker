package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"time"

	api "github.com/spirifoxy/teleworker/internal/api/v1"
	cg "github.com/spirifoxy/teleworker/pkg/cgroup"
	tw "github.com/spirifoxy/teleworker/pkg/teleworker"
	"github.com/spirifoxy/teleworker/server/internal/auth"
	"github.com/spirifoxy/teleworker/server/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	const port = ":50051"

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to create listener on port %s: %s", port, err)
	}

	twServer, err := NewTWServer()
	if err != nil {
		log.Fatalf("error registering internal services: %s", err)
	}

	grpcServer := grpc.NewServer(
		credsOption(),
		grpc.StreamInterceptor(auth.StreamServerInterceptor(auth.CertAuthFunc)),
		grpc.UnaryInterceptor(auth.UnaryServerInterceptor(auth.CertAuthFunc)),
	)
	api.RegisterTeleWorkerServer(grpcServer, twServer)

	grpcServer.Serve(listener)
}

func credsOption() grpc.ServerOption {
	const certPath = "../security/server-cert.pem"
	const keyPath = "../security/server-key.pem"
	const caPath = "../security/ca-cert.pem"

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Fatalf("failed to load key pair: %s", err)
	}
	tlsCfg := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	}

	ca, err := ioutil.ReadFile(caPath)
	if err != nil {
		log.Fatalf("failed to load ca certificate: %s", err)
	}
	caPool := x509.NewCertPool()
	ok := caPool.AppendCertsFromPEM(ca)
	if !ok {
		log.Fatalln("error while parsin ca certificate")
	}
	tlsCfg.ClientCAs = caPool

	return grpc.Creds(credentials.NewTLS(tlsCfg))
}
