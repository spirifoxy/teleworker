package logstreamer

type BrokerSub struct {
	load chan []byte
}

// Broker is message broker consuming messages from one
// source and delivering them to multiple subscribers
type Broker struct {
	listener chan []byte
	sub      chan *BrokerSub
	unsub    chan *BrokerSub
	quit     chan struct{}
}

func NewBroker() *Broker {
	return &Broker{
		listener: make(chan []byte, 1),
		sub:      make(chan *BrokerSub, 1),
		unsub:    make(chan *BrokerSub, 1),
		quit:     make(chan struct{}),
	}
}

func (b *Broker) Subscribe() *BrokerSub {
	s := &BrokerSub{
		load: make(chan []byte),
	}
	b.sub <- s
	return s
}

func (b *Broker) Unsubscribe(s *BrokerSub) {
	b.unsub <- s
}

func (b *Broker) Send(msg []byte) {
	b.listener <- msg
}

func (b *Broker) Stop() {
	close(b.quit)
}

func (b *Broker) Broadcast() {
	subs := map[*BrokerSub]struct{}{}
	for {
		select {
		case msg := <-b.listener:
			// Resending the published message to all the subs
			for sub := range subs {
				sub.load <- msg
			}
		case sub := <-b.sub:
			subs[sub] = struct{}{}
		case sub := <-b.unsub:
			close(sub.load)
			delete(subs, sub)
		case <-b.quit:
			for sub := range subs {
				close(sub.load)
			}
			return
		}
	}
}
