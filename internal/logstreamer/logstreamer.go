package logstreamer

import (
	"context"
	"io"
	"log"
	"sync"
	"time"
)

const defaultBufSize = 1024

type LogStreamer struct {
	reader  io.ReadCloser
	broker  *Broker
	buf     *SyncBuf
	mu      sync.Mutex
	streams []chan []byte
}

func NewLogStreamer(reader io.ReadCloser) *LogStreamer {
	broker := NewBroker()
	sub := broker.Subscribe()
	buf := NewSyncBuf(sub)

	ls := &LogStreamer{
		reader:  reader,
		broker:  broker,
		buf:     buf,
		streams: []chan []byte{},
	}

	go ls.readLogs()
	go broker.Broadcast()

	return ls
}

func (s *LogStreamer) readLogs() {
	for {
		pack := make([]byte, defaultBufSize)
		_, err := s.reader.Read(pack)
		if err == io.EOF {
			// There is nothing left to read now, but the task is still
			// running - so just chill for a second and try again
			time.Sleep(time.Second)
			continue
		} else if err != nil {
			// We end up here in case the task is finished and the reader
			// is closed. It might be that something unexpected happened,
			// but anyway shutting down this goroutine as we aren't going
			// anywhere from here
			break
		}
		s.broker.Send(pack)
	}
}

func (s *LogStreamer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.reader.Close()
	s.broker.Stop()
	for _, stream := range s.streams {
		close(stream)
	}
}

func (s *LogStreamer) Stream(ongoing bool, ctx context.Context) <-chan []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan []byte)
	s.streams = append(s.streams, ch)

	go func() {
		bufCopy, sub := s.dupBuffer(ongoing)

		for {
			select {
			case <-ctx.Done():
				// If broker is not yet closed, i.e. the task is still alive
				if sub != nil {
					s.broker.Unsubscribe(sub)
				}
				return
			default:
			}

			pack := make([]byte, defaultBufSize)
			_, err := bufCopy.Read(pack)

			if err == io.EOF {
				// If the task is not alive anymore and yet somebody
				// wants to get logs - when we reach EOF we might as well
				// close the stream and exit
				if !ongoing {
					close(ch)
					return
				}
				// On the other hand, if the task is still in progress -
				// the same logic as for the main buffer applies, keep
				// reading even when reached EOF as the subscriber might
				// add something to the buffer
				time.Sleep(time.Second)
				continue

			} else if err != nil {
				log.Printf("unexpected error happened while streaming: %v", err)
				break
			}

			ch <- pack
		}
	}()

	return ch
}

func (s *LogStreamer) dupBuffer(withUpdates bool) (*SyncBuf, *BrokerSub) {
	if withUpdates {
		sub := s.broker.Subscribe()
		bufCopy := s.buf.CopyWithSub(sub)
		return bufCopy, sub
	}

	return s.buf.Copy(), nil
}
