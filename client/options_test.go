package client

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithOptions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Run("docker-host", func(t *testing.T) {
			cli := &Client{}
			require.NoError(t, WithDockerHost("tcp://localhost:2375").Apply(cli))
			require.Equal(t, "tcp://localhost:2375", cli.dockerHost)
		})

		t.Run("docker-context", func(t *testing.T) {
			cli := &Client{}
			require.NoError(t, WithDockerContext("test-context").Apply(cli))
			require.Equal(t, "test-context", cli.dockerContext)
		})

		t.Run("extra-headers", func(t *testing.T) {
			cli := &Client{}
			require.NoError(t, WithExtraHeaders(map[string]string{"X-Test": "test"}).Apply(cli))
			require.Equal(t, map[string]string{"X-Test": "test"}, cli.extraHeaders)
		})

		t.Run("health-check", func(t *testing.T) {
			cli := &Client{}
			require.NoError(t, WithHealthCheck(func(_ context.Context) func(_ *Client) error {
				return nil
			}).Apply(cli))
			require.NotNil(t, cli.healthCheck)
		})

		t.Run("logger", func(t *testing.T) {
			cli := &Client{}
			require.NoError(t, WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))).Apply(cli))
			require.NotNil(t, cli.log)
		})
	})
}
