package dockercontainer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStopOptions(t *testing.T) {
	t.Run("with-stop-timeout", func(t *testing.T) {
		opts := NewStopOptions(context.Background(), StopTimeout(10*time.Second))
		require.Equal(t, 10*time.Second, opts.StopTimeout())
	})

	t.Run("with-stop-timeout-and-context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		opts := NewStopOptions(ctx, StopTimeout(10*time.Second))
		require.Equal(t, ctx, opts.Context())
	})
}
