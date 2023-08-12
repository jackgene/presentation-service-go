package moderation

import (
	"log"
	"presentation-service/internal/chat"
	"presentation-service/internal/notification"
	"sync"
)

type TextCollector struct {
	name                       string
	chatText                   []string
	mutex                      sync.RWMutex
	initialCapacity            int
	messages                   chan chat.Message
	chatMessageBroadcaster     *chat.Broadcaster
	rejectedMessageBroadcaster *chat.Broadcaster
	notification               *notification.Notification[Messages]
}

func (t *TextCollector) copyMessages() Messages {
	messages := Messages{
		ChatText: make([]string, 0, len(t.chatText)),
	}
	// Safe copy in reverse
	for i := len(t.chatText) - 1; i >= 0; i-- {
		messages.ChatText = append(messages.ChatText, t.chatText[i])
	}

	return messages
}

func (t *TextCollector) notifyAllSubscribers() {
	t.notification.NotifyAll(t.copyMessages())
}

func (t *TextCollector) NewMessage(message chat.Message) {
	if message.Sender != "" {
		t.rejectedMessageBroadcaster.NewMessage(message)
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.chatText = append(t.chatText, message.Text)
	t.notifyAllSubscribers()
}

func (t *TextCollector) Subscribe(subscriber chan<- Messages) {
	go func() {
		t.mutex.RLock()
		defer t.mutex.RUnlock()
		subscriber <- t.copyMessages()
	}()
	if t.messages == nil {
		t.messages = make(chan chat.Message)
		t.chatMessageBroadcaster.Subscribe(t.messages)
		go func() {
			for msg := range t.messages {
				t.NewMessage(msg)
			}
		}()
	}
	numSubs := t.notification.Subscribe(subscriber)
	log.Printf("+1 %s subscriber (=%d)", t.name, numSubs)
}

func (t *TextCollector) Unsubscribe(subscriber chan<- Messages) {
	numSubs := t.notification.Unsubscribe(subscriber)
	if numSubs == 0 {
		t.chatMessageBroadcaster.Unsubscribe(t.messages)
		t.messages = nil
	}
	log.Printf("-1 %s subscriber (=%d)", t.name, numSubs)
}

func (t *TextCollector) Reset() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.chatText = make([]string, 0, t.initialCapacity)
	t.notifyAllSubscribers()
}

func NewMessageRouter(
	name string, chatMessageBroadcaster, rejectedMessageBroadcaster *chat.Broadcaster, initialCapacity int,
) *TextCollector {
	return &TextCollector{
		name:                       name,
		chatText:                   make([]string, 0, initialCapacity),
		initialCapacity:            initialCapacity,
		chatMessageBroadcaster:     chatMessageBroadcaster,
		rejectedMessageBroadcaster: rejectedMessageBroadcaster,
		notification:               notification.NewNotification[Messages](),
	}
}
