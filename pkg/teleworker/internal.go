package teleworker

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/alexflint/go-arg"
	cg "github.com/spirifoxy/teleworker/pkg/cgroup"
)

// InternalCallHandle is required to be used by the server
// in order to process the user's commands launches. The only
// thing required is to call this method in the beginning of
// the server main function and then proceed with setting up
// grpc as usual, no additional setup needed.
// See Job.selfWrapCommand for more details
func InternalCallHandle() {
	var internal struct {
		Command string `arg:"required"`
		JobID   string `arg:"required"`
		Limits
		Args []string `arg:"positional"`
	}
	err := arg.Parse(&internal)
	if err != nil {
		fmt.Println(err)
		return
	}

	cgroup := cg.NewV1Service()
	pid := os.Getpid()
	err = cgroup.Put(internal.JobID, pid, internal.Limits.ToCgroupLimits())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	cmd := exec.Command(internal.Command, internal.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Println(err)
	}

	os.Exit(cmd.ProcessState.ExitCode())
}
