package volume_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/volume"
)

func TestNew(t *testing.T) {
	t.Run("with-client", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)

		v, err := volume.New(context.Background(), volume.WithClient(cli))
		require.NoError(t, err)
		volume.Cleanup(t, v)
		require.Equal(t, "local", v.Driver)

		labels := v.Labels
		require.NotEmpty(t, v.ID()) // Docker generated a random name for the volume.
		require.NotEmpty(t, v.Name) // Docker generated a random name for the volume.
		require.Equal(t, "true", labels["com.docker.sdk"])
		require.Equal(t, "go", labels["com.docker.sdk.lang"])
		require.Equal(t, client.Version(), labels["com.docker.sdk.version"])
		require.NotEmpty(t, v.Mountpoint)
	})

	t.Run("no-name", func(t *testing.T) {
		v, err := volume.New(context.Background())
		require.NoError(t, err)
		volume.Cleanup(t, v)
		require.Equal(t, "local", v.Driver)

		labels := v.Labels
		require.NotEmpty(t, v.ID()) // Docker generated a random name for the volume.
		require.NotEmpty(t, v.Name) // Docker generated a random name for the volume.
		require.Equal(t, "true", labels["com.docker.sdk"])
		require.Equal(t, "go", labels["com.docker.sdk.lang"])
		require.Equal(t, client.Version(), labels["com.docker.sdk.version"])
	})

	t.Run("with-name", func(t *testing.T) {
		v, err := volume.New(context.Background(), volume.WithName("test"))
		volume.Cleanup(t, v)
		require.NoError(t, err)
		require.Equal(t, "test", v.ID())
		require.Equal(t, "test", v.Name)
		require.Equal(t, "local", v.Driver)

		labels := v.Labels
		require.Equal(t, "true", labels["com.docker.sdk"])
		require.Equal(t, "go", labels["com.docker.sdk.lang"])
		require.Equal(t, client.Version(), labels["com.docker.sdk.version"])
	})

	t.Run("with-very-long-name", func(t *testing.T) {
		tooLongName := strings.Repeat("longname-", 256)
		v, err := volume.New(context.Background(), volume.WithName(tooLongName))
		volume.Cleanup(t, v)
		require.Error(t, err)
		require.Nil(t, v)
	})

	t.Run("with-labels", func(t *testing.T) {
		v, err := volume.New(context.Background(), volume.WithLabels(map[string]string{
			"foo": "bar",
		}))
		volume.Cleanup(t, v)
		require.NoError(t, err)
		require.NotEmpty(t, v.ID())
		require.NotEmpty(t, v.Name)
		require.Equal(t, "local", v.Driver)

		labels := v.Labels
		require.Equal(t, "bar", labels["foo"])
	})
}
