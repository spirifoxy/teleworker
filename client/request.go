package main

import (
	"context"
	"fmt"
	"io"
	"log"

	api "github.com/spirifoxy/teleworker/internal/api/v1"
)

type StartCmd struct {
	Command string `arg:"required"`
	CPU     int32
	Mem     int32
	IO      int32
	Args    []string `arg:"positional"`
}
type StopCmd struct {
	UUID string `arg:"positional"`
}

type StatusCmd struct {
	UUID string `arg:"positional"`
}

type StreamCmd struct {
	Err  bool
	UUID string `arg:"positional"`
}

func (c *StartCmd) run() {
	con, client := connect()
	defer con.Close()

	ctx, cancel := timeoutCtx()
	defer cancel()

	r, err := client.Start(ctx, &api.StartRequest{
		Command:       c.Command,
		Args:          c.Args,
		CpuWeight:     c.CPU,
		IoWeight:      c.IO,
		MemoryLimitMb: c.Mem,
	})
	if err != nil {
		log.Fatalf("could not start the job: %v", err)
	}

	fmt.Println(r.GetJobId())
}

func (c *StopCmd) run() {
	con, client := connect()
	defer con.Close()

	ctx, cancel := timeoutCtx()
	defer cancel()

	_, err := client.Stop(ctx, &api.StopRequest{
		JobId: c.UUID,
	})
	if err != nil {
		log.Fatalf("could not stop the job: %v", err)
	}
}

func (c *StatusCmd) run() {
	con, client := connect()
	defer con.Close()

	ctx, cancel := timeoutCtx()
	defer cancel()

	r, err := client.Status(ctx, &api.StatusRequest{
		JobId: c.UUID,
	})
	if err != nil {
		log.Fatalf("could not get the job status: %v", err)
	}

	fmt.Println(r.String())
}

func (c *StreamCmd) run() {
	con, client := connect()
	defer con.Close()

	r, err := client.Stream(context.Background(), &api.StreamRequest{
		JobId:        c.UUID,
		StreamErrors: c.Err,
	})
	if err != nil {
		log.Fatalf("could not start streaming: %v", err)
	}

	for {
		resp, err := r.Recv()
		if err == io.EOF {
			fmt.Println("EOF")
			break
		}
		if err != nil {
			log.Fatalf("error during the stream: %v", err)
		}

		pack := resp.GetOutStream()
		fmt.Print(string(pack))
	}
}
