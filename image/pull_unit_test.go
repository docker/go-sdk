package image

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/image"
	sdkclient "github.com/docker/go-sdk/client"
)

func TestPull(t *testing.T) {
	defaultPullOpts := []PullOption{WithPullOptions(image.PullOptions{})}

	testPull := func(t *testing.T, imageName string, pullOpts []PullOption, mockCli *errMockCli, shouldRetry bool) string {
		t.Helper()
		buf := &bytes.Buffer{}

		if len(pullOpts) > 0 && mockCli != nil {
			sdk, err := sdkclient.New(context.TODO(),
				sdkclient.WithDockerAPI(mockCli),
				sdkclient.WithLogger(slog.New(slog.NewTextHandler(buf, nil))))
			require.NoError(t, err)

			pullOpts = append(pullOpts, WithPullClient(sdk))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := Pull(ctx, imageName, pullOpts...)
		if mockCli.err != nil {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		defer mockCli.Close()

		// Only validate the retry logic if there are more than 1 pull option.
		if len(pullOpts) > 1 {
			require.Positive(t, mockCli.imagePullCount)
			require.Equal(t, shouldRetry, mockCli.imagePullCount > 1)
		}
		return buf.String()
	}

	t.Run("error/no-image", func(t *testing.T) {
		testPull(t, "", []PullOption{}, &errMockCli{err: errors.New("image name is not set")}, false)
	})

	t.Run("error/no-client", func(t *testing.T) {
		testPull(t, "someTag", []PullOption{}, &errMockCli{err: errors.New("image name is not set")}, false)
	})

	t.Run("success/no-retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: nil}, false)
	})

	t.Run("not-available/no-retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrNotFound.WithMessage("not available")}, false)
	})

	t.Run("invalid-parameters/no-retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrInvalidArgument.WithMessage("invalid")}, false)
	})

	t.Run("unauthorized/retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrUnauthenticated.WithMessage("not authorized")}, false)
	})

	t.Run("forbidden/retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrPermissionDenied.WithMessage("forbidden")}, false)
	})

	t.Run("not-implemented/retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrNotImplemented.WithMessage("unknown method")}, false)
	})

	t.Run("non-permanent-error/retry", func(t *testing.T) {
		mockCliWithLogger := &errMockCli{
			err: errors.New("whoops"),
		}

		out := testPull(t, "some_tag", defaultPullOpts, mockCliWithLogger, true)
		require.Contains(t, out, "failed to pull image, will retry")
	})
}
