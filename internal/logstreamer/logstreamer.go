package logstreamer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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
		n, err := s.reader.Read(pack)
		if n > 0 {
			s.broker.Send(pack[:n])
		}

		if errors.Is(err, io.EOF) {
			// There is nothing left to read now, but the task is still
			// running - so just chill for a second and try again
			time.Sleep(time.Second)
			continue
		} else if errors.Is(err, os.ErrClosed) {
			// This is the expected behavior - ending up here means that
			// that task was either finished or terminated and the reader
			// was closed, so we just break the routine
			break
		} else if err != nil {
			// Something unexpected happened. Write the error to the buffer
			// so if the client will run the stream - he will see the problem
			// and might restart the job.
			// Should hardly ever happen
			readError := fmt.Sprintf("unexpected error while reading the task output: %v\n", err)
			log.Println(readError)
			s.broker.Send([]byte(readError))
			break
		}
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
				if sub != nil {
					// If broker is not yet closed, i.e. the task is still alive
					s.broker.Unsubscribe(sub)
				}
				return
			default:
			}

			pack := make([]byte, defaultBufSize)
			n, err := bufCopy.Read(pack)
			if n > 0 {
				ch <- pack[:n]
			}

			if errors.Is(err, io.EOF) {
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
				// Should hardly ever happen
				streamError := fmt.Sprintf("unexpected error happened while streaming: %v\n", err)
				log.Println(streamError)
				s.broker.Send([]byte(streamError))
				break
			}
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
