package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"time"

	"github.com/alexflint/go-arg"
	api "github.com/spirifoxy/teleworker/internal/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var args struct {
	Start  *StartCmd  `arg:"subcommand:start"`
	Stop   *StopCmd   `arg:"subcommand:stop"`
	Status *StatusCmd `arg:"subcommand:status"`
	Stream *StreamCmd `arg:"subcommand:stream"`
}

func timeoutCtx() (context.Context, context.CancelFunc) {
	const timeout = 10 * time.Second

	return context.WithTimeout(context.Background(), timeout)
}

func main() {
	arg.MustParse(&args)

	switch {
	case args.Start != nil:
		args.Start.run()
	case args.Stop != nil:
		args.Stop.run()
	case args.Status != nil:
		args.Status.run()
	case args.Stream != nil:
		args.Stream.run()
	default:
		log.Fatalln("command is not supported")
	}
}

func connect() (*grpc.ClientConn, api.TeleWorkerClient) {
	const address = "localhost:50051"

	ctx, cancel := timeoutCtx()
	defer cancel()

	con, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithBlock(),
		credsOption(),
	)
	if err != nil {
		log.Fatalf("could not connect to the server: %v", err)
	}

	return con, api.NewTeleWorkerClient(con)
}

func credsOption() grpc.DialOption {
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
