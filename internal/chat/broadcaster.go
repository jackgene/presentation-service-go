package chat

import (
	"log"
	"presentation-service/internal/notification"
)

type Broadcaster struct {
	name         string
	notification *notification.Notification[Message]
}

func (b *Broadcaster) NewMessage(message Message) {
	log.Printf("Received %s message - %s", b.name, message)
	b.notification.NotifyAll(message)
}

func (b *Broadcaster) Subscribe(subscriber chan<- Message) {
	numSubs := b.notification.Subscribe(subscriber)
	log.Printf("+1 %s message notification (=%d)", b.name, numSubs)
}

func (b *Broadcaster) Unsubscribe(subscriber chan<- Message) {
	numSubs := b.notification.Unsubscribe(subscriber)
	log.Printf("-1 %s message notification (=%d)", b.name, numSubs)
}

func NewBroadcaster(name string) *Broadcaster {
	return &Broadcaster{
		name:         name,
		notification: notification.NewNotification[Message](),
	}
}
