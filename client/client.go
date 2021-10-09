package main

import (
	"context"
	"log"
	"time"

	"github.com/alexflint/go-arg"
	api "github.com/spirifoxy/teleworker/internal/api/v1"
	"google.golang.org/grpc"
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
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("could not connect to the server: %v", err)
	}

	return con, api.NewTeleWorkerClient(con)
}
