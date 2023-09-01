package counter

import (
	lru "github.com/hashicorp/golang-lru/v2"
	"log"
	"presentation-service/internal/chat"
	"presentation-service/internal/notification"
	"strings"
	"sync"
	"time"
)

const batchPeriodMillis = 100

type SendersByTokenCounter struct {
	name                       string
	extractTokens              func(string) []string
	tokensPerSender            int
	tokensBySender             map[string]*lru.Cache[string, struct{}]
	tokens                     multiSet[string]
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
			[][]any, 0, len(c.tokens.elementsByCount),
		),
	}
	// safe deep copy
	for count, items := range c.tokens.elementsByCount {
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
	extractedTokens := c.extractTokens(message.Text)
	extractedTokensLen := len(extractedTokens)

	if extractedTokensLen > 0 {
		log.Printf(`Extracted token "%s"`, strings.Join(extractedTokens, `", "`))
		newTokens := make([]string, 0, extractedTokensLen)
		newTokenSet := map[string]struct{}{}
		oldTokens := make([]string, 0, extractedTokensLen)
		oldTokenSet := map[string]struct{}{}
		// Iterate in reverse, prioritizing first tokens
		for i := extractedTokensLen - 1; i >= 0; i-- {
			extractedToken := extractedTokens[i]
			if sender == "" {
				newTokens = append(newTokens, extractedToken)
			} else {
				if _, present := c.tokensBySender[sender]; !present {
					tokens, newLRUError := lru.New[string, struct{}](c.tokensPerSender)
					if newLRUError != nil {
						log.Printf("Error creating LRU cache")
						continue
					}
					c.tokensBySender[sender] = tokens
				}
				tokens := c.tokensBySender[sender]
				oldestToken, _, gotOldest := tokens.GetOldest()
				exists := tokens.Contains(extractedToken)
				evicted := tokens.Add(extractedToken, struct{}{})

				if !exists {
					newTokens = append(newTokens, extractedToken)
					newTokenSet[extractedToken] = struct{}{}
					if gotOldest && evicted {
						oldTokens = append(oldTokens, oldestToken)
						oldTokenSet[oldestToken] = struct{}{}
					}
				}
			}
		}
		for i := len(oldTokens) - 1; i >= 0; i-- {
			oldToken := oldTokens[i]
			if _, alsoNew := newTokenSet[oldToken]; !alsoNew {
				c.tokens.update(oldToken, -1)
			}
		}
		for i := len(newTokens) - 1; i >= 0; i-- {
			newToken := newTokens[i]
			if _, alsoOld := oldTokenSet[newToken]; !alsoOld {
				c.tokens.update(newToken, 1)
			}
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
	c.tokensBySender = make(map[string]*lru.Cache[string, struct{}], c.initialCapacity)
	c.tokens = newMultiSet[string](c.initialCapacity)

	c.scheduleNotification()
}

func NewSendersByTokenActor(
	name string, tokensPerSender int, extractTokens func(string) []string,
	chatMessageBroadcaster, rejectedMessageBroadcaster *chat.Broadcaster,
	initialCapacity int,
) *SendersByTokenCounter {
	return &SendersByTokenCounter{
		name:                       name,
		extractTokens:              extractTokens,
		tokensPerSender:            tokensPerSender,
		tokensBySender:             make(map[string]*lru.Cache[string, struct{}], initialCapacity),
		tokens:                     newMultiSet[string](initialCapacity),
		initialCapacity:            initialCapacity,
		chatMessageBroadcaster:     chatMessageBroadcaster,
		rejectedMessageBroadcaster: rejectedMessageBroadcaster,
		notification:               notification.NewNotification[Counts](),
		awaitingNotify:             false,
	}
}
