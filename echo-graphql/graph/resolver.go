//go:generate go run github.com/99designs/gqlgen generate

package graph

import (
	"strings"
	"sync"

	"github.com/jsr-probitas/echo-servers/echo-graphql/graph/model"
)

// filteredSubscriber represents a subscriber with optional text filter
type filteredSubscriber struct {
	ch         chan *model.Message
	textFilter *string // nil means no filter
}

// Resolver is the root resolver for all GraphQL operations
type Resolver struct {
	mu                  sync.RWMutex
	messages            map[string]*model.Message
	nextID              int
	messageChannels     []chan *model.Message
	filteredSubscribers []filteredSubscriber
}

// NewResolver creates a new resolver instance
func NewResolver() *Resolver {
	return &Resolver{
		messages: make(map[string]*model.Message),
		nextID:   1,
	}
}

// Subscribe adds a channel to receive message events
func (r *Resolver) Subscribe() chan *model.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan *model.Message, 1)
	r.messageChannels = append(r.messageChannels, ch)
	return ch
}

// SubscribeFiltered adds a channel to receive filtered message events
func (r *Resolver) SubscribeFiltered(textFilter *string) chan *model.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan *model.Message, 1)
	r.filteredSubscribers = append(r.filteredSubscribers, filteredSubscriber{
		ch:         ch,
		textFilter: textFilter,
	})
	return ch
}

// Unsubscribe removes a channel from message events
func (r *Resolver) Unsubscribe(ch chan *model.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, c := range r.messageChannels {
		if c == ch {
			r.messageChannels = append(r.messageChannels[:i], r.messageChannels[i+1:]...)
			close(ch)
			return
		}
	}
}

// UnsubscribeFiltered removes a filtered channel from message events
func (r *Resolver) UnsubscribeFiltered(ch chan *model.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, sub := range r.filteredSubscribers {
		if sub.ch == ch {
			r.filteredSubscribers = append(r.filteredSubscribers[:i], r.filteredSubscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// Broadcast sends a message to all subscribers
func (r *Resolver) Broadcast(msg *model.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Broadcast to unfiltered subscribers
	for _, ch := range r.messageChannels {
		select {
		case ch <- msg:
		default:
		}
	}
	// Broadcast to filtered subscribers (only if filter matches)
	for _, sub := range r.filteredSubscribers {
		if sub.textFilter == nil || strings.Contains(msg.Text, *sub.textFilter) {
			select {
			case sub.ch <- msg:
			default:
			}
		}
	}
}
