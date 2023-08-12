package counter

import (
	"log"
	"presentation-service/internal/chat"
	"time"
)

const batchPeriodMillis = 100

type sendersByToken struct {
	name                  string
	self                  Actor
	extractToken          func(string) string
	tokensBySender        map[string]string
	tokenFrequencies      frequencies
	initialCapacity       int
	messages              chan chat.Message
	chatMessagesActor     chat.Actor
	rejectedMessagesActor chat.Actor
	listeners             map[chan<- Counts]struct{}
	awaitingNotify        bool
}

func (c *sendersByToken) copyCounts() Counts {
	counts := Counts{
		TokensAndCounts: make(
			[][]interface{}, 0, len(c.tokenFrequencies.itemsByCount),
		),
	}
	// safe deep copy
	for count, items := range c.tokenFrequencies.itemsByCount {
		itemsCopy := make([]string, len(items))
		copy(itemsCopy, items)
		counts.TokensAndCounts = append(
			counts.TokensAndCounts, []interface{}{count, itemsCopy},
		)
	}

	return counts
}

func (c *sendersByToken) notifyAllListeners() {
	counts := c.copyCounts()
	for listener := range c.listeners {
		listener <- counts
	}
}

func (c *sendersByToken) scheduleNotification() {
	if !c.awaitingNotify {
		time.AfterFunc(batchPeriodMillis*time.Millisecond, func() {
			c.self.notifyAllListeners()
			c.awaitingNotify = false
		})
		c.awaitingNotify = true
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

		c.scheduleNotification()
	} else {
		log.Printf("No token extracted")
		c.rejectedMessagesActor.NewMessage(message)
	}
}

func (c *sendersByToken) Register(listener chan<- Counts) {
	listener <- c.copyCounts()
	if c.messages == nil {
		c.messages = make(chan chat.Message)
		c.chatMessagesActor.Register(c.messages)
		go func() {
			for msg := range c.messages {
				c.self.NewMessage(msg)
			}
		}()
	}
	c.listeners[listener] = struct{}{}
	log.Printf("+1 %s listener (=%d)", c.name, len(c.listeners))
}

func (c *sendersByToken) Unregister(listener chan<- Counts) {
	close(listener)
	delete(c.listeners, listener)
	if len(c.listeners) == 0 {
		c.chatMessagesActor.Unregister(c.messages)
		c.messages = nil
	}
	log.Printf("-1 %s listener (=%d)", c.name, len(c.listeners))
}

func (c *sendersByToken) Reset() {
	c.tokensBySender = make(map[string]string, c.initialCapacity)
	c.tokenFrequencies = newFrequencies(c.initialCapacity)
	c.notifyAllListeners()
}

func newSendersByToken(
	name string, extractToken func(string) string, chatMessages, rejectedMessages chat.Actor, initialCapacity int,
) sendersByToken {
	return sendersByToken{
		name:                  name,
		extractToken:          extractToken,
		tokensBySender:        make(map[string]string, initialCapacity),
		tokenFrequencies:      newFrequencies(initialCapacity),
		initialCapacity:       initialCapacity,
		chatMessagesActor:     chatMessages,
		rejectedMessagesActor: rejectedMessages,
		listeners:             map[chan<- Counts]struct{}{},
		awaitingNotify:        false,
	}
}
