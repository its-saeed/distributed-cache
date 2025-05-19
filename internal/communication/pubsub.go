package communication

import (
	"log"
	"sync"
	"sync/atomic"
)

// Message represents a pub/sub message
type Message struct {
	Topic     string
	Payload   []byte
	Sender    string
	Timestamp int64
	Metadata  map[string]string
}

// MessageHandler defines the function signature for message handlers
type MessageHandler func(Message)

// PubSub implements the publish-subscribe pattern with better performance
type PubSub struct {
	mu     sync.RWMutex
	subs   map[string][]MessageHandler // Topic -> handlers mapping
	closed atomic.Bool                 // Atomic flag for PubSub closure
	debug  bool                        // Debug logging
}

// NewPubSub creates a new PubSub instance
func NewPubSub(debug bool) *PubSub {
	return &PubSub{
		subs:  make(map[string][]MessageHandler),
		debug: debug,
	}
}

// Subscribe registers a handler for a topic
func (ps *PubSub) Subscribe(topic string, handler MessageHandler) {
	ps.mu.Lock()
	ps.subs[topic] = append(ps.subs[topic], handler)
	ps.mu.Unlock()

	if ps.debug {
		log.Printf("Subscribed to topic: %s", topic)
	}
}

// PublishAsync publishes a message concurrently to subscribers
func (ps *PubSub) PublishAsync(topic string, msg Message) {
	ps.mu.RLock()
	handlers, exists := ps.subs[topic]
	ps.mu.RUnlock()

	if !exists || ps.closed.Load() {
		return
	}

	for _, h := range handlers {
		go h(msg) // Non-blocking execution
	}
}

// PublishSync publishes a message synchronously (blocking)
func (ps *PubSub) PublishSync(topic string, msg Message) {
	ps.mu.RLock()
	handlers, exists := ps.subs[topic]
	ps.mu.RUnlock()

	if !exists || ps.closed.Load() {
		return
	}

	for _, h := range handlers {
		h(msg) // Direct execution
	}
}

// Unsubscribe removes a specific handler from a topic
func (ps *PubSub) Unsubscribe(topic string, handler MessageHandler) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	handlers, exists := ps.subs[topic]
	if !exists {
		return
	}

	for i, h := range handlers {
		if &h == &handler { // Compare function references
			ps.subs[topic] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	if len(ps.subs[topic]) == 0 {
		delete(ps.subs, topic)
	}

	if ps.debug {
		log.Printf("Unsubscribed from topic: %s", topic)
	}
}

// Close shuts down the PubSub system
func (ps *PubSub) Close() {
	if ps.closed.CompareAndSwap(false, true) {
		ps.mu.Lock()
		ps.subs = nil // Free memory
		ps.mu.Unlock()

		if ps.debug {
			log.Println("PubSub system closed.")
		}
	}
}
