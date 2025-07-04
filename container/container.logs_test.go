package container_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/container/wait"
)

func TestContainer_Logs_fromFailedContainer(t *testing.T) {
	ctx := context.Background()
	c, err := container.Run(
		ctx,
		container.WithImage(alpineLatest),
		container.WithCmd("echo", "-n", "I was not expecting this"),
		container.WithWaitStrategy(wait.ForLog("I was expecting this").WithTimeout(5*time.Second)),
	)

	container.Cleanup(t, c)
	require.ErrorContains(t, err, "container exited with code 0")

	logs, logErr := c.Logs(ctx)
	require.NoError(t, logErr)

	b, err := io.ReadAll(logs)
	require.NoError(t, err)

	log := string(b)
	require.Contains(t, log, "I was not expecting this")
}

func TestContainer_Logs_shouldBeWithoutStreamHeader(t *testing.T) {
	ctx := context.Background()
	ctr, err := container.Run(ctx,
		container.WithImage(alpineLatest),
		container.WithCmd("sh", "-c", "echo 'abcdefghi' && echo 'foo'"),
		container.WithWaitStrategy(wait.ForExit()),
	)
	container.Cleanup(t, ctr)
	require.NoError(t, err)

	r, err := ctr.Logs(ctx)
	require.NoError(t, err)
	defer r.Close()

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "abcdefghi\nfoo", strings.TrimSpace(string(b)))
}

func TestContainer_Logs_shouldStripHeadersFromStderr(t *testing.T) {
	ctx := context.Background()
	ctr, err := container.Run(ctx,
		container.WithImage(alpineLatest),
		container.WithCmd("sh", "-c", "echo 'stdout line' && echo 'stderr line' 1>&2"),
		container.WithWaitStrategy(wait.ForExit()),
	)
	container.Cleanup(t, ctr)
	require.NoError(t, err)

	r, err := ctr.Logs(ctx)
	require.NoError(t, err)
	defer r.Close()

	b, err := io.ReadAll(r)
	require.NoError(t, err)

	logs := strings.TrimSpace(string(b))

	// Both stdout and stderr should be present without stream headers
	require.Contains(t, logs, "stdout line")
	require.Contains(t, logs, "stderr line")

	// Verify no binary stream headers are present in the output
	// Stream headers start with 0x01 (stdout) or 0x02 (stderr)
	require.NotContains(t, logs, "\x01")
	require.NotContains(t, logs, "\x02")
}

func TestContainer_Logs_printOnError(t *testing.T) {
	ctx := context.Background()

	buf := new(bytes.Buffer)
	logger := slog.New(slog.NewTextHandler(buf, nil))

	cli, err := client.New(ctx, client.WithLogger(logger))
	require.NoError(t, err)

	ctr, err := container.Run(ctx,
		container.WithDockerClient(cli),
		container.WithImage(alpineLatest),
		container.WithCmd("echo", "-n", "I am expecting this"),
		container.WithWaitStrategy(wait.ForLog("I was expecting that").WithTimeout(5*time.Second)),
	)
	container.Cleanup(t, ctr)
	// it should fail because the waiting for condition is not met
	require.Error(t, err)

	containerLogs, err := ctr.Logs(ctx)
	require.NoError(t, err)
	defer containerLogs.Close()

	// read container logs line by line, checking that each line is present in the client's logger
	rd := bufio.NewReader(containerLogs)
	for {
		line, err := rd.ReadString('\n')

		// Process the line if we have data, even if there's an EOF error
		if line != "" {
			// the last line of the array should contain the line of interest,
			// but we are checking all the lines to make sure that is present
			found := false
			for _, l := range strings.Split(buf.String(), "\n") {
				if strings.Contains(l, line) {
					found = true
					break
				}
			}
			require.True(t, found, "container log line not found in the output of the logger: %s", line)
		}

		// Check for errors after processing any data
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoErrorf(t, err, "Read Error")
	}
}

func TestContainer_Logs_TTYEnabled(t *testing.T) {
	ctx := context.Background()
	ctr, err := container.Run(ctx,
		container.WithImage(alpineLatest),
		container.WithCmd("sh", "-c", "echo 'tty output'"),
		container.WithConfigModifier(func(cfg *dockercontainer.Config) {
			cfg.Tty = true
		}),
		container.WithWaitStrategy(wait.ForExit()),
	)
	container.Cleanup(t, ctr)
	require.NoError(t, err)

	r, err := ctr.Logs(ctx)
	require.NoError(t, err)
	defer r.Close()

	b, err := io.ReadAll(r)
	require.NoError(t, err)

	logs := strings.TrimSpace(string(b))
	require.Contains(t, logs, "tty output")
}
