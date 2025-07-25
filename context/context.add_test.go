package context_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/context"
)

func TestNew(t *testing.T) {
	t.Run("empty-name", func(tt *testing.T) {
		_, err := context.New("")
		require.Error(tt, err)
	})

	t.Run("default-name", func(tt *testing.T) {
		_, err := context.New("default")
		require.Error(tt, err)
	})

	t.Run("error/meta-root", func(tt *testing.T) {
		tt.Setenv("HOME", tt.TempDir())
		tt.Setenv("USERPROFILE", tt.TempDir()) // Windows support

		_, err := context.New("test")
		require.Error(tt, err)
	})

	t.Run("success", func(t *testing.T) {
		context.SetupTestDockerContexts(t, 1, 3)

		t.Run("no-current", func(tt *testing.T) {
			ctx, err := context.New(
				"test1234",
				context.WithHost("tcp://127.0.0.1:1234"),
				context.WithDescription("test description"),
				context.WithAdditionalFields(map[string]any{"testKey": "testValue"}),
			)
			require.NoError(tt, err)
			defer func() {
				require.NoError(tt, ctx.Delete())
			}()

			list, err := context.List()
			require.NoError(tt, err)
			require.Contains(tt, list, ctx.Name)

			require.Equal(tt, "test1234", ctx.Name)
			require.Equal(tt, "test description", ctx.Metadata.Description)
			require.Equal(tt, "tcp://127.0.0.1:1234", ctx.Endpoints["docker"].Host)
			require.False(tt, ctx.Endpoints["docker"].SkipTLSVerify)

			fields := ctx.Metadata.Fields()
			require.Equal(tt, map[string]any{"testKey": "testValue"}, fields)

			value, exists := fields["testKey"]
			require.True(tt, exists)
			require.Equal(tt, "testValue", value)

			// the current context is not the new one
			current, err := context.Current()
			require.NoError(tt, err)
			require.NotEqual(tt, "test1234", current)
		})

		t.Run("as-current", func(tt *testing.T) {
			ctx, err := context.New("test1234", context.WithHost("tcp://127.0.0.1:1234"), context.AsCurrent())
			require.NoError(tt, err)
			defer func() {
				require.NoError(tt, ctx.Delete())
			}()

			list, err := context.List()
			require.NoError(tt, err)
			require.Contains(tt, list, ctx.Name)

			require.Equal(tt, "test1234", ctx.Name)
			require.Equal(tt, "tcp://127.0.0.1:1234", ctx.Endpoints["docker"].Host)
			require.False(tt, ctx.Endpoints["docker"].SkipTLSVerify)

			// the current context is the new one
			current, err := context.Current()
			require.NoError(tt, err)
			require.Equal(tt, "test1234", current)
		})
	})
}
