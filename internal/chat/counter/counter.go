package counter

import (
	"log"
	"presentation-service/internal/chat"
	"presentation-service/internal/notification"
	"sync"
	"time"
)

const batchPeriodMillis = 100

type SendersByTokenCounter struct {
	name                       string
	extractToken               func(string) string
	tokensBySender             map[string]string
	tokenFrequencies           frequencies
	mutex                      sync.RWMutex
	initialCapacity            int
	messages                   chan chat.Message
	chatMessageBroadcaster     *chat.Broadcaster
	rejectedMessageBroadcaster *chat.Broadcaster
	notification               *notification.Notification[Counts]
	awaitingNotify             bool
	awaitingNotifyMutex        sync.Mutex
}

func (c *SendersByTokenCounter) copyCounts() Counts {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	counts := Counts{
		TokensAndCounts: make(
			[][]any, 0, len(c.tokenFrequencies.itemsByCount),
		),
	}
	// safe deep copy
	for count, items := range c.tokenFrequencies.itemsByCount {
		itemsCopy := make([]string, len(items))
		copy(itemsCopy, items)
		counts.TokensAndCounts = append(
			counts.TokensAndCounts, []any{count, itemsCopy},
		)
	}

	return counts
}

func (c *SendersByTokenCounter) notifyAllSubscribers() {
	c.notification.NotifyAll(c.copyCounts())
}

func (c *SendersByTokenCounter) scheduleNotification() {
	c.awaitingNotifyMutex.Lock()
	defer c.awaitingNotifyMutex.Unlock()
	if !c.awaitingNotify {
		time.AfterFunc(batchPeriodMillis*time.Millisecond, func() {
			c.notifyAllSubscribers()
			c.awaitingNotifyMutex.Lock()
			defer c.awaitingNotifyMutex.Unlock()
			c.awaitingNotify = false
		})
		c.awaitingNotify = true
	}
}

func (c *SendersByTokenCounter) NewMessage(message chat.Message) {
	var sender string
	if message.Sender != "Me" {
		sender = message.Sender
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
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
		c.rejectedMessageBroadcaster.NewMessage(message)
	}
}

func (c *SendersByTokenCounter) Subscribe(subscriber chan<- Counts) {
	go func() {
		subscriber <- c.copyCounts()
	}()
	if c.messages == nil {
		c.messages = make(chan chat.Message)
		c.chatMessageBroadcaster.Subscribe(c.messages)
		go func() {
			for msg := range c.messages {
				c.NewMessage(msg)
			}
		}()
	}
	numSubs := c.notification.Subscribe(subscriber)
	log.Printf("+1 %s notification (=%d)", c.name, numSubs)
}

func (c *SendersByTokenCounter) Unsubscribe(subscriber chan<- Counts) {
	numSubs := c.notification.Unsubscribe(subscriber)
	if numSubs == 0 {
		c.chatMessageBroadcaster.Unsubscribe(c.messages)
		c.messages = nil
	}
	log.Printf("-1 %s notification (=%d)", c.name, numSubs)
}

func (c *SendersByTokenCounter) Reset() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tokensBySender = make(map[string]string, c.initialCapacity)
	c.tokenFrequencies = newFrequencies(c.initialCapacity)

	c.scheduleNotification()
}

func NewSendersByTokenActor(
	name string, extractToken func(string) string,
	chatMessageBroadcaster, rejectedMessageBroadcaster *chat.Broadcaster,
	initialCapacity int,
) *SendersByTokenCounter {
	return &SendersByTokenCounter{
		name:                       name,
		extractToken:               extractToken,
		tokensBySender:             make(map[string]string, initialCapacity),
		tokenFrequencies:           newFrequencies(initialCapacity),
		initialCapacity:            initialCapacity,
		chatMessageBroadcaster:     chatMessageBroadcaster,
		rejectedMessageBroadcaster: rejectedMessageBroadcaster,
		notification:               notification.NewNotification[Counts](),
		awaitingNotify:             false,
	}
}
