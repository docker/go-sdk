package image_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apiimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

func TestPull(t *testing.T) {
	ctx := context.Background()
	dockerClient, err := client.New(ctx)
	require.NoError(t, err)
	defer dockerClient.Close()

	err = image.Pull(ctx,
		"nginx:alpine",
		image.WithPullClient(dockerClient),
		image.WithPullOptions(apiimage.PullOptions{}),
	)
	require.NoError(t, err)
}
