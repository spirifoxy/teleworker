package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"testing"

	api "github.com/spirifoxy/teleworker/internal/api/v1"
	"github.com/spirifoxy/teleworker/pkg/teleworker"
	"github.com/spirifoxy/teleworker/server/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type StoreMocked struct {
	mock.Mock
}

func (m *StoreMocked) Get(id string) (*teleworker.Job, error) {

	args := m.Called(id)
	return args.Get(0).(*teleworker.Job), args.Error(1)

}

func (s *StoreMocked) Put(*teleworker.Job) error {
	return nil
}

var twServer *TWServer

func init() {
	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
	grpcServer := grpc.NewServer(
		credsOption(),
		grpc.StreamInterceptor(auth.StreamServerInterceptor(auth.CertAuthFunc)),
		grpc.UnaryInterceptor(auth.UnaryServerInterceptor(auth.CertAuthFunc)),
	)
	twServer = &TWServer{
		store: &StoreMocked{},
	}

	api.RegisterTeleWorkerServer(grpcServer, twServer)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("server exited with error: %v", err)
		}
	}()
}

func TestAuthFails(t *testing.T) {
	ctx := context.Background()
	con, err := grpc.DialContext(
		ctx,
		"localhost:50052",
		grpc.WithInsecure(),
	)

	if err != nil {
		t.Fatal(err)
	}
	defer con.Close()
	client := api.NewTeleWorkerClient(con)

	store := new(StoreMocked)
	store.On("Get", "1234").Return(teleworker.NewJob("echo", []string{"1"}))
	twServer.store = store

	_, err = client.Status(ctx, &api.StatusRequest{JobId: "1234"})
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "connection closed"))
}

func TestAuthSucceed(t *testing.T) {
	ctx := context.Background()
	con, err := grpc.DialContext(
		ctx,
		"localhost:50052",
		clientCredsOption(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer con.Close()
	client := api.NewTeleWorkerClient(con)

	store := new(StoreMocked)
	store.On("Get", "1234").Return(teleworker.NewJob("echo", []string{"1"}))
	twServer.store = store

	_, err = client.Status(ctx, &api.StatusRequest{JobId: "1234"})
	fmt.Println(err)
	assert.Nil(t, err)
}

func clientCredsOption() grpc.DialOption {
	const certPath = "../security/client-cert.pem"
	const keyPath = "../security/client-key.pem"
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
		log.Fatalln("error while parsing ca certificate")
	}
	tlsCfg.RootCAs = caPool

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
}
