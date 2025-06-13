package dockercontainer

import (
	"sync"
	"time"
)

const (
	// StdoutLog is the log type for STDOUT
	StdoutLog = "STDOUT"

	// StderrLog is the log type for STDERR
	StderrLog = "STDERR"
)

// LogProductionOption is a function that modifies a [Container].
type LogProductionOption func(*Container)

// WithLogProductionTimeout is a functional option that sets the timeout for the log production.
// If the timeout is lower than 5s or greater than 60s it will be set to 5s or 60s respectively.
func WithLogProductionTimeout(timeout time.Duration) LogProductionOption {
	return func(c *Container) {
		c.logProductionTimeout = &timeout
	}
}

// Log represents a message that was created by a process,
// LogType is either "STDOUT" or "STDERR",
// Content is the byte contents of the message itself
type Log struct {
	LogType string
	Content []byte
}

// LogConsumer represents any object that can handle a Log.
// It is up to the LogConsumer instance what to do with the log.
type LogConsumer interface {
	Accept(Log)
}

// LogConsumerConfig is a configuration object for the producer/consumer pattern
type LogConsumerConfig struct {
	// Opts the options for the production of logs
	Opts []LogProductionOption

	// Consumers the consumers for the logs. In case you need to have a thread-safe
	// consumer, you can use [NewThreadSafeLogConsumer] to wrap the consumer.
	Consumers []LogConsumer
}

// logConsumerWriter is a writer that writes to a LogConsumer.
type logConsumerWriter struct {
	logType   string
	consumers []LogConsumer
	mu        sync.RWMutex // Protects the consumers slice
}

// newLogConsumerWriter creates a new logConsumerWriter for logType that sends messages to all consumers.
func newLogConsumerWriter(logType string, consumers []LogConsumer) *logConsumerWriter {
	return &logConsumerWriter{
		logType:   logType,
		consumers: consumers,
	}
}

// Write writes the p content to all consumers.
func (lw *logConsumerWriter) Write(p []byte) (int, error) {
	// Create a new Log for each write
	log := Log{
		LogType: lw.logType,
		Content: append([]byte(nil), p...), // Make a copy of the content
	}

	lw.mu.RLock()
	consumers := lw.consumers
	lw.mu.RUnlock()

	for _, consumer := range consumers {
		consumer.Accept(log)
	}
	return len(p), nil
}

// ThreadSafeLogConsumer wraps a LogConsumer to make it thread-safe.
// It uses a mutex to protect the Accept method from concurrent access.
type ThreadSafeLogConsumer struct {
	consumer LogConsumer
	mu       sync.Mutex
}

// NewThreadSafeLogConsumer creates a new thread-safe log consumer that wraps the given consumer.
func NewThreadSafeLogConsumer(consumer LogConsumer) *ThreadSafeLogConsumer {
	return &ThreadSafeLogConsumer{
		consumer: consumer,
	}
}

// Accept implements LogConsumer.Accept in a thread-safe way.
func (c *ThreadSafeLogConsumer) Accept(log Log) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consumer.Accept(log)
}

// Unwrap returns the underlying LogConsumer.
func (c *ThreadSafeLogConsumer) Unwrap() LogConsumer {
	return c.consumer
}
