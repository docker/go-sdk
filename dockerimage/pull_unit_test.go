package dockerimage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// mockImageClient is a mock implementation of client.APIClient, which is handy for simulating
type mockImageClient struct {
	client client.APIClient
	logger *slog.Logger
}

func newMockImageClient(c client.APIClient) *mockImageClient {
	return &mockImageClient{
		client: c,
		logger: slog.Default(),
	}
}

func (m *mockImageClient) Close() error {
	return m.client.Close()
}

func (m *mockImageClient) ImagePull(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error) {
	return m.client.ImagePull(ctx, image, options)
}

func (m *mockImageClient) Logger() *slog.Logger {
	return m.logger
}

// errMockCli is a mock implementation of client.APIClient, which is handy for simulating
// error returns in retry scenarios.
type errMockCli struct {
	client.APIClient

	err            error
	imagePullCount int
}

func (f *errMockCli) ImagePull(_ context.Context, _ string, _ image.PullOptions) (io.ReadCloser, error) {
	f.imagePullCount++
	return io.NopCloser(&bytes.Buffer{}), f.err
}

func (f *errMockCli) Close() error {
	return nil
}

func TestPull(t *testing.T) {
	testPull := func(t *testing.T, errReturned error, shouldRetry bool) {
		t.Helper()

		m := &errMockCli{err: errReturned}

		mockImageClient := newMockImageClient(m)

		// give a chance to retry
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := Pull(ctx, mockImageClient, "someTag", image.PullOptions{})
		if errReturned != nil {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		defer mockImageClient.Close()

		require.Positive(t, m.imagePullCount)
		require.Equal(t, shouldRetry, m.imagePullCount > 1)
	}

	t.Run("success/no-retry", func(t *testing.T) {
		testPull(t, nil, false)
	})

	t.Run("not-available/no-retry", func(t *testing.T) {
		testPull(t, errdefs.ErrNotFound.WithMessage("not available"), false)
	})

	t.Run("invalid-parameters/no-retry", func(t *testing.T) {
		testPull(t, errdefs.ErrInvalidArgument.WithMessage("invalid"), false)
	})

	t.Run("unauthorized/retry", func(t *testing.T) {
		testPull(t, errdefs.ErrUnauthenticated.WithMessage("not authorized"), false)
	})

	t.Run("forbidden/retry", func(t *testing.T) {
		testPull(t, errdefs.ErrPermissionDenied.WithMessage("forbidden"), false)
	})

	t.Run("not-implemented/retry", func(t *testing.T) {
		testPull(t, errdefs.ErrNotImplemented.WithMessage("unknown method"), false)
	})

	t.Run("non-permanent-error/retry", func(t *testing.T) {
		testPull(t, errors.New("whoops"), true)
	})
}
