package container

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/container"
)

// Logger returns the logger for the container.
func (c *Container) Logger() *slog.Logger {
	return c.logger
}

// Logs will fetch both STDOUT and STDERR from the current container. Returns a
// ReadCloser and leaves it up to the caller to extract what it wants.
func (c *Container) Logs(ctx context.Context) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}

	rc, err := c.dockerClient.ContainerLogs(ctx, c.ID(), options)
	if err != nil {
		return nil, err
	}

	// Check if the container has TTY enabled, to determine the log format
	inspect, err := c.Inspect(ctx)
	if err != nil {
		rc.Close()
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	// If TTY is enabled, logs are not multiplexed - return them directly
	if inspect.Config.Tty {
		return rc, nil
	}

	// TTY is disabled, logs are multiplexed with stream headers - parse them
	return c.parseMultiplexedLogs(rc), nil
}

// parseMultiplexedLogs handles the multiplexed log format used when TTY is disabled
func (c *Container) parseMultiplexedLogs(rc io.ReadCloser) io.ReadCloser {
	const streamHeaderSize = 8

	pr, pw := io.Pipe()
	r := bufio.NewReader(rc)

	go func() {
		defer rc.Close()

		var closeErr error
		defer func() {
			if r := recover(); r != nil {
				closeErr = fmt.Errorf("panic in log processing: %v", r)
			}

			if closeErr != nil && !errors.Is(closeErr, io.EOF) {
				// Real error, close the pipe with the error
				if err := pw.CloseWithError(closeErr); err != nil {
					c.logger.Debug("failed to close pipe writer with error", "error", err, "original", closeErr)
				}
			} else {
				// No error or EOF, close the pipe normally
				if err := pw.Close(); err != nil {
					c.logger.Debug("failed to close pipe writer", "error", err)
				}
			}
		}()

		// Process the Docker multiplexed stream format which includes:
		// - Byte 0: Stream type (1 = stdout, 2 = stderr)
		// - Bytes 1-3: Reserved
		// - Bytes 4-7: Frame size (big-endian uint32)
		streamHeader := make([]byte, streamHeaderSize)

		for {
			// Read complete stream header - ensures all 8 bytes are read
			if _, err := io.ReadFull(r, streamHeader); err != nil {
				closeErr = err
				break
			}

			// Extract frame size from header
			frameSize := binary.BigEndian.Uint32(streamHeader[4:])

			// Copy frame data
			if _, err := io.CopyN(pw, r, int64(frameSize)); err != nil {
				closeErr = err
				break
			}
		}
	}()

	return pr
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
