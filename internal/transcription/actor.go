package transcription

type Actor interface {
	NewTranscriptionText(text string)
	Register(chan<- Transcript)
	Unregister(chan<- Transcript)
}

type newTranscriptionTextReq struct {
	text string
}

type registerReq struct {
	listener chan<- Transcript
}

type unregisterReq struct {
	listener chan<- Transcript
}

type transcriptionActor struct {
	state   transcription
	mailbox chan interface{}
}

func (a *transcriptionActor) run() {
	for reqUntyped := range a.mailbox {
		switch req := reqUntyped.(type) {
		case *newTranscriptionTextReq:
			a.state.NewTranscriptionText(req.text)
		case *registerReq:
			a.state.Register(req.listener)
		case *unregisterReq:
			a.state.Unregister(req.listener)
		}
	}
}

func (a *transcriptionActor) NewTranscriptionText(text string) {
	a.mailbox <- &newTranscriptionTextReq{text: text}
}

func (a *transcriptionActor) Register(listener chan<- Transcript) {
	a.mailbox <- &registerReq{listener: listener}
}

func (a *transcriptionActor) Unregister(listener chan<- Transcript) {
	a.mailbox <- &unregisterReq{listener: listener}
}

func NewBroadcasterActor() Actor {
	actor := &transcriptionActor{
		state:   newBroadcaster(),
		mailbox: make(chan interface{}, 16),
	}
	go actor.run()

	return actor
}
