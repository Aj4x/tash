package msgbus_test

import (
	"errors"
	"fmt"
	"github.com/Aj4x/tash/internal/msgbus"
	"github.com/Aj4x/tash/internal/uuid"
	"sync"
	"testing"
	"time"
)

func TestNewMessageBus(t *testing.T) {
	bus := msgbus.NewMessageBus()
	if bus == nil {
		t.Error("Expected non-nil MessageBus, got nil")
	}
}

func TestSubscribe(t *testing.T) {
	t.Run("Valid subscription", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")
		handler := make(msgbus.MessageHandler, 10)

		key, err := bus.Subscribe(topic, handler)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if key == (uuid.UUID{}) {
			t.Error("Expected non-empty UUID, got empty UUID")
		}

		// Clean up
		close(handler)
	})

	t.Run("Nil handler", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")

		_, err := bus.Subscribe(topic, nil)

		if !errors.Is(err, msgbus.ErrNilSubChannel) {
			t.Errorf("Expected ErrNilSubChannel, got %v", err)
		}
	})

	t.Run("Multiple subscriptions to same topic", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")
		handler1 := make(msgbus.MessageHandler, 10)
		handler2 := make(msgbus.MessageHandler, 10)

		key1, err1 := bus.Subscribe(topic, handler1)
		key2, err2 := bus.Subscribe(topic, handler2)

		if err1 != nil || err2 != nil {
			t.Errorf("Expected no errors, got %v and %v", err1, err2)
		}

		if key1 == key2 {
			t.Error("Expected unique keys for different subscriptions")
		}

		// Clean up
		close(handler1)
		close(handler2)
	})
}

func TestPublish(t *testing.T) {
	t.Run("Basic publish and receive", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")
		message := []byte("test message")
		handler := make(msgbus.MessageHandler, 10)

		_, err := bus.Subscribe(topic, handler)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		bus.Publish(msgbus.TopicMessage{
			Topic:   topic,
			Message: message,
		})

		// Wait for a message to be processed
		receivedMsg := <-handler

		if receivedMsg.Topic != topic {
			t.Errorf("Expected topic %s, got %s", topic, receivedMsg.Topic)
		}

		if string(receivedMsg.Message) != string(message) {
			t.Errorf("Expected message %s, got %s", message, receivedMsg.Message)
		}

		// Clean up
		close(handler)
	})

	t.Run("Publish to multiple subscribers", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")
		message := []byte("test message")
		handler1 := make(msgbus.MessageHandler, 10)
		handler2 := make(msgbus.MessageHandler, 10)

		_, err1 := bus.Subscribe(topic, handler1)
		_, err2 := bus.Subscribe(topic, handler2)

		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to subscribe: %v, %v", err1, err2)
		}

		bus.Publish(msgbus.TopicMessage{
			Topic:   topic,
			Message: message,
		})

		// Wait for messages to be processed
		receivedMsg1 := <-handler1
		receivedMsg2 := <-handler2

		if string(receivedMsg1.Message) != string(message) {
			t.Errorf("Handler 1: Expected message %s, got %s", message, receivedMsg1.Message)
		}

		if string(receivedMsg2.Message) != string(message) {
			t.Errorf("Handler 2: Expected message %s, got %s", message, receivedMsg2.Message)
		}

		// Clean up
		close(handler1)
		close(handler2)
	})

	t.Run("Publish to nonexistent topic", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("nonexistent-topic")
		message := []byte("test message")

		// This should not panic
		bus.Publish(msgbus.TopicMessage{
			Topic:   topic,
			Message: message,
		})
	})
}

