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
	pull := func(t *testing.T, dockerClient *client.Client) {
		t.Helper()

		ctx := context.Background()

		err := image.Pull(ctx,
			"nginx:alpine",
			image.WithPullClient(dockerClient),
			image.WithPullOptions(apiimage.PullOptions{}),
		)
		require.NoError(t, err)
	}

	t.Run("new-client", func(t *testing.T) {
		dockerClient, err := client.New(context.Background())
		require.NoError(t, err)
		defer dockerClient.Close()

		pull(t, dockerClient)
	})

	t.Run("default-client", func(t *testing.T) {
		pull(t, client.DefaultClient)
	})
}
