package storage

import "github.com/spirifoxy/teleworker/pkg/teleworker"

// Storage is an inner storage base interface
type Storage interface {
	Get(id string) (*teleworker.Job, error)
	Put(*teleworker.Job) error
}