func TestUnsubscribe(t *testing.T) {
	t.Run("Basic unsubscribe", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")
		handler := make(msgbus.MessageHandler, 10)

		key, err := bus.Subscribe(topic, handler)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		bus.Unsubscribe(topic, key)

		// Publish after unsubscribing should not deliver messages
		bus.Publish(msgbus.TopicMessage{
			Topic:   topic,
			Message: []byte("test message"),
		})

		// We need to verify no message was delivered
		// We can use a timeout to ensure no message arrives
		select {
		case msg := <-handler:
			t.Errorf("Received unexpected message after unsubscribe: %v", msg)
		case <-time.After(100 * time.Millisecond):
			// This is the expected path - no message received
		}
	})

	t.Run("Unsubscribe one of multiple subscribers", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")
		handler1 := make(msgbus.MessageHandler, 10)
		handler2 := make(msgbus.MessageHandler, 10)

		key1, _ := bus.Subscribe(topic, handler1)
		_, _ = bus.Subscribe(topic, handler2)

		bus.Unsubscribe(topic, key1)

		message := []byte("test message")
		bus.Publish(msgbus.TopicMessage{
			Topic:   topic,
			Message: message,
		})

		// Check handler1 (unsubscribed) doesn't receive message
		select {
		case msg := <-handler1:
			t.Errorf("Received unexpected message after unsubscribe: %v", msg)
		case <-time.After(100 * time.Millisecond):
			// Expected - no message received
		}

		// Check handler2 still receives a message
		select {
		case msg := <-handler2:
			if string(msg.Message) != string(message) {
				t.Errorf("Expected message %s, got %s", message, msg.Message)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Handler 2 didn't receive message")
		}

		// Clean up
		close(handler2)
	})

	t.Run("Unsubscribe nonexistent subscription", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")
		nonexistentKey := uuid.UUID{} // Empty UUID

		// This should not panic
		bus.Unsubscribe(topic, nonexistentKey)
	})

	t.Run("Unsubscribe nonexistent topic", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("nonexistent-topic")
		key := uuid.UUID{} // Empty UUID

		// This should not panic
		bus.Unsubscribe(topic, key)
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("Concurrent subscriptions and publications", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		var wg sync.WaitGroup

		// Create a bunch of topics and handlers
		topicCount := 10
		pubCount := 5
		topics := make([]msgbus.Topic, topicCount)
		handlers := make([]msgbus.MessageHandler, topicCount)

		for i := 0; i < topicCount; i++ {
			topics[i] = msgbus.Topic(fmt.Sprintf("topic-%d", i))
			handlers[i] = make(msgbus.MessageHandler, pubCount)

			// Subscribe
			_, err := bus.Subscribe(topics[i], handlers[i])
			if err != nil {
				t.Fatalf("Failed to subscribe: %v", err)
			}
		}

		// Concurrently publish to all topics
		for i := 0; i < topicCount; i++ {
			wg.Add(1)
			go func(topicIndex int) {
				defer wg.Done()
				for j := 0; j < pubCount; j++ {
					msg := []byte(fmt.Sprintf("message-%d", j))
					bus.Publish(msgbus.TopicMessage{
						Topic:   topics[topicIndex],
						Message: msg,
					})
				}
			}(i)
		}

		// Wait for all publications to complete
		wg.Wait()

		// Verify all messages were received
		for i := 0; i < topicCount; i++ {
			for j := 0; j < pubCount; j++ {
				select {
				case msg := <-handlers[i]:
					// Verify message content if needed
					_ = msg
				case <-time.After(1 * time.Second):
					t.Errorf("Timeout waiting for message on topic %s", topics[i])
				}
			}
			// Clean up
			close(handlers[i])
		}
	})
}

func TestErrorCases(t *testing.T) {
	t.Run("Error type implements error interface", func(t *testing.T) {
		var err error = msgbus.ErrNilSubChannel
		if err.Error() != "Uninitialised subscriber channel" {
			t.Errorf("Expected error message 'Uninitialised subscriber channel', got '%s'", err.Error())
		}
	})
}

func TestTimeout(t *testing.T) {
	t.Run("Publish with slow consumer", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic := msgbus.Topic("test-topic")
		// Create unbuffered channel to simulate slow consumer
		handler := make(msgbus.MessageHandler)

		_, err := bus.Subscribe(topic, handler)
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Send multiple messages - this will fill up the channel buffer
		noMessages := 3
		receivedMessages := 0
		for i := 0; i < noMessages; i++ {
			bus.Publish(msgbus.TopicMessage{
				Topic:   topic,
				Message: []byte("test message"),
			})
		}

		// Start consuming after a delay to simulate a timeout scenario
		time.Sleep(100 * time.Millisecond)

		// Now read at least one message to verify publish is working
		select {
		case <-handler:
			// Success - we received at least one message
			receivedMessages++
		case <-time.After(6 * time.Second): // Longer than the 5s timeout in Publish
			t.Error("No message received within timeout period")
		}

		// Wait for remaining messages before closing the channel
		for i := receivedMessages; i < noMessages; i++ {
			<-handler
		}

		// Clean up
		close(handler)
	})
}

func TestMultipleTopics(t *testing.T) {
	t.Run("Subscribe to multiple topics", func(t *testing.T) {
		bus := msgbus.NewMessageBus()
		topic1 := msgbus.Topic("topic-1")
		topic2 := msgbus.Topic("topic-2")
		message1 := []byte("message 1")
		message2 := []byte("message 2")

		handler1 := make(msgbus.MessageHandler, 10)
		handler2 := make(msgbus.MessageHandler, 10)

		_, err1 := bus.Subscribe(topic1, handler1)
		_, err2 := bus.Subscribe(topic2, handler2)

		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to subscribe: %v, %v", err1, err2)
		}

		// Publish to topic 1
		bus.Publish(msgbus.TopicMessage{
			Topic:   topic1,
			Message: message1,
		})

		// Publish to topic 2
		bus.Publish(msgbus.TopicMessage{
			Topic:   topic2,
			Message: message2,
		})

		// Check handler1 receives message1
		select {
		case msg := <-handler1:
			if string(msg.Message) != string(message1) {
				t.Errorf("Handler 1: Expected message %s, got %s", message1, msg.Message)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Handler 1 didn't receive message")
		}

		// Check handler2 receives message2
		select {
		case msg := <-handler2:
			if string(msg.Message) != string(message2) {
				t.Errorf("Handler 2: Expected message %s, got %s", message2, msg.Message)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Handler 2 didn't receive message")
		}

		// Handler 1 should not receive a message for topic 2
		select {
		case msg := <-handler1:
			t.Errorf("Handler 1 received unexpected message: %v", msg)
		case <-time.After(100 * time.Millisecond):
			// Expected - no cross-topic message
		}

		// Clean up
		close(handler1)
		close(handler2)
	})
}
