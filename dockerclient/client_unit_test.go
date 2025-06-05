package dockerclient

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockercontext"
)

func TestNew_internal_state(t *testing.T) {
	t.Run("debug-host-resolution", func(t *testing.T) {
		t.Setenv(dockercontext.EnvOverrideContext, "default")

		// Get the host before creating the client
		host, err := dockercontext.CurrentDockerHost()
		t.Logf("Docker host before client creation: %q, error: %v", host, err)

		cli, err := New(context.Background())
		if err != nil {
			t.Logf("Client creation error: %v", err)
		}
		require.NoError(t, err)
		require.NotNil(t, cli)

		// Log the actual host being used
		if cli.cfg != nil {
			t.Logf("Client config host: %q", cli.cfg.Host)
		}
	})

	t.Run("success", func(t *testing.T) {
		client, err := New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, client)

		require.Empty(t, client.extraHeaders)
		require.NotNil(t, client.cfg)
		require.NotNil(t, client.client)
		require.Nil(t, client.log)
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
