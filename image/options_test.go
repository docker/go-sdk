package image

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
)

func TestWithOptions(t *testing.T) {
	t.Run("with-pull-client", func(t *testing.T) {
		pullClient := &mockImagePullClient{}
		sdk, err := client.New(context.TODO(), client.WithDockerAPI(pullClient))
		require.NoError(t, err)
		pullOpts := &pullOptions{}
		err = WithPullClient(sdk)(pullOpts)
		require.NoError(t, err)
		require.Equal(t, sdk, pullOpts.client)
	})

	t.Run("with-pull-options", func(t *testing.T) {
		opts := image.PullOptions{}
		pullOpts := &pullOptions{}
		err := WithPullOptions(opts)(pullOpts)
		require.NoError(t, err)
		require.Equal(t, opts, pullOpts.pullOptions)
	})
}
