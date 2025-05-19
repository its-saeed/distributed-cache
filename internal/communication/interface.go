package communication

// Publisher defines the publishing interface
type Publisher interface {
	Publish(topic string, msg Message) error
}

// Subscriber defines the subscription interface
type Subscriber interface {
	Subscribe(topic string, handler MessageHandler)
	Unsubscribe(topic string, handler MessageHandler)
}

// PubSubber combines both interfaces
type PubSubber interface {
	Publisher
	Subscriber
	Close() error
}
