package volume_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/volume"
)

func TestByID(t *testing.T) {
	v, err := volume.New(context.Background())
	volume.Cleanup(t, v)
	require.NoError(t, err)

	t.Run("default-client", func(t *testing.T) {
		vol, err := volume.FindByID(context.Background(), v.Name)
		require.NoError(t, err)

		require.Equal(t, v.Name, vol.Name)
		require.Equal(t, v.Driver, vol.Driver)
		require.Equal(t, v.Labels, vol.Labels)
		require.Equal(t, v.Mountpoint, vol.Mountpoint)
		require.Equal(t, v.Scope, vol.Scope)
		require.Equal(t, v.Status, vol.Status)
		require.Equal(t, v.UsageData, vol.UsageData)
		require.Equal(t, v.Options, vol.Options)
		require.Equal(t, v.CreatedAt, vol.CreatedAt)
		require.Equal(t, v.ClusterVolume, vol.ClusterVolume)
	})

	t.Run("with-find-client", func(t *testing.T) {
		client, err := client.New(context.Background())
		require.NoError(t, err)
		defer client.Close()

		vol, err := volume.FindByID(context.Background(), v.Name, volume.WithFindClient(client))
		require.NoError(t, err)

		require.Equal(t, v.Name, vol.Name)
	})
}

func TestList(t *testing.T) {
	t.Run("one-volume", func(t *testing.T) {
		v, err := volume.New(context.Background())
		volume.Cleanup(t, v)
		require.NoError(t, err)

		vols, err := volume.List(context.Background(), volume.WithFilters(filters.NewArgs(filters.Arg("name", v.Name))))
		require.NoError(t, err)
		require.Len(t, vols, 1)
		require.Equal(t, v.Name, vols[0].Name)
	})

	t.Run("multiple-volumes", func(t *testing.T) {
		for i := range 10 {
			v, err := volume.New(context.Background(), volume.WithName(fmt.Sprintf("test-volume-%d", i)))
			volume.Cleanup(t, v)
			require.NoError(t, err)
		}

		vols, err := volume.List(context.Background())
		require.NoError(t, err)
		require.Len(t, vols, 10)

		names := make([]string, len(vols))
		for i, v := range vols {
			names[i] = v.Name
		}

		for i := range 10 {
			require.Contains(t, names, fmt.Sprintf("test-volume-%d", i))
		}
	})

	t.Run("with-filters/labels", func(t *testing.T) {
		v, err := volume.New(context.Background(), volume.WithLabels(map[string]string{"volume.type": "test"}))
		volume.Cleanup(t, v)
		require.NoError(t, err)

		vols, err := volume.List(context.Background(), volume.WithFilters(filters.NewArgs(filters.Arg("label", "volume.type=test"))))
		require.NoError(t, err)
		require.Len(t, vols, 1)
		require.Equal(t, v.Name, vols[0].Name)
	})

	t.Run("EMPTY", func(t *testing.T) {
		vols, err := volume.List(context.Background(), volume.WithFilters(filters.NewArgs(filters.Arg("label", "volume.type=FOO"))))
		require.NoError(t, err)
		require.Empty(t, vols)
	})
}
