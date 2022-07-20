package transcription

import "log"

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
	log.Printf("+1 transcription listener (=%d)", len(t.listeners))
}

func (t *transcription) Unregister(listener chan<- Transcript) {
	close(listener)
	delete(t.listeners, listener)
	log.Printf("-1 transcription listener (=%d)", len(t.listeners))
}

func newBroadcaster() transcription {
	return transcription{
		text:      "",
		listeners: map[chan<- Transcript]struct{}{},
	}
}
