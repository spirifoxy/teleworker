package teleworker

import (
	"context"
	"fmt"
	"time"

	api "github.com/spirifoxy/teleworker/internal/api/v1"
	cg "github.com/spirifoxy/teleworker/pkg/cgroup"
)

func (j *Job) Start() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.state.Status != api.JobStatus_STARTING {
		return fmt.Errorf("not possible to start the job: unexpected status %s on start", j.state.Status.String())
	}

	err := j.cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting the task: %w", err)
	}
	j.state.Status = api.JobStatus_ALIVE

	go j.wait()
	return nil
}

func (j *Job) wait() {
	err := j.cmd.Wait()

	j.mu.Lock()
	exitCode := j.cmd.ProcessState.ExitCode()
	if err != nil && exitCode != -1 {
		j.state.ExitErr = err
	}

	j.state.ExitCode = exitCode
	j.state.Status = api.JobStatus_FINISHED
	j.mu.Unlock()

	j.outLogger.Close()
	j.errLogger.Close()
	close(j.done)
}

func (j *Job) Stop() error {
	j.mu.Lock()
	if j.state.Status != api.JobStatus_ALIVE {
		j.mu.Unlock()
		return fmt.Errorf("not possible to stop the job as it's not alive; please check the status")
	}

	err := j.cmd.Process.Kill()
	if err != nil {
		j.mu.Unlock()
		return fmt.Errorf("not possible to stop the task: %w", err)
	}
	j.mu.Unlock() // Unlock here in order not to lock forever in the wait call

	// Wait for the goroutine launched upon the task creation to finish.
	// Then override the status from finished to stopped
	select {
	case <-j.done:
		j.mu.Lock()
		defer j.mu.Unlock()

		j.state.Status = api.JobStatus_STOPPED
		j.state.ExitedAt = time.Now()

		if j.state.ExitErr != nil {
			return fmt.Errorf("error while trying to stop the task: %w", j.state.ExitErr)
		}

		err := j.tryRemovingCgroup()
		if err != nil {
			return err
		}
	case <-time.After(10 * time.Second):
		return fmt.Errorf("error while trying to stop the task: timeout exceeded")
	}

	return nil
}

// tryRemovingCgroup makes several attempts on
// cleaning up the job cgroup directory.
// See cgroup.Remove for more details
func (j *Job) tryRemovingCgroup() error {
	if !j.Limited() {
		return nil
	}

	cgroup := cg.NewV1Service()
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err = cgroup.Remove(j.ID.String())
			if err == nil {
				return err
			}
		case <-ctx.Done():
			return err
		}
	}
}

func (j *Job) Status() *JobState {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return j.state
}

func (j *Job) StreamStdout(ctx context.Context) <-chan []byte {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return j.outLogger.Stream(j.Active(), ctx)
}

func (j *Job) StreamStderr(ctx context.Context) <-chan []byte {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return j.errLogger.Stream(j.Active(), ctx)
}
