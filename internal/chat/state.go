package chat

import "log"

type broadcaster struct {
	name      string
	listeners map[chan<- Message]struct{}
}

func (t *broadcaster) NewMessage(message Message) {
	for listener := range t.listeners {
		listener <- message
	}
}

func (t *broadcaster) Register(listener chan<- Message) {
	t.listeners[listener] = struct{}{}
	log.Printf("registered %s message listener (count: %d)", t.name, len(t.listeners))
}

func (t *broadcaster) Unregister(listener chan<- Message) {
	close(listener)
	delete(t.listeners, listener)
	log.Printf("unregistered %s message listener (count: %d)", t.name, len(t.listeners))
}

func newBroadcaster(name string) broadcaster {
	return broadcaster{
		name:      name,
		listeners: map[chan<- Message]struct{}{},
	}
}
