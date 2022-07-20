package approval

import "presentation-service/internal/chat"

type Actor interface {
	NewMessage(message chat.Message)
	Register(chan<- Messages)
	Unregister(chan<- Messages)
	Reset()
}

type onNewMessageReq struct {
	message chat.Message
}

type registerReq struct {
	listener chan<- Messages
}

type unregisterReq struct {
	listener chan<- Messages
}

type resetReq struct{}

type approvedMessagesActor struct {
	state   approvedMessages
	mailbox chan interface{}
}

func (a *approvedMessagesActor) run() {
	for reqUntyped := range a.mailbox {
		switch req := reqUntyped.(type) {
		case *onNewMessageReq:
			a.state.NewMessage(req.message)
		case *registerReq:
			a.state.Register(req.listener)
		case *unregisterReq:
			a.state.Unregister(req.listener)
		case *resetReq:
			a.state.Reset()
		}
	}
}

func (a *approvedMessagesActor) NewMessage(message chat.Message) {
	a.mailbox <- &onNewMessageReq{message: message}
}

func (a *approvedMessagesActor) Register(listener chan<- Messages) {
	a.mailbox <- &registerReq{listener: listener}
}

func (a *approvedMessagesActor) Unregister(listener chan<- Messages) {
	a.mailbox <- &unregisterReq{listener: listener}
}

func (a *approvedMessagesActor) Reset() {
	a.mailbox <- &resetReq{}
}

func NewMessageRouter(name string, chatMessages, rejectedMessages chat.Actor, initialCapacity int) Actor {
	actor := &approvedMessagesActor{
		state:   newMessageRouter(name, chatMessages, rejectedMessages, initialCapacity),
		mailbox: make(chan interface{}, 16),
	}
	go actor.run()

	return actor
}
