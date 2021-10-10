package main

import (
	"context"
	"time"

	api "github.com/spirifoxy/teleworker/internal/api/v1"
	tw "github.com/spirifoxy/teleworker/pkg/teleworker"
	"github.com/spirifoxy/teleworker/server/internal/auth"
)

type UnauthorizedReq struct{}

func (e *UnauthorizedReq) Error() string {
	return "request authorization error"
}

type AccessDenied struct{}

func (e *AccessDenied) Error() string {
	return "you have no rights to perform that operation"
}

func (s *TWServer) Start(ctx context.Context, req *api.StartRequest) (*api.StartResponse, error) {
	user, ok := auth.UsernameFromCtx(ctx)
	if !ok {
		return nil, &UnauthorizedReq{}
	}

	command := req.GetCommand()
	args := req.GetArgs()
	limits := &tw.Limits{
		MemoryMB:  int(req.GetMemoryLimitMb()),
		CpuWeight: int(req.GetCpuWeight()),
		IOWeight:  int(req.GetIoWeight()),
	}

	var err error
	job, err := tw.NewJob(
		command,
		args,
		tw.WithLimits(limits),
		tw.WithUsername(user.Name),
	)
	if err != nil {
		return nil, err
	}

	err = job.Start()
	if err != nil {
		return nil, err
	}

	err = s.store.Put(job)
	if err != nil {
		return nil, err
	}

	return &api.StartResponse{
		JobId: job.ID.String(),
	}, nil
}

func (s *TWServer) Stop(ctx context.Context, req *api.StopRequest) (*api.StopResponse, error) {
	user, ok := auth.UsernameFromCtx(ctx)
	if !ok {
		return nil, &UnauthorizedReq{}
	}

	id := req.GetJobId()

	var err error
	job, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}

	if user.Name != job.User {
		return nil, &AccessDenied{}
	}

	err = job.Stop()
	if err != nil {
		return nil, err
	}

	return &api.StopResponse{}, nil
}

func (s *TWServer) Status(ctx context.Context, req *api.StatusRequest) (*api.StatusResponse, error) {
	id := req.GetJobId()
	job, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}

	state := job.Status()
	return &api.StatusResponse{
		Status:             state.Status,
		MemoryLimitMb:      int32(state.Limits.MemoryMB),
		CpuLimitPercentage: int32(state.Limits.CpuWeight),
		IoLimitPercentage:  int32(state.Limits.IOWeight),
		ExitCode:           int32(state.ExitCode),
	}, nil
}

func (s *TWServer) Stream(req *api.StreamRequest, stream api.TeleWorker_StreamServer) error {
	id := req.GetJobId()
	job, err := s.store.Get(id)
	if err != nil {
		return err
	}

	var streamCh <-chan []byte
	var streamCancel context.CancelFunc

	if req.StreamErrors {
		streamCh, streamCancel = job.StreamStderr()
	} else {
		streamCh, streamCancel = job.StreamStdout()
	}
	defer streamCancel()

	for {
		select {
		case res, ok := <-streamCh:
			if !ok {
				// streamCh was closed - the broker is stopped,
				// so we can just safely return
				return nil
			}

			resp := &api.StreamResponse{
				OutStream: res,
			}

			err := stream.Send(resp)
			if err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		case <-time.After(time.Minute):
			return nil
		}
	}
}
