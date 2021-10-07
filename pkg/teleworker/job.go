package teleworker

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	api "github.com/spirifoxy/teleworker/internal/api/v1"
	ls "github.com/spirifoxy/teleworker/internal/logstreamer"
	cg "github.com/spirifoxy/teleworker/pkg/cgroup"
)

type Limits struct {
	MemoryMB  int
	CpuWeight int
	IOWeight  int
}

// ToCgroupLimits formats limits in format acceptable as
// cgroup parameters and return them as strings
func (l *Limits) ToCgroupLimits() cg.Limits {
	formatted := cg.Limits{}
	if l.MemoryMB > 0 {
		formatted[cg.MemLimit] = fmt.Sprintf("%dM", l.MemoryMB)
	}

	if l.CpuWeight > 0 {
		formatted[cg.CpuShares] = strconv.Itoa(l.CpuWeight * 10)
	}

	if l.IOWeight > 0 {
		formatted[cg.BlkioWeight] = strconv.Itoa(l.IOWeight * 10)
	}

	return formatted
}

func (l *Limits) ToFlags() []string {
	flags := []string{}
	if l.MemoryMB > 0 {
		flags = append(flags, fmt.Sprintf("-memorymb=%d", l.MemoryMB))
	}

	if l.CpuWeight > 0 {
		flags = append(flags, fmt.Sprintf("-cpuweight=%d", l.CpuWeight))
	}

	if l.IOWeight > 0 {
		flags = append(flags, fmt.Sprintf("-ioweight=%d", l.IOWeight))
	}
	return flags
}

type JobState struct {
	Status   api.JobStatus
	ExitCode int
	ExitErr  error
	ExitedAt time.Time
	Limits   *Limits
}

type Job struct {
	ID          uuid.UUID
	UserCommand string
	UserArgs    []string

	mu        sync.RWMutex
	cmd       *exec.Cmd
	outLogger *ls.LogStreamer
	errLogger *ls.LogStreamer

	user  string
	state *JobState

	done chan struct{}
}

type Option func(*Job)

func NewJob(command string, args []string, options ...Option) (*Job, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("unexpected error generating uuid: %w", err)
	}

	j := &Job{
		ID:          uuid,
		UserCommand: command,
		UserArgs:    args,

		state: &JobState{
			Limits: &Limits{},
			Status: api.JobStatus_STARTING,
		},

		done: make(chan struct{}),
	}

	// It is easy to imagine that in some cases it is required to use
	// the library without, for example, resource limits, so some of the
	// parameters are optional and can be omitted when configuring the server
	for _, opt := range options {
		opt(j)
	}

	cmd := j.selfWrapCommand()

	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error setting up stdout logger: %w", err)
	}
	errReader, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("error setting up stderr logger: %w", err)
	}

	j.cmd = cmd
	j.outLogger = ls.NewLogStreamer(outReader)
	j.errLogger = ls.NewLogStreamer(errReader)

	return j, nil
}

// selfWrapCommand wraps the user command in the self call.
// Whenever user wants to start some task we launch another server process and
// distinguish it from the main flow using flags. Then we can use that process ID
// in order to limit the resources of user's task before we actually start it.
func (j *Job) selfWrapCommand() *exec.Cmd {
	const selfExe = "/proc/self/exe"

	jobID := fmt.Sprintf("-jobid=%s", j.ID.String())
	userCommand := fmt.Sprintf("-command=%s", j.UserCommand)
	limitFlags := j.state.Limits.ToFlags()

	callArgs := append(limitFlags, jobID)
	callArgs = append(callArgs, userCommand)
	callArgs = append(callArgs, j.UserArgs...)

	return exec.Command(selfExe, callArgs...)
}

// WithUsername sets the name of the job's creator.
// In case server doesn't want to limit tasks management based
// on the user requesting it setting the job creator can be
// skipped by omitting this call
func WithUsername(username string) Option {
	return func(j *Job) {
		j.user = username
	}
}

// WithLimits sets mem, cpu and io limits to the job.
// If resource management is not required than set up of limits
// might be omitted, which will result in all the tasks being
// created within the single root control group
func WithLimits(limits *Limits) Option {
	return func(j *Job) {
		j.state.Limits = limits
	}
}

// Limited return whether any of the resource limits were
// applied to the task upon creation
func (j *Job) Limited() bool {
	l := j.state.Limits
	return l.MemoryMB > 0 || l.CpuWeight > 0 || l.IOWeight > 0
}

func (j *Job) Active() bool {
	return j.state.Status == api.JobStatus_ALIVE
}
