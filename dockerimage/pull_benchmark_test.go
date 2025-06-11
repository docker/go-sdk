package dockerimage

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/stretchr/testify/require"
)

// mockImagePullClient implements ImagePullClient for testing
type mockImagePullClient struct {
	*testLogger
	pullFunc func(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error)
}

func (m *mockImagePullClient) ImagePull(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error) {
	return m.pullFunc(ctx, image, options)
}

func setupPullBenchmark(b *testing.B) *mockImagePullClient {
	return &mockImagePullClient{
		testLogger: newTestLogger(b),
		pullFunc: func(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error) {
			return io.NopCloser(io.Reader(io.MultiReader())), nil
		},
	}
}

func BenchmarkPull(b *testing.B) {
	ctx := context.Background()
	imageName := "test/image:latest"
	pullOpt := image.PullOptions{}

	b.Run("pull-without-auth", func(b *testing.B) {
		client := setupPullBenchmark(b)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := Pull(ctx, client, imageName, pullOpt)
			require.NoError(b, err)
		}
	})

	b.Run("pull-with-auth", func(b *testing.B) {
		client := setupPullBenchmark(b)
		// Mock registry credentials
		client.pullFunc = func(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error) {
			require.NotEmpty(b, options.RegistryAuth)
			return io.NopCloser(io.Reader(io.MultiReader())), nil
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := Pull(ctx, client, imageName, pullOpt)
			require.NoError(b, err)
		}
	})

	b.Run("pull-with-retries", func(b *testing.B) {
		client := setupPullBenchmark(b)
		attempts := 0
		client.pullFunc = func(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error) {
			attempts++
			if attempts < 3 {
				return nil, errors.New("temporary error")
			}
			return io.NopCloser(io.Reader(io.MultiReader())), nil
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			attempts = 0
			err := Pull(ctx, client, imageName, pullOpt)
			require.NoError(b, err)
		}
	})
}

// testLogger implements a simple logger for testing
type testLogger struct {
	t testing.TB
}

func newTestLogger(t testing.TB) *testLogger {
	return &testLogger{t: t}
}

func (l *testLogger) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
