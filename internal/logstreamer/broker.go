package logstreamer

import "sync"

type BrokerSub struct {
	load chan []byte
}

// Broker is message broker consuming messages from one
// source and delivering them to multiple subscribers
type Broker struct {
	mu   sync.Mutex
	subs map[*BrokerSub]struct{}

	listener chan []byte
	quit     chan struct{}
}

func NewBroker() *Broker {
	return &Broker{
		subs: make(map[*BrokerSub]struct{}),

		listener: make(chan []byte, 1),
		quit:     make(chan struct{}),
	}
}

func (b *Broker) Subscribe() *BrokerSub {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := &BrokerSub{
		load: make(chan []byte),
	}
	b.subs[sub] = struct{}{}

	return sub
}

func (b *Broker) Unsubscribe(s *BrokerSub) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subs[s]; !ok {
		// Check this as at this point we might already stop
		// broadcasting and the channel is already closed
		return
	}
	close(s.load)
	delete(b.subs, s)
}

func (b *Broker) Send(msg []byte) {
	b.listener <- msg
}

func (b *Broker) Stop() {
	close(b.quit)
}

func (b *Broker) Broadcast() {
	for {
		select {
		case msg := <-b.listener:
			// Resending the published message to all the subs
			b.mu.Lock()
			for sub := range b.subs {
				sub.load <- msg
			}
			b.mu.Unlock()
		case <-b.quit:
			b.mu.Lock()
			for sub := range b.subs {
				close(sub.load)
				delete(b.subs, sub)
			}
			b.mu.Unlock()
			return
		}
	}
}
