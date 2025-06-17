package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	dockercontext "github.com/docker/go-sdk/context"
)

func TestNew_internal_state(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, err := New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, client)

		require.Empty(t, client.extraHeaders)
		require.NotNil(t, client.cfg)
		require.NotNil(t, client.dockerClient)
		require.NotNil(t, client.log)
		require.Equal(t, slog.New(slog.NewTextHandler(io.Discard, nil)), client.log)
		require.False(t, client.dockerInfoSet)
		require.Empty(t, client.dockerInfo)
		require.NoError(t, client.err)
	})

	t.Run("with-headers", func(t *testing.T) {
		client, err := New(context.Background(), WithExtraHeaders(map[string]string{"X-Test": "test"}))
		require.NoError(t, err)
		require.NotNil(t, client)

		require.Equal(t, map[string]string{"X-Test": "test"}, client.extraHeaders)
	})

	t.Run("with-logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		client, err := New(context.Background(), WithLogger(logger))
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, logger, client.log)
	})

	t.Run("with-healthcheck", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		logger := slog.New(slog.NewTextHandler(buf, nil))

		healthcheck := func(_ context.Context) func(*Client) error {
			return func(c *Client) error {
				c.Logger().Info("healthcheck")
				return nil
			}
		}

		client, err := New(context.Background(), WithHealthCheck(healthcheck), WithLogger(logger))
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, logger, client.log)
		require.Contains(t, buf.String(), "healthcheck")
	})

	t.Run("with-healthcheck-error", func(t *testing.T) {
		healthcheck := func(_ context.Context) func(*Client) error {
			return func(_ *Client) error {
				return errors.New("healthcheck error")
			}
		}

		client, err := New(context.Background(), WithHealthCheck(healthcheck))
		require.ErrorContains(t, err, "healthcheck error")
		require.Nil(t, client)
	})

	t.Run("with-dockerhost", func(t *testing.T) {
		noopHealthCheck := func(_ context.Context) func(*Client) error {
			return func(_ *Client) error {
				// NOOP for testing
				return nil
			}
		}

		client, err := New(context.Background(), WithHealthCheck(noopHealthCheck), WithDockerHost("unix:///var/run/docker.sock"))
		require.NoError(t, err)
		require.NotNil(t, client)
		require.Equal(t, "unix:///var/run/docker.sock", client.cfg.Host)
	})

	t.Run("with-dockerhost-and-dockercontext", func(t *testing.T) {
		noopHealthCheck := func(_ context.Context) func(*Client) error {
			return func(_ *Client) error {
				// NOOP for testing
				return nil
			}
		}

		// current context is context1
		dockercontext.SetupTestDockerContexts(t, 1, 1)

		client, err := New(
			context.Background(),
			WithHealthCheck(noopHealthCheck),
			WithDockerHost("wont-be-used"),
			WithDockerContext("context1"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)

		// the docker host from the context takes precedence over the one set with WithDockerHost
		require.Equal(t, "tcp://127.0.0.1:1", client.cfg.Host)
	})

	t.Run("with-dockercontext", func(t *testing.T) {
		noopHealthCheck := func(_ context.Context) func(*Client) error {
			return func(_ *Client) error {
				// NOOP for testing
				return nil
			}
		}

		// current context is context1
		dockercontext.SetupTestDockerContexts(t, 1, 1)

		client, err := New(
			context.Background(),
			WithHealthCheck(noopHealthCheck),
			WithDockerContext("context1"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)

		// the docker host from the context is used
		require.Equal(t, "tcp://127.0.0.1:1", client.cfg.Host)
	})

	t.Run("with-docker-context/not-existing", func(t *testing.T) {
		noopHealthCheck := func(_ context.Context) func(*Client) error {
			return func(_ *Client) error {
				// NOOP for testing
				return nil
			}
		}

		// the test context does not exist, so the client creation fails
		client, err := New(context.Background(), WithHealthCheck(noopHealthCheck), WithDockerContext("test"))
		require.ErrorContains(t, err, "docker host from context")
		require.Nil(t, client)
	})
}
