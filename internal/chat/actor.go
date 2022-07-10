package chat

type Actor interface {
	NewMessage(Message)
	Register(chan<- Message)
	Unregister(chan<- Message)
}

type newMessageReq struct {
	message Message
}

type registerReq struct {
	listener chan<- Message
}

type unregisterReq struct {
	listener chan<- Message
}

type broadcasterActor struct {
	state   broadcaster
	mailbox chan interface{}
}

func (a *broadcasterActor) run() {
	for reqUntyped := range a.mailbox {
		switch req := reqUntyped.(type) {
		case *newMessageReq:
			a.state.NewMessage(req.message)
		case *registerReq:
			a.state.Register(req.listener)
		case *unregisterReq:
			a.state.Unregister(req.listener)
		}
	}
}

func (a *broadcasterActor) NewMessage(msg Message) {
	a.mailbox <- &newMessageReq{message: msg}
}

func (a *broadcasterActor) Register(listener chan<- Message) {
	a.mailbox <- &registerReq{listener: listener}
}

func (a *broadcasterActor) Unregister(listener chan<- Message) {
	a.mailbox <- &unregisterReq{listener: listener}
}

func NewBroadcasterActor(name string) Actor {
	actor := &broadcasterActor{
		state:   newBroadcaster(name),
		mailbox: make(chan interface{}, 16),
	}
	go actor.run()

	return actor
}
