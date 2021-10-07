package cgroup

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	permissions = 0555
	MemLimit    = "memory.limit_in_bytes"
	BlkioWeight = "blkio.weight"
	CpuShares   = "cpu.shares"
)

type Subsystem struct {
	name string
	// params made to be slice just for the sake of possible future extension,
	// when we might need more than one parameter from one group
	params []string
}

type V1Service struct {
	mu         sync.Mutex
	subsystems []*Subsystem

	root      string
	rootGroup string
	procsFile string
}

// NewV1Service sets up cgroup service to work with
// hardcoded (on purpose) cpu, memory and blkio parameters
func NewV1Service() *V1Service {
	subsystems := map[string][]string{
		"cpu":    {CpuShares},
		"memory": {MemLimit},
		"blkio":  {BlkioWeight},
	}

	s := &V1Service{
		root:      "/sys/fs/cgroup",
		rootGroup: "teleworker",
		procsFile: "cgroup.procs",
	}
	for name, params := range subsystems {
		sys := &Subsystem{
			name:   name,
			params: params,
		}
		s.subsystems = append(s.subsystems, sys)
	}

	return s
}

// NewV1Runner initialize cgroups service, but also runs
// checks, creates root directory (if not presented)
func NewV1Runner() (*V1Service, error) {
	s := NewV1Service()

	// Check that kernel supports everything we need
	for _, sys := range s.subsystems {
		for _, param := range sys.params {
			p := path.Join(s.root, sys.name, param)
			if _, err := os.Stat(p); os.IsNotExist(err) {
				return nil, &NotSupportedError{param}
			}
		}
	}

	// In case root directory already exists it'll do nothing.
	if err := s.createGroupDir(s.rootGroup); err != nil {
		return nil, &InitError{err}
	}

	// Cleans up possible left overs from the last launch
	go s.cleanup()

	return s, nil
}

func (s *V1Service) createGroupDir(groupPath string) error {
	for _, sys := range s.subsystems {
		p := path.Join(s.root, sys.name, groupPath)
		err := os.MkdirAll(p, permissions)
		if err != nil {
			return &CreateError{
				groupPath: groupPath,
				subsystem: sys.name,
				err:       err,
			}
		}
	}

	return nil
}

func (s *V1Service) Put(id string, pid int, limits Limits) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Path for the group relative to the subsystem dir
	groupRelPath := s.rootGroup
	if len(limits) > 0 {
		groupRelPath = fmt.Sprintf("%s/%s", s.rootGroup, id)
		if err := s.createGroupDir(groupRelPath); err != nil {
			return err
		}
	}

	for _, sys := range s.subsystems {
		for param, val := range limits {
			paramSystem := strings.Split(param, ".")[0]
			// Check that we are writing the parameter
			// to the right subsystem
			if sys.name != paramSystem {
				continue
			}

			paramFile := path.Join(s.root, sys.name, groupRelPath, param)
			if err := s.appendToFile(paramFile, val); err != nil {
				return err
			}
		}

		procsFile := path.Join(s.root, sys.name, groupRelPath, s.procsFile)
		if err := s.appendToFile(procsFile, strconv.Itoa(pid)); err != nil {
			return err
		}
	}

	return nil
}

func (s *V1Service) appendToFile(filePath, value string) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, permissions)
	if err != nil {
		return &AppendError{filePath, err}
	}
	defer f.Close()

	_, err = f.WriteString(value)
	if err != nil {
		return &AppendError{filePath, err}
	}

	return nil
}

// Remove removes the subgroup by provided id
func (s *V1Service) Remove(groupID string) error {
	// As removal might fail because the directories won't be
	// empty immediately after the job is terminated,
	// we make several attempts removing them adding pauses
	// in case of failure
	const attempts = 5

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sys := range s.subsystems {
		p := path.Join(s.root, sys.name, s.rootGroup, groupID)
		// If, for example, removal for some reason won't work correctly and throw
		// error in the middle of processing, it can be simply launched again later
		// since any attempt to remove non-existent directory will be ignored

		for i := 0; ; i++ {
			err := os.RemoveAll(p)
			if err == nil {
				break
			}

			if i >= (attempts - 1) {
				return &RemoveError{
					group:     groupID,
					subsystem: sys.name,
					err:       err,
				}
			}

			time.Sleep(time.Second)
		}
	}

	return nil
}

// cleanup checks the root group directory and cleanups if anything found in it.
// Though the job group is removed when it is finished or stopped,
// this will be usefull in case of unexpected server shutdown,
// as everything is stored in memory and there is no way to recover
func (s *V1Service) cleanup() {
	// TODO not yet implemented.
	// To be added in case of free time left
}
