package notification

import (
	"sync"
)

type Notification[T any] struct {
	subscribers map[chan<- T]struct{}
	mutex       sync.RWMutex
}

func (n *Notification[T]) NotifyAll(value T) {
	for subscriber := range n.subscribers {
		subscriber <- value
	}

}

func (n *Notification[T]) Subscribe(subscriber chan<- T) int {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.subscribers[subscriber] = struct{}{}

	return len(n.subscribers)
}

func (n *Notification[T]) Unsubscribe(subscriber chan<- T) int {
	close(subscriber)

	n.mutex.Lock()
	defer n.mutex.Unlock()
	delete(n.subscribers, subscriber)

	return len(n.subscribers)
}

func NewNotification[T any]() *Notification[T] {
	return &Notification[T]{
		subscribers: map[chan<- T]struct{}{},
	}
}
