package container

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container/wait"
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

type testLogConsumer struct {
	logs []string
}

func (l *testLogConsumer) Accept(log Log) {
	l.logs = append(l.logs, string(log.Content))
}

func (l *testLogConsumer) Logs() []string {
	return l.logs
}

func TestRestartContainerWithLogConsumer(t *testing.T) {
	logConsumer := NewFooLogConsumer(t)

	ctx := context.Background()

	ctr, err := Run(ctx,
		WithImage("hello-world"),
		WithAlwaysPull(),
		WithLogConsumerConfig(&LogConsumerConfig{
			Consumers: []LogConsumer{logConsumer},
		}),
		WithNoStart(),
	)
	Cleanup(t, ctr)
	require.NoError(t, err)

	// Start and confirm that the log consumer receives the log message.
	err = ctr.Start(ctx)
	require.NoError(t, err)

	logConsumer.AssertRead()

	// Stop the container and clear any pending message.
	err = ctr.Stop(ctx, StopTimeout(5*time.Second))
	require.NoError(t, err)

	logConsumer.SlurpOne()

	// Restart the container and confirm that the log consumer receives new log messages.
	err = ctr.Start(ctx)
	require.NoError(t, err)

	// First message is from the first start.
	logConsumer.AssertRead()
	logConsumer.AssertRead()
}

func TestLogConsumers_multiple(t *testing.T) {
	ctx := context.Background()
	consumer1 := &testLogConsumer{}
	consumer2 := &testLogConsumer{}

	ctr, err := Run(ctx,
		WithImage("alpine:latest"),
		WithCmd("sh", "-c", `for i in $(seq 1 50); do echo "test log $i"; done`),
		WithLogConsumerConfig(&LogConsumerConfig{
			Consumers: []LogConsumer{consumer1, consumer2},
		}),
		WithWaitStrategy(wait.ForLog("test log 50")), // Wait for the last log message
	)
	require.NoError(t, err)
	require.NoError(t, ctr.Terminate(ctx))

	// Verify logs for both consumers, but right after the container is terminated
	// else we might get a race condition because the log consumer could still be
	// writing to the log consumer.
	logs1 := consumer1.Logs()
	logs2 := consumer2.Logs()
	require.Len(t, logs1, 50)
	require.Len(t, logs2, 50)
	for i := range 50 {
		require.Equal(t, logs1[i], logs2[i])
	}
}
