package storage

import (
	"sync"
	"time"

	tw "github.com/spirifoxy/teleworker/pkg/teleworker"
)

type Memory struct {
	mu   sync.RWMutex
	data map[string]*tw.Job
	// If TTL is specified, then the dead jobs will yet be
	// presented in the storage for at least that time.
	ttl time.Duration
}

// Option is function used for applying configurations to storage
type Option func(*Memory)

func NewMemStorage(options ...Option) *Memory {
	s := &Memory{
		data: make(map[string]*tw.Job),
	}

	for _, opt := range options {
		opt(s)
	}

	if s.ttl > 0 {
		// Run cleanup routine every minute in case the ttl was set
		// while creating the storage. Interval is hardcoded for simplicity.
		const cleanupInterval = 1 * time.Minute

		// If ttl is set the worst case scenario time of the job being present
		// in the storage is going to be:
		//     ttl + interval time + N * time to calculate expiration and remove the job,
		// where N is number of jobs in the storage. This behavior is expected as
		// we don't care that much if the job lives in memory some time more than
		// the exact TTL and aim to the simplest implementation possible.
		go func() {
			for range time.Tick(cleanupInterval) {
				s.cleanup()
			}
		}()
	}

	return s
}

// WithTTL is used for specifying the terminated jobs ttl.
func WithTTL(ttl time.Duration) Option {
	return func(s *Memory) {
		s.ttl = ttl
	}
}

func (s *Memory) Get(id string) (*tw.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, alreadyIn := s.data[id]
	if !alreadyIn {
		return nil, &NotFoundError{id}
	}

	return job, nil
}

func (s *Memory) Put(job *tw.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := job.ID.String()
	if _, alreadyIn := s.data[id]; alreadyIn {
		return &AlreadyInError{id}
	}

	s.data[id] = job

	return nil
}

// cleanup frees server memory by calculating the job
// remaining time to leave based on ExitedAt set upon job
// termination and server default ttl value
func (s *Memory) cleanup() {
	// TODO not yet implemented.
}
