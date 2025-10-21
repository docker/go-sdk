package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cli)

		info, err := cli.Info(context.Background())
		require.NoError(t, err)
		require.NotNil(t, info)
	})

	t.Run("success/info-cached", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cli)

		info1, err := cli.Info(context.Background())
		require.NoError(t, err)
		require.NotNil(t, info1)

		info2, err := cli.Info(context.Background())
		require.NoError(t, err)
		require.NotNil(t, info2)

		require.Equal(t, info1, info2)
	})

	t.Run("client", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cli)
	})

	t.Run("close", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cli)

		// multiple calls to Close() are idempotent
		require.NoError(t, cli.Close())
		require.NoError(t, cli.Close())
	})
}
