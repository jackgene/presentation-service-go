package chat

import "log"

type broadcaster struct {
	name      string
	listeners map[chan<- Message]struct{}
}

func (b *broadcaster) NewMessage(message Message) {
	log.Printf("Received %s message - %s", b.name, message)
	for listener := range b.listeners {
		listener <- message
	}
}

func (b *broadcaster) Register(listener chan<- Message) {
	b.listeners[listener] = struct{}{}
	log.Printf("+1 %s message listener (=%d)", b.name, len(b.listeners))
}

func (b *broadcaster) Unregister(listener chan<- Message) {
	close(listener)
	delete(b.listeners, listener)
	log.Printf("-1 %s message listener (=%d)", b.name, len(b.listeners))
}

func newBroadcaster(name string) broadcaster {
	return broadcaster{
		name:      name,
		listeners: map[chan<- Message]struct{}{},
	}
}
