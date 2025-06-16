package dockerimage

import (
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/stretchr/testify/require"
)

func TestWithOptions(t *testing.T) {
	t.Run("with-pull-client", func(t *testing.T) {
		pullClient := &mockImagePullClient{}
		pullOpts := &pullOptions{}
		WithPullClient(pullClient)(pullOpts)
		require.Equal(t, pullClient, pullOpts.pullClient)
	})

	t.Run("with-pull-options", func(t *testing.T) {
		opts := image.PullOptions{}
		pullOpts := &pullOptions{}
		WithPullOptions(opts)(pullOpts)
		require.Equal(t, opts, pullOpts.pullOptions)
	})
}
