package counter

import (
	"presentation-service/internal/chat"
)

type Actor interface {
	NewMessage(message chat.Message)
	Register(chan<- Counts)
	Unregister(chan<- Counts)
	Reset()
}

type onNewMessageReq struct {
	message chat.Message
}

type registerReq struct {
	listener chan<- Counts
}

type unregisterReq struct {
	listener chan<- Counts
}

type resetReq struct{}

type sendersByTokenActor struct {
	state   sendersByToken
	mailbox chan interface{}
}

func (a *sendersByTokenActor) run() {
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

func (a *sendersByTokenActor) NewMessage(message chat.Message) {
	a.mailbox <- &onNewMessageReq{message: message}
}

func (a *sendersByTokenActor) Register(listener chan<- Counts) {
	a.mailbox <- &registerReq{listener: listener}
}

func (a *sendersByTokenActor) Unregister(listener chan<- Counts) {
	a.mailbox <- &unregisterReq{listener: listener}
}

func (a *sendersByTokenActor) Reset() {
	a.mailbox <- &resetReq{}
}

func NewSendersByTokenActor(
	name string, extractToken func(string) string, chatMessages, rejectedMessages chat.Actor, initialCapacity int,
) Actor {
	actor := &sendersByTokenActor{
		state:   newSendersByToken(name, extractToken, chatMessages, rejectedMessages, initialCapacity),
		mailbox: make(chan interface{}, 16),
	}
	go actor.run()

	return actor
}
