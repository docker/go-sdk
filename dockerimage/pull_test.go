package dockerimage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockerimage"
)

func TestPull(t *testing.T) {
	ctx := context.Background()
	dockerClient, err := dockerclient.New(ctx)
	require.NoError(t, err)
	defer dockerClient.Close()

	err = dockerimage.Pull(ctx,
		"nginx:alpine",
		dockerimage.WithPullClient(dockerClient),
		dockerimage.WithPullOptions(image.PullOptions{}),
	)
	require.NoError(t, err)
}
