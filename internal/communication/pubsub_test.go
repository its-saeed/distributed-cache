package communication

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Test PubSub subscription and message publishing (async)
func TestPubSubSubscribeAndPublishAsync(t *testing.T) {
	pubsub := NewPubSub(true)
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(msg Message) {
		if string(msg.Payload) != "test-message" {
			t.Errorf("Expected 'test-message', got '%s'", string(msg.Payload))
		}
		wg.Done()
	}

	// Subscribe to a topic
	pubsub.Subscribe("test_topic", handler)

	time.Sleep(50 * time.Millisecond) // Allow subscription setup

	// Publish an async message
	pubsub.PublishAsync("test_topic", Message{
		Topic:     "test_topic",
		Payload:   []byte("test-message"),
		Sender:    "unit-test",
		Timestamp: time.Now().Unix(),
	})

	wg.Wait()
}

// Test synchronous publishing
func TestPubSubPublishSync(t *testing.T) {
	pubsub := NewPubSub(false)
	var received bool

	handler := func(msg Message) {
		received = true
	}

	// Subscribe
	pubsub.Subscribe("sync_topic", handler)

	// Publish synchronously
	pubsub.PublishSync("sync_topic", Message{
		Topic:   "sync_topic",
		Payload: []byte("data"),
	})

	if !received {
		t.Errorf("Synchronous publish failed; handler did not receive message")
	}
}

// Test unsubscribing a handler
func TestPubSubUnsubscribe(t *testing.T) {
	pubsub := NewPubSub(false)
	handlerCalled := false

	handler := func(msg Message) {
		handlerCalled = true
	}

	// Subscribe then unsubscribe
	pubsub.Subscribe("test_topic", handler)
	pubsub.Unsubscribe("test_topic", handler)

	time.Sleep(50 * time.Millisecond) // Ensure unsubscribe is processed

	// Publish a message (should not trigger handler)
	pubsub.PublishAsync("test_topic", Message{Payload: []byte("ignored")})

	time.Sleep(100 * time.Millisecond) // Allow time for async processing

	if handlerCalled {
		t.Errorf("Handler should not have been called after unsubscribe")
	}
}

// Test closing the PubSub system
func TestPubSubClose(t *testing.T) {
	pubsub := NewPubSub(false)

	pubsub.Close()
	pubsub.PublishSync("closed_topic", Message{Payload: []byte("data")})

	// If PublishSync should return an error, update the method signature and test accordingly.
	// Otherwise, remove error checking as PublishSync does not return a value.
}

// Test multiple subscribers receiving messages
func TestPubSubMultipleSubscribers(t *testing.T) {
	pubsub := NewPubSub(false)
	var wg sync.WaitGroup
	wg.Add(2)

	handler1 := func(msg Message) { wg.Done() }
	handler2 := func(msg Message) { wg.Done() }

	pubsub.Subscribe("multi_topic", handler1)
	pubsub.Subscribe("multi_topic", handler2)

	pubsub.PublishAsync("multi_topic", Message{Payload: []byte("data")})

	wg.Wait()
}

// Test PubSub stress with high concurrency
func TestPubSubHighConcurrency(t *testing.T) {
	pubsub := NewPubSub(false)
	var wg sync.WaitGroup
	numMessages := 1000

	handler := func(msg Message) {
		wg.Done()
	}

	pubsub.Subscribe("stress_topic", handler)

	// Publish a large number of messages concurrently
	wg.Add(numMessages)
	for i := 0; i < numMessages; i++ {
		go pubsub.PublishAsync("stress_topic", Message{Payload: []byte(fmt.Sprintf("msg-%d", i))})
	}

	wg.Wait()
}
