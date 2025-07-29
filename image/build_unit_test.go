package image

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/go-sdk/client"
)

func TestBuild_withRetries(t *testing.T) {
	testBuild := func(t *testing.T, errReturned error, shouldRetry bool) {
		t.Helper()

		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))
		m := &errMockCli{err: errReturned}

		sdk, err := client.New(context.TODO(), client.WithDockerAPI(m), client.WithLogger(logger))
		require.NoError(t, err)

		contextArchive, err := ArchiveBuildContext("testdata/retry", "Dockerfile")
		require.NoError(t, err)

		// give a chance to retry
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		tag, err := Build(
			ctx, contextArchive, "test",
			WithBuildClient(sdk),
			WithBuildOptions(build.ImageBuildOptions{
				Dockerfile: "Dockerfile",
			}),
		)
		if errReturned != nil {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, "test", tag)
		}

		require.Positive(t, m.imageBuildCount)
		require.Equal(t, shouldRetry, m.imageBuildCount > 1)

		s := buf.String()
		require.Equal(t, shouldRetry, strings.Contains(s, "Failed to build image, will retry"))
	}

	t.Run("success/no-retry", func(t *testing.T) {
		testBuild(t, nil, false)
	})

	t.Run("not-available/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrNotFound.WithMessage("not available"), false)
	})

	t.Run("invalid-parameters/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrInvalidArgument.WithMessage("invalid"), false)
	})

	t.Run("unauthorized/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrUnauthenticated.WithMessage("not authorized"), false)
	})

	t.Run("forbidden/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrPermissionDenied.WithMessage("forbidden"), false)
	})

	t.Run("not-implemented/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrNotImplemented.WithMessage("unknown method"), false)
	})

	t.Run("system-error/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrInternal.WithMessage("system error"), false)
	})

	t.Run("permanent-error/retry", func(t *testing.T) {
		testBuild(t, errors.New("whoops"), true)
	})
}
