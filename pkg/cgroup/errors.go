package cgroup

import "fmt"

type NotSupportedError struct {
	param string
}

func (e *NotSupportedError) Error() string {
	return fmt.Sprintf("%s is not supported", e.param)
}

type InitError struct {
	err error
}

func (e *InitError) Error() string {
	return fmt.Sprintf("cgroup service initialization error. %v", e.err)
}

// The following filesystem errors (create, append, remove)
// might expose server internals to the client, but considering
// that we allow executing any user commands - it is expected behavior

type CreateError struct {
	groupPath string
	subsystem string
	err       error
}

func (e *CreateError) Error() string {
	return fmt.Sprintf(
		"attempt to create group %s for %s failed: %v",
		e.groupPath,
		e.subsystem,
		e.err,
	)
}

type AppendError struct {
	filePath string
	err      error
}

func (e *AppendError) Error() string {
	return fmt.Sprintf(
		"attempt to write to %s file failed: %v",
		e.filePath,
		e.err,
	)
}

type RemoveError struct {
	group     string
	subsystem string
	err       error
}

func (e *RemoveError) Error() string {
	return fmt.Sprintf(
		"attempt to remove group %s in %s failed: %v",
		e.group,
		e.subsystem,
		e.err,
	)
}
