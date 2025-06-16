package client

import (
	"context"
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
		require.NotNil(t, client.Client)
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
}
