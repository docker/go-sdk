package image

import (
	"context"
	"errors"
	"io"
	"iter"
	"log/slog"
	"testing"

	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	sdkclient "github.com/docker/go-sdk/client"
)

// mockImagePullClient implements ImagePullClient for testing
type mockImagePullClient struct {
	client.APIClient
	*testLogger
	pullFunc func(ctx context.Context, image string, options client.ImagePullOptions) (client.ImagePullResponse, error)
}

func (m *mockImagePullClient) Close() error {
	return nil
}

func (m *mockImagePullClient) ImagePull(ctx context.Context, image string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
	return m.pullFunc(ctx, image, options)
}

func (m *mockImagePullClient) Ping(_ context.Context, _ client.PingOptions) (client.PingResult, error) {
	return client.PingResult{}, nil
}

func setupPullBenchmark(b *testing.B) *mockImagePullClient {
	b.Helper()

	return &mockImagePullClient{
		testLogger: newTestLogger(b),
		pullFunc: func(_ context.Context, _ string, _ client.ImagePullOptions) (client.ImagePullResponse, error) {
			return mockImagePullResponse{io.NopCloser(io.MultiReader())}, nil
		},
	}
}

type mockImagePullResponse struct {
	io.ReadCloser
}

func (f mockImagePullResponse) JSONMessages(_ context.Context) iter.Seq2[jsonstream.Message, error] {
	return nil
}

func (f mockImagePullResponse) Wait(_ context.Context) error {
	return nil
}

func BenchmarkPull(b *testing.B) {
	ctx := context.Background()
	imageName := "test/image:latest"
	pullOpt := client.ImagePullOptions{}

	b.Run("pull-without-auth", func(b *testing.B) {
		c := setupPullBenchmark(b)
		b.ResetTimer()
		b.ReportAllocs()

		sdk, err := sdkclient.New(context.TODO(), sdkclient.WithDockerAPI(c))
		require.NoError(b, err)

		for range b.N {
			err := Pull(ctx, imageName, WithPullClient(sdk), WithPullOptions(pullOpt), WithCredentialsFn(func(_ string) (string, string, error) {
				return "", "", nil
			}))
			require.NoError(b, err)
		}
	})

	b.Run("pull-with-auth", func(b *testing.B) {
		c := setupPullBenchmark(b)
		// Mock registry credentials
		c.pullFunc = func(_ context.Context, _ string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
			require.NotEmpty(b, options.RegistryAuth)
			return mockImagePullResponse{io.NopCloser(io.MultiReader())}, nil
		}
		b.ResetTimer()
		b.ReportAllocs()

		sdk, err := sdkclient.New(context.TODO(), sdkclient.WithDockerAPI(c))
		require.NoError(b, err)

		for range b.N {
			err := Pull(ctx, imageName, WithPullClient(sdk), WithPullOptions(pullOpt), WithCredentialsFn(func(_ string) (string, string, error) {
				return "user", "pass", nil
			}))
			require.NoError(b, err)
		}
	})

	b.Run("pull-with-retries", func(b *testing.B) {
		c := setupPullBenchmark(b)
		attempts := 0
		c.pullFunc = func(_ context.Context, _ string, _ client.ImagePullOptions) (client.ImagePullResponse, error) {
			attempts++
			if attempts < 3 {
				return nil, errors.New("temporary error")
			}
			return mockImagePullResponse{io.NopCloser(io.MultiReader())}, nil
		}
		b.ResetTimer()
		b.ReportAllocs()

		sdk, err := sdkclient.New(context.TODO(), sdkclient.WithDockerAPI(c))
		require.NoError(b, err)

		// Use a custom pull handler that discards output to avoid measuring display performance
		discardHandler := func(r io.ReadCloser) error {
			_, err := io.Copy(io.Discard, r)
			return err
		}

		for range b.N {
			attempts = 0
			err := Pull(ctx, imageName, WithPullClient(sdk), WithPullOptions(pullOpt), WithPullHandler(discardHandler), WithCredentialsFn(func(_ string) (string, string, error) {
				return "", "", nil
			}))
			require.NoError(b, err)
		}
	})

	b.Run("with-pull-handler", func(b *testing.B) {
		c := setupPullBenchmark(b)
		// Mock registry credentials
		c.pullFunc = func(_ context.Context, _ string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
			require.NotEmpty(b, options.RegistryAuth)
			return mockImagePullResponse{io.NopCloser(io.MultiReader())}, nil
		}
		b.ResetTimer()
		b.ReportAllocs()

		sdk, err := sdkclient.New(context.TODO(), sdkclient.WithDockerAPI(c))
		require.NoError(b, err)

		for range b.N {
			err := Pull(ctx, imageName, WithPullClient(sdk), WithPullOptions(pullOpt), WithPullHandler(func(r io.ReadCloser) error {
				_, err := io.Copy(io.Discard, r)
				return err
			}), WithCredentialsFn(func(_ string) (string, string, error) {
				return "user", "pass", nil
			}))
			require.NoError(b, err)
		}
	})
}

// testLogger implements a simple logger for testing
type testLogger struct {
	t testing.TB
}

func newTestLogger(tb testing.TB) *testLogger {
	tb.Helper()

	return &testLogger{t: tb}
}

func (l *testLogger) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
