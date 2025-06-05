package dockercontainer

import (
	"strings"
	"testing"
	"time"
)

// FooLogConsumer is a test log consumer that accepts logs from the
// "hello-world" Docker image, which prints out the "Hello from Docker!"
// log message.
type FooLogConsumer struct {
	LogChannel chan string
	t          *testing.T
}

// Accept receives a log message and sends it to the log channel if it
// contains the "Hello from Docker!" message.
func (c FooLogConsumer) Accept(rawLog Log) {
	log := string(rawLog.Content)
	if strings.Contains(log, "Hello from Docker!") {
		select {
		case c.LogChannel <- log:
		default:
		}
	}
}

// AssertRead waits for a log message to be received.
func (c FooLogConsumer) AssertRead() {
	select {
	case <-c.LogChannel:
	case <-time.After(5 * time.Second):
		c.t.Fatal("receive timeout")
	}
}

// SlurpOne reads a value from the channel if it is available.
func (c FooLogConsumer) SlurpOne() {
	select {
	case <-c.LogChannel:
	default:
	}
}

func NewFooLogConsumer(t *testing.T) *FooLogConsumer {
	t.Helper()

	return &FooLogConsumer{
		t:          t,
		LogChannel: make(chan string, 2),
	}
}
