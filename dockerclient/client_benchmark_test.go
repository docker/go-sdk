package dockerclient_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/client"
	"github.com/docker/go-sdk/dockerclient"
)

func BenchmarkNew(b *testing.B) {
	b.Run("default", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cli, err := dockerclient.New(context.Background())
			require.NoError(b, err)
			require.NoError(b, cli.Close())
		}
	})

	b.Run("with-host", func(b *testing.B) {
		opt := dockerclient.FromDockerOpt(client.WithHost("tcp://localhost:2375"))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cli, err := dockerclient.New(context.Background(), opt)
			require.NoError(b, err)
			require.NoError(b, cli.Close())
		}
	})

	b.Run("with-logger", func(b *testing.B) {
		opt := dockerclient.WithLogger(nil) // Using nil logger for benchmark
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cli, err := dockerclient.New(context.Background(), opt)
			require.NoError(b, err)
			require.NoError(b, cli.Close())
		}
	})

	b.Run("with-healthcheck", func(b *testing.B) {
		noopHealthCheck := func(_ context.Context) func(c *dockerclient.Client) error {
			return func(_ *dockerclient.Client) error {
				return nil
			}
		}
		opt := dockerclient.WithHealthCheck(noopHealthCheck)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cli, err := dockerclient.New(context.Background(), opt)
			require.NoError(b, err)
			require.NoError(b, cli.Close())
		}
	})
}

func BenchmarkClientConcurrentCreation(b *testing.B) {
	b.Run("parallel-creation", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cli, err := dockerclient.New(context.Background())
				require.NoError(b, err)
				require.NoError(b, cli.Close())
			}
		})
	})

	b.Run("shared-client", func(b *testing.B) {
		cli, err := dockerclient.New(context.Background())
		require.NoError(b, err)
		defer cli.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Just access the client to test concurrent access
				_ = cli.Client
			}
		})
	})
}

func BenchmarkClientClose(b *testing.B) {
	b.Run("sequential-close", func(b *testing.B) {
		cli, err := dockerclient.New(context.Background())
		require.NoError(b, err)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			require.NoError(b, cli.Close())
		}
	})

	b.Run("concurrent-close", func(b *testing.B) {
		cli, err := dockerclient.New(context.Background())
		require.NoError(b, err)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				require.NoError(b, cli.Close())
			}
		})
	})
}
