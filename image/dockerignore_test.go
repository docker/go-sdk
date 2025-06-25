package image_test

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/image"
)

func TestParseDockerIgnore(t *testing.T) {
	parse := func(t *testing.T, filePath string, expectedExists bool, expectedErr error, expectedExcluded []string) {
		t.Helper()

		exists, excluded, err := image.ParseDockerIgnore(filePath)
		require.Equal(t, expectedExists, exists)
		require.ErrorIs(t, expectedErr, err)
		require.Equal(t, expectedExcluded, excluded)
	}

	t.Run("dockerignore", func(t *testing.T) {
		parse(t, path.Join("testdata", "dockerignore"), true, nil, []string{"vendor", "foo", "bar"})
	})

	t.Run("no-dockerignore", func(t *testing.T) {
		parse(t, path.Join("testdata", "retry"), false, nil, nil)
	})
}
