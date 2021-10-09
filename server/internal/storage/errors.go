package storage

import "fmt"

type NotFoundError struct {
	id string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("job %s was not found in the storage", e.id)
}

type AlreadyInError struct {
	id string
}

func (e *AlreadyInError) Error() string {
	return fmt.Sprintf("job %s is already in the storage", e.id)
}
