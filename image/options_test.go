package image

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/image"
)

func TestWithOptions(t *testing.T) {
	t.Run("with-pull-client", func(t *testing.T) {
		pullClient := &mockImagePullClient{}
		pullOpts := &pullOptions{}
		err := WithPullClient(pullClient)(pullOpts)
		require.NoError(t, err)
		require.Equal(t, pullClient, pullOpts.pullClient)
	})

	t.Run("with-pull-options", func(t *testing.T) {
		opts := image.PullOptions{}
		pullOpts := &pullOptions{}
		err := WithPullOptions(opts)(pullOpts)
		require.NoError(t, err)
		require.Equal(t, opts, pullOpts.pullOptions)
	})
}
