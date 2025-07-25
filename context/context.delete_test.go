package context_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/context"
)

func TestDelete(t *testing.T) {
	context.SetupTestDockerContexts(t, 1, 3)

	t.Run("success", func(tt *testing.T) {
		ctx, err := context.New("test", context.WithHost("tcp://127.0.0.1:1234"))
		require.NoError(tt, err)
		require.NoError(tt, ctx.Delete())

		list, err := context.List()
		require.NoError(tt, err)
		require.NotContains(tt, list, ctx.Name)

		got, err := context.Inspect(ctx.Name)
		require.ErrorIs(tt, err, context.ErrDockerContextNotFound)
		require.Empty(tt, got)
	})

	t.Run("error/encoded-name", func(tt *testing.T) {
		context.SetupTestDockerContexts(tt, 1, 3)

		ctx := context.Context{
			Name: "test",
		}

		err := ctx.Delete()
		require.ErrorContains(tt, err, "context has no encoded name")
	})
}
