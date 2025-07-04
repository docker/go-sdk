package container

import (
	"bufio"
	"context"
	"encoding/binary"
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
		header := make([]byte, streamHeaderSize)
		for err == nil {
			_, errH := r.Read(header)
			if errH != nil {
				_ = pw.CloseWithError(err)
				return
			}

			frameSize := binary.BigEndian.Uint32(header[4:])
			if _, err := io.CopyN(pw, r, int64(frameSize)); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	return pr, nil
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
