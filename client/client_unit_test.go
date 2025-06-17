package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
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
}
