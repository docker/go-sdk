package dockercontainer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

var (
	// minLogProductionTimeout is the minimum log production timeout.
	minLogProductionTimeout = time.Duration(5 * time.Second)

	// maxLogProductionTimeout is the maximum log production timeout.
	maxLogProductionTimeout = time.Duration(60 * time.Second)

	// errLogProductionStop is the cause for stopping log production.
	errLogProductionStop = errors.New("log production stopped")
)

// Logger returns the logger for the container.
func (c *Container) Logger() *slog.Logger {
	return c.logger
}

// Logs will fetch both STDOUT and STDERR from the current container. Returns a
// ReadCloser and leaves it up to the caller to extract what it wants.
func (c *Container) Logs(ctx context.Context) (io.ReadCloser, error) {
	const streamHeaderSize = 8

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}

	rc, err := c.dockerClient.ContainerLogs(ctx, c.ID(), options)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	r := bufio.NewReader(rc)

	go func() {
		lineStarted := true
		for err == nil {
			line, isPrefix, err := r.ReadLine()

			if lineStarted && len(line) >= streamHeaderSize {
				line = line[streamHeaderSize:] // trim stream header
				lineStarted = false
			}
			if !isPrefix {
				lineStarted = true
			}

			_, errW := pw.Write(line)
			if errW != nil {
				return
			}

			if !isPrefix {
				_, errW := pw.Write([]byte("\n"))
				if errW != nil {
					return
				}
			}

			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}
	}()

	return pr, nil
}

// GetLogProductionErrorChannel exposes the only way for the consumer
// to be able to listen to errors and react to them.
func (c *Container) GetLogProductionErrorChannel() <-chan error {
	if c.logProductionCtx == nil {
		return nil
	}

	errCh := make(chan error, 1)
	go func(ctx context.Context) {
		<-ctx.Done()
		errCh <- context.Cause(ctx)
		close(errCh)
	}(c.logProductionCtx)

	return errCh
}

// copyLogs copies logs from the container to stdout and stderr.
func (c *Container) copyLogs(ctx context.Context, stdout, stderr io.Writer, options container.LogsOptions) error {
	rc, err := c.dockerClient.ContainerLogs(ctx, c.ID(), options)
	if err != nil {
		return fmt.Errorf("container logs: %w", err)
	}
	defer rc.Close()

	if _, err = stdcopy.StdCopy(stdout, stderr, rc); err != nil {
		return fmt.Errorf("stdcopy: %w", err)
	}

	return nil
}

// copyLogsTimeout copies logs from the container to stdout and stderr with a timeout.
// It returns true if the log production should be retried, false otherwise.
func (c *Container) copyLogsTimeout(stdout, stderr io.Writer, options *container.LogsOptions) bool {
	timeoutCtx, cancel := context.WithTimeout(c.logProductionCtx, *c.logProductionTimeout)
	defer cancel()

	err := c.copyLogs(timeoutCtx, stdout, stderr, *options)
	switch {
	case err == nil:
		// No more logs available.
		return false
	case c.logProductionCtx.Err() != nil:
		// Log production was stopped or caller context is done.
		return false
	case timeoutCtx.Err() != nil, errors.Is(err, net.ErrClosed):
		// Timeout or client connection closed, retry.
	default:
		// Unexpected error, retry.
		c.logger.Error("Unexpected error reading logs", "error", err)
	}

	// Retry from the last log received.
	now := time.Now()
	options.Since = fmt.Sprintf("%d.%09d", now.Unix(), int64(now.Nanosecond()))

	return true
}

// followOutput adds a LogConsumer to be sent logs from the container's
// STDOUT and STDERR
func (c *Container) followOutput(consumer LogConsumer) {
	c.consumers = append(c.consumers, consumer)
}

// logProducer read logs from the container and writes them to stdout, stderr until either:
//   - logProductionCtx is done
//   - A fatal error occurs
//   - No more logs are available
func (c *Container) logProducer(stdout, stderr io.Writer) {
	// Setup the log options, start from the beginning.
	options := &container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	// Use a separate method so that timeout cancel function is
	// called correctly.
	for c.copyLogsTimeout(stdout, stderr, options) {
	}
}

// printLogs is a helper function that will print the logs of a Docker container
// We are going to use this helper function to inform the user of the logs when an error occurs
func (c *Container) printLogs(ctx context.Context, cause error) {
	reader, err := c.Logs(ctx)
	if err != nil {
		c.logger.Error("failed accessing container logs", "error", err)
		return
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		if len(b) > 0 {
			c.logger.Error("failed reading container logs", "error", err, "cause", cause, "logs", b)
		} else {
			c.logger.Error("failed reading container logs", "error", err, "cause", cause)
		}
		return
	}

	c.logger.Info("container logs", "cause", cause, "logs", b)
}

// startLogProduction will start a concurrent process that will continuously read logs
// from the container and will send them to each added LogConsumer.
//
// Default log production timeout is 5s. It is used to set the context timeout
// which means that each log-reading loop will last at up to the specified timeout.
//
// Use functional option WithLogProductionTimeout() to override default timeout. If it's
// lower than 5s and greater than 60s it will be set to 5s or 60s respectively.
func (c *Container) startLogProduction(ctx context.Context, opts ...LogProductionOption) error {
	for _, opt := range opts {
		opt(c)
	}

	// Validate the log production timeout.
	switch {
	case c.logProductionTimeout == nil:
		c.logProductionTimeout = &minLogProductionTimeout
	case *c.logProductionTimeout < minLogProductionTimeout:
		c.logProductionTimeout = &minLogProductionTimeout
	case *c.logProductionTimeout > maxLogProductionTimeout:
		c.logProductionTimeout = &maxLogProductionTimeout
	}

	// Get a snapshot of current consumers
	consumers := make([]LogConsumer, len(c.consumers))
	copy(consumers, c.consumers)

	// Setup the log writers.
	stdout := newLogConsumerWriter(StdoutLog, consumers)
	stderr := newLogConsumerWriter(StderrLog, consumers)

	// Setup the log production context which will be used to stop the log production.
	c.logProductionCtx, c.logProductionCancel = context.WithCancelCause(ctx)

	// We capture context cancel function to avoid data race with multiple
	// calls to startLogProduction.
	go func() {
		defer c.logProductionCancel(nil)

		c.logProducer(stdout, stderr)
	}()

	return nil
}

// stopLogProduction will stop the concurrent process that is reading logs
// and sending them to each added LogConsumer
func (c *Container) stopLogProduction() error {
	if c.logProductionCancel == nil {
		return nil
	}

	// Signal the log production to stop.
	c.logProductionCancel(errLogProductionStop)

	// Wait for the log producer to finish with timeout
	select {
	case <-c.logProductionCtx.Done():
		// Check the context cause after the context is done
		if err := context.Cause(c.logProductionCtx); err != nil {
			switch {
			case errors.Is(err, errLogProductionStop):
				// Log production was stopped normally.
				return nil
			case errors.Is(err, context.DeadlineExceeded),
				errors.Is(err, context.Canceled):
				// Parent context is done.
				return nil
			default:
				// Unexpected error
				return err
			}
		}
		return nil
	case <-time.After(maxLogProductionTimeout):
		return fmt.Errorf("timeout waiting for log producer to stop: %w", context.Cause(c.logProductionCtx))
	}
}
