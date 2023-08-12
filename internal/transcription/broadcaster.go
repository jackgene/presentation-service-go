package transcription

import (
	"log"
	"presentation-service/internal/notification"
	"sync"
)

type Broadcaster struct {
	text         string
	mutex        sync.RWMutex
	notification *notification.Notification[Transcript]
}

func (b *Broadcaster) NewTranscriptionText(text string) {
	log.Printf("Got transcription text: %v", text)
	transcript := Transcript{Text: text}
	b.notification.NotifyAll(transcript)

	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.text = text
}

func (b *Broadcaster) Subscribe(subscriber chan<- Transcript) {
	go func() {
		b.mutex.RLock()
		defer b.mutex.RUnlock()
		subscriber <- Transcript{Text: b.text}
	}()

	numSubs := b.notification.Subscribe(subscriber)
	log.Printf("+1 transcription subscriber (=%d)", numSubs)
}

func (b *Broadcaster) Unsubscribe(subscriber chan<- Transcript) {
	numSubs := b.notification.Unsubscribe(subscriber)
	log.Printf("-1 transcription subscriber (=%d)", numSubs)
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		text:         "",
		notification: notification.NewNotification[Transcript](),
	}
}
