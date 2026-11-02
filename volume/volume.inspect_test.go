package volume_test

import (
	"context"
	"testing"

	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/volume"
)

func TestInspect(t *testing.T) {
	t.Run("volume-exists", func(t *testing.T) {
		v, err := volume.New(context.Background())
		volume.Cleanup(t, v)
		require.NoError(t, err)

		inspect, err := v.Inspect(context.Background(), dockerclient.VolumeInspectOptions{})
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.Volume.Labels, client.LabelBase)
		require.Contains(t, inspect.Volume.Labels, client.LabelLang)
		require.Contains(t, inspect.Volume.Labels, client.LabelVersion)
		require.Contains(t, inspect.Volume.Labels, client.LabelBase+".volume")
		require.Equal(t, volume.Version(), inspect.Volume.Labels[client.LabelBase+".volume"])
	})

	t.Run("volume-does-not-exist", func(t *testing.T) {
		v := &volume.Volume{}

		inspect, err := v.Inspect(context.Background(), dockerclient.VolumeInspectOptions{})
		require.Error(t, err)
		require.Zero(t, inspect)
	})
}
