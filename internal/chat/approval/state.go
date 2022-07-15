package approval

import (
	"log"
	"presentation-service/internal/chat"
)

type approvedMessages struct {
	chatText              []string
	messages              chan chat.Message
	chatMessagesActor     chat.Actor
	rejectedMessagesActor chat.Actor
	listeners             map[chan<- Messages]struct{}
}

func (m *approvedMessages) notifyListener(listener chan<- Messages) {
	msgs := Messages{
		ChatText: make([]string, 0, len(m.chatText)),
	}
	for i := len(m.chatText) - 1; i >= 0; i-- {
		msgs.ChatText = append(msgs.ChatText, m.chatText[i])
	}

	listener <- msgs
}

func (m *approvedMessages) notifyAllListener() {
	for listener := range m.listeners {
		m.notifyListener(listener)
	}
}

func (m *approvedMessages) NewMessage(message chat.Message) {
	if message.Sender != "Me" {
		m.rejectedMessagesActor.NewMessage(message)
		return
	}

	m.chatText = append(m.chatText, message.Text)
	m.notifyAllListener()
}

func (m *approvedMessages) Register(listener chan<- Messages) {
	m.notifyListener(listener)
	if m.messages == nil {
		m.messages = make(chan chat.Message)
		m.chatMessagesActor.Register(m.messages)
		go func() {
			for msg := range m.messages {
				m.NewMessage(msg)
			}
		}()
	}
	m.listeners[listener] = struct{}{}
	log.Printf("registered approved messages listener (count: %v)", len(m.listeners))
}

func (m *approvedMessages) Unregister(listener chan<- Messages) {
	close(listener)
	delete(m.listeners, listener)
	if len(m.listeners) == 0 {
		m.chatMessagesActor.Unregister(m.messages)
		m.messages = nil
	}
	log.Printf("unregistered approved messages listener (count: %v)", len(m.listeners))
}

func (m *approvedMessages) Reset() {
	m.chatText = nil
	m.notifyAllListener()
}

func newMessageRouter(chatMessages, rejectedMessages chat.Actor, initialCapacity int) approvedMessages {
	return approvedMessages{
		chatText:              make([]string, 0, initialCapacity),
		chatMessagesActor:     chatMessages,
		rejectedMessagesActor: rejectedMessages,
		listeners:             map[chan<- Messages]struct{}{},
	}
}
