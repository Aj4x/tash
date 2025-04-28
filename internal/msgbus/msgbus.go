package msgbus

import (
	"context"
	"fmt"
	"github.com/Aj4x/tash/internal/uuid"
	"sync"
	"time"
)

// Error represents a textual error value that implements the error interface.
type Error string

// Error returns the string representation of the Error type. It satisfies the error interface.
func (e Error) Error() string {
	return string(e)
}

// ErrNilSubChannel represents an error occurring when a subscriber channel is uninitialized.
// ErrGeneratingKey represents an error that occurs while generating a key.
const (
	ErrNilSubChannel = Error("Uninitialised subscriber channel")
	ErrGeneratingKey = Error("Error generating key")
)

// Topic represents a category or channel for messages in a publish-subscribe system.
type Topic string

// TopicMessage represents a message linked to a specific topic within a pub/sub system or message bus.
// The Topic field defines the subject, and Message holds the message payload as a byte slice.
type TopicMessage[T any] struct {
	Topic   Topic
	Message T
}

// MessageHandler is a channel used to handle incoming TopicMessage objects for a specific subscription. It allows processing messages in a concurrent manner.
type MessageHandler[T any] chan TopicMessage[T]

// subscription represents a registration to a specific Topic with a unique Key and a Handler to process incoming messages for the Topic.
type subscription[T any] struct {
	Topic   Topic
	Key     uuid.UUID
	Handler MessageHandler[T]
}

// publish sends a TopicMessage to the associated MessageHandler channel of the subscription.
func (s *subscription[T]) publish(msg TopicMessage[T]) {
	s.Handler <- msg
}

// Publisher is an interface for publishing messages to a specified topic.
// It provides the `Publish` method, which accepts a `TopicMessage` for delivery.
// Typically used in messaging systems to distribute messages across subscribers.
type Publisher[T any] interface {
	Publish(msg TopicMessage[T])
}

// Subscriber defines behaviour for consuming messages from specific topics with unique handlers.
// The Subscribe method registers a handler for a topic and returns a unique identifier or an error.
type Subscriber[T any] interface {
	Subscribe(topic Topic, handler MessageHandler[T]) (uuid.UUID, error)
}

// Unsubscriber defines an interface for removing a subscription from a specified topic using a unique identifier.
type Unsubscriber interface {
	Unsubscribe(topic Topic, key uuid.UUID)
}

// PublisherSubscriber is an interface that combines publishing, subscribing, and unsubscribing functionalities for a message-bus system.
type PublisherSubscriber[T any] interface {
	Publisher[T]
	Subscriber[T]
	Unsubscriber
}

// messageBus is a struct implementing a publisher-subscriber mechanism with concurrency control.
// It maintains a map of topics to a list of subscriptions and ensures thread-safe access via a mutex.
type messageBus[T any] struct {
	subscribers map[Topic][]subscription[T]
	subLock     sync.Mutex
}

// NewMessageBus creates and initialises a new instance of a message bus implementing the PublisherSubscriber interface.
func NewMessageBus[T any]() PublisherSubscriber[T] {
	return &messageBus[T]{
		subscribers: make(map[Topic][]subscription[T]),
	}
}

// Publish sends a TopicMessage to all subscribers of the specified topic, using a goroutine for each subscriber, with a timeout of 5 seconds for publishing.
func (m *messageBus[T]) Publish(msg TopicMessage[T]) {
	m.subLock.Lock()
	defer m.subLock.Unlock()
	subscriptions, ok := m.subscribers[msg.Topic]
	if !ok {
		return
	}
	publish := func(s subscription[T], ctx context.Context, cancel context.CancelFunc) {
		select {
		case <-ctx.Done():
			cancel()
			return
		default:
			s.publish(msg)
			fmt.Printf("published msg to %s\n", s.Topic)
		}
		cancel()
	}
	for _, sub := range subscriptions {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		go publish(sub, ctx, cancel)
	}
}

// Subscribe registers a handler to a specific topic and returns a unique identifier for the subscription or an error if registration fails.
func (m *messageBus[T]) Subscribe(topic Topic, handler MessageHandler[T]) (uuid.UUID, error) {
	if handler == nil {
		return uuid.UUID{}, ErrNilSubChannel
	}
	key, err := uuid.NewUUID()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%w: %w", ErrGeneratingKey, err)
	}
	s := subscription[T]{
		Topic:   topic,
		Key:     key,
		Handler: handler,
	}
	m.subLock.Lock()
	defer m.subLock.Unlock()
	m.subscribers[s.Topic] = append(m.subscribers[s.Topic], s)
	return key, nil
}

// Unsubscribe removes a subscription identified by a topic and its unique key from the message bus.
func (m *messageBus[T]) Unsubscribe(topic Topic, key uuid.UUID) {
	m.subLock.Lock()
	defer m.subLock.Unlock()
	subscriptions, ok := m.subscribers[topic]
	if !ok {
		return
	}
	for i, subscription := range subscriptions {
		if subscription.Key == key {
			if len(subscriptions) == 1 {
				delete(m.subscribers, topic)
				fmt.Printf("removed topic %s, no more subscribers\n", topic)
				break
			}
			m.subscribers[topic] = append(subscriptions[:i], subscriptions[i+1:]...)
			fmt.Printf("removed topic %s, %d subscribers remaining\n", topic, len(m.subscribers[topic]))
			break
		}
	}
}
