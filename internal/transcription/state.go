package transcription

type transcription struct {
	text      string
	listeners map[chan<- Transcript]struct{}
}

func (t *transcription) NewTranscriptionText(text string) {
	transcript := Transcript{Text: text}
	for listener := range t.listeners {
		listener <- transcript
	}
	t.text = text
}

func (t *transcription) Register(listener chan<- Transcript) {
	listener <- Transcript{Text: t.text}
	t.listeners[listener] = struct{}{}
}

func (t *transcription) Unregister(listener chan<- Transcript) {
	close(listener)
	delete(t.listeners, listener)
}

func newBroadcaster() transcription {
	return transcription{
		text:      "",
		listeners: map[chan<- Transcript]struct{}{},
	}
}
