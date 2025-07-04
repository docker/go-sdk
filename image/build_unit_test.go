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
)

func TestBuild_withRetries(t *testing.T) {
	testBuild := func(t *testing.T, errReturned error, shouldRetry bool) {
		t.Helper()

		buf := &bytes.Buffer{}
		m := &errMockCli{err: errReturned, logger: slog.New(slog.NewTextHandler(buf, nil))}

		contextArchive, err := ArchiveBuildContext("testdata/retry", "Dockerfile")
		require.NoError(t, err)

		// give a chance to retry
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		tag, err := Build(
			ctx, contextArchive, "test",
			WithBuildClient(m),
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

		require.Equal(t, shouldRetry, strings.Contains(buf.String(), "Failed to build image, will retry"))
	}

	t.Run("success/no-retry", func(t *testing.T) {
		testBuild(t, nil, false)
	})

	t.Run("resource-not-found/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrNotFound.WithMessage("not available"), false)
	})

	t.Run("parameters-invalid/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrInvalidArgument.WithMessage("invalid"), false)
	})

	t.Run("access-not-authorized/no-retry", func(t *testing.T) {
		testBuild(t, errdefs.ErrUnauthenticated.WithMessage("not authorized"), false)
	})

	t.Run("access-forbidden/no-retry", func(t *testing.T) {
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
