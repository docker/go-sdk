package dockercontainer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTerminateOptions(t *testing.T) {
	t.Run("with-stop-timeout", func(t *testing.T) {
		opts := NewTerminateOptions(context.Background(), TerminateTimeout(10*time.Second))
		require.Equal(t, 10*time.Second, opts.StopTimeout())
	})

	t.Run("with-volumes", func(t *testing.T) {
		opts := NewTerminateOptions(context.Background(), RemoveVolumes("vol1", "vol2"))
		require.Equal(t, []string{"vol1", "vol2"}, opts.volumes)
	})

	t.Run("with-stop-timeout-and-volumes", func(t *testing.T) {
		opts := NewTerminateOptions(context.Background(), TerminateTimeout(10*time.Second), RemoveVolumes("vol1", "vol2"))
		require.Equal(t, 10*time.Second, opts.StopTimeout())
		require.Equal(t, []string{"vol1", "vol2"}, opts.volumes)
	})

	t.Run("with-stop-timeout-and-volumes-and-context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		opts := NewTerminateOptions(ctx, TerminateTimeout(10*time.Second), RemoveVolumes("vol1", "vol2"))
		require.Equal(t, ctx, opts.Context())
	})
}
