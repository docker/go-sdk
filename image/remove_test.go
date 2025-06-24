package image_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

func TestRemove(t *testing.T) {
	img := "redis:alpine"

	t.Run("success", func(t *testing.T) {
		pullImage(t, img)

		resp, err := image.Remove(context.Background(), img)
		require.NoError(t, err)
		require.NotEmpty(t, resp)
	})

	t.Run("success/with-client", func(t *testing.T) {
		pullImage(t, img)

		dockerClient, err := client.New(context.Background())
		require.NoError(t, err)

		resp, err := image.Remove(context.Background(), img, image.WithRemoveClient(dockerClient))
		require.NoError(t, err)
		require.NotEmpty(t, resp)
	})

	t.Run("success/with-options", func(t *testing.T) {
		pullImage(t, img)

		resp, err := image.Remove(context.Background(), img, image.WithRemoveOptions(dockerimage.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		}))
		require.NoError(t, err)
		require.NotEmpty(t, resp)
	})

	t.Run("error/blank-image", func(t *testing.T) {
		pullImage(t, img)

		resp, err := image.Remove(context.Background(), "")
		require.Error(t, err)
		require.Empty(t, resp)
	})
}

func pullImage(t *testing.T, img string) {
	t.Helper()

	err := image.Pull(context.Background(), img)
	require.NoError(t, err)
}
