package image_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	apiimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

var noopShowProgress = func(_ io.ReadCloser) error {
	return nil
}

func TestPull(t *testing.T) {
	pull := func(t *testing.T, dockerClient *client.Client, expectedErr error, opts ...image.PullOption) {
		t.Helper()

		opts = append(opts, image.WithPullClient(dockerClient))
		opts = append(opts, image.WithPullOptions(apiimage.PullOptions{}))

		ctx := context.Background()

		err := image.Pull(ctx, "nginx:alpine", opts...)
		if expectedErr != nil {
			require.ErrorContains(t, err, expectedErr.Error())
		} else {
			require.NoError(t, err)
		}
	}

	t.Run("new-client", func(t *testing.T) {
		dockerClient, err := client.New(context.Background())
		require.NoError(t, err)
		defer dockerClient.Close()

		pull(t, dockerClient, nil)
	})

	t.Run("default-client", func(t *testing.T) {
		pull(t, client.DefaultClient, nil)
	})

	t.Run("pull-handler/nil", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		defer cli.Close()

		pull(t, cli, errors.New("pull handler is nil"), image.WithPullHandler(nil))
	})

	t.Run("pull-handler/noop", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		defer cli.Close()

		pull(t, cli, nil, image.WithPullHandler(noopShowProgress))
	})

	t.Run("pull-handler/custom", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cli)

		buf := &bytes.Buffer{}

		pull(t, cli, nil, image.WithPullHandler(func(r io.ReadCloser) error {
			_, err := io.Copy(buf, r)
			defer func() {
				if err := r.Close(); err != nil {
					t.Logf("failed to close reader: %v", err)
				}
			}()
			return err
		}))

		require.Contains(t, buf.String(), "Pulling from library/nginx")
	})
}
