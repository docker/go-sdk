package image_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

func TestSave(t *testing.T) {
	img := "redis:alpine"

	pullImage(t, img)

	t.Run("success", func(t *testing.T) {
		output := filepath.Join(t.TempDir(), "images.tar")
		err := image.Save(context.Background(), output, img)
		require.NoError(t, err)

		info, err := os.Stat(output)
		require.NoError(t, err)

		require.NotZero(t, info.Size())
	})

	t.Run("success/with-client", func(t *testing.T) {
		output := filepath.Join(t.TempDir(), "images.tar")

		dockerClient, err := client.New(context.Background())
		require.NoError(t, err)

		err = image.Save(context.Background(), output, img, image.WithSaveClient(dockerClient))
		require.NoError(t, err)
	})

	t.Run("success/with-platforms", func(t *testing.T) {
		output := filepath.Join(t.TempDir(), "images.tar")
		err := image.Save(context.Background(), output, img, image.WithPlatforms(ocispec.Platform{
			OS:           "linux",
			Architecture: "amd64",
		}))
		require.NoError(t, err)
	})

	t.Run("error/no-output", func(t *testing.T) {
		err := image.Save(context.Background(), "", img)
		require.Error(t, err)
	})

	t.Run("error/no-image", func(t *testing.T) {
		err := image.Save(context.Background(), filepath.Join(t.TempDir(), "images.tar"), "")
		require.Error(t, err)
	})
}
