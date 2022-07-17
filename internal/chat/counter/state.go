package counter

import (
	"log"
	"presentation-service/internal/chat"
)

type sendersByToken struct {
	extractToken          func(string) string
	tokensBySender        map[string]string
	tokenFrequencies      frequencies
	initialCapacity       int
	messages              chan chat.Message
	chatMessagesActor     chat.Actor
	rejectedMessagesActor chat.Actor
	listeners             map[chan<- Counts]struct{}
}

func (c *sendersByToken) notifyListener(listener chan<- Counts) {
	counts := Counts{
		ItemsByCount: make(map[int][]string),
	}
	// Safe deep copy
	for count, items := range c.tokenFrequencies.itemsByCount {
		itemsCopy := make([]string, len(items))
		copy(itemsCopy, items)
		counts.ItemsByCount[count] = itemsCopy
	}

	listener <- counts
}

func (c *sendersByToken) notifyAllListener() {
	for listener := range c.listeners {
		c.notifyListener(listener)
	}
}

func (c *sendersByToken) NewMessage(message chat.Message) {
	var sender string
	if message.Sender != "Me" {
		sender = message.Sender
	}
	oldToken := c.tokensBySender[sender]
	newToken := c.extractToken(message.Text)

	if newToken != "" {
		log.Printf(`Extracted token "%s"`, newToken)
		if sender != "" {
			c.tokensBySender[sender] = newToken
		}

		c.tokenFrequencies.update(newToken, 1)
		if oldToken != "" {
			c.tokenFrequencies.update(oldToken, -1)
		}

		c.notifyAllListener()
	} else {
		log.Printf("No token extracted")
		c.rejectedMessagesActor.NewMessage(message)
	}
}

func (c *sendersByToken) Register(listener chan<- Counts) {
	c.notifyListener(listener)
	if c.messages == nil {
		c.messages = make(chan chat.Message)
		c.chatMessagesActor.Register(c.messages)
		go func() {
			for msg := range c.messages {
				c.NewMessage(msg)
			}
		}()
	}
	c.listeners[listener] = struct{}{}
}

func (c *sendersByToken) Unregister(listener chan<- Counts) {
	close(listener)
	delete(c.listeners, listener)
	if len(c.listeners) == 0 {
		c.chatMessagesActor.Unregister(c.messages)
		c.messages = nil
	}
}

func (c *sendersByToken) Reset() {
	c.tokensBySender = make(map[string]string, c.initialCapacity)
	c.tokenFrequencies = newFrequencies(c.initialCapacity)
	c.notifyAllListener()
}

func newSendersByToken(extractToken func(string) string, chatMessages, rejectedMessages chat.Actor, initialCapacity int) sendersByToken {
	return sendersByToken{
		extractToken:          extractToken,
		tokensBySender:        make(map[string]string, initialCapacity),
		tokenFrequencies:      newFrequencies(initialCapacity),
		initialCapacity:       initialCapacity,
		chatMessagesActor:     chatMessages,
		rejectedMessagesActor: rejectedMessages,
		listeners:             map[chan<- Counts]struct{}{},
	}
}
