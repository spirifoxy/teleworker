package logstreamer

import (
	"bytes"
	"sync"
)

// SyncBuf is basically a bugger with locks and
// update from broker subscriber
type SyncBuf struct {
	mu  sync.RWMutex
	buf *bytes.Buffer
}

func NewSyncBuf(sub *BrokerSub) *SyncBuf {
	sb := &SyncBuf{
		buf: &bytes.Buffer{},
	}

	go func() {
		for pack := range sub.load {
			sb.Write(pack)
		}
	}()
	return sb
}

func (s *SyncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *SyncBuf) Read(p []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.buf.Read(p)
}

// CopyWithSub copies the buffer with new subscription to keep
// the new buffer updated with data from broker reader
func (s *SyncBuf) CopyWithSub(sub *BrokerSub) *SyncBuf {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := make([]byte, len(s.buf.Bytes()))
	copy(cp, s.buf.Bytes())
	sb := &SyncBuf{
		buf: bytes.NewBuffer(cp),
	}

	go func() {
		for pack := range sub.load {
			sb.Write(pack)
		}
	}()

	return sb
}

// Copy copies the buffer, useful when the streaming is
// already over and we don't need to update the buf over time
func (s *SyncBuf) Copy() *SyncBuf {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := make([]byte, len(s.buf.Bytes()))
	copy(cp, s.buf.Bytes())

	return &SyncBuf{
		buf: bytes.NewBuffer(cp),
	}
}
