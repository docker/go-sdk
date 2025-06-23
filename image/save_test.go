package image_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/image"
)

func TestSave(t *testing.T) {
	img := "redis:alpine"

	err := image.Pull(context.Background(), img)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		output := filepath.Join(t.TempDir(), "images.tar")
		err = image.Save(context.Background(), output, img)
		require.NoError(t, err)

		info, err := os.Stat(output)
		require.NoError(t, err)

		require.NotZero(t, info.Size())
	})

	t.Run("error/no-output", func(t *testing.T) {
		err = image.Save(context.Background(), "", img)
		require.Error(t, err)
	})

	t.Run("error/no-image", func(t *testing.T) {
		err = image.Save(context.Background(), filepath.Join(t.TempDir(), "images.tar"), "")
		require.Error(t, err)
	})
}
