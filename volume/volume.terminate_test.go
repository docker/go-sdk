package volume_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/volume"
)

func TestVolumeTerminate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		v, err := volume.New(context.Background())
		require.NoError(t, err)
		require.NoError(t, v.Terminate(context.Background()))

		// Safe to cleanup (ErrNotFound)
		volume.Cleanup(t, v)
	})

	t.Run("with-force", func(t *testing.T) {
		v, err := volume.New(context.Background())
		require.NoError(t, err)
		require.NoError(t, v.Terminate(context.Background(), volume.WithForce()))
	})
}
