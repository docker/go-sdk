package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-sdk/client"
)

func BenchmarkNew(b *testing.B) {
	b.Run("default", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			cli, err := client.New(context.Background())
			require.NoError(b, err)
			require.NoError(b, cli.Close())
		}
	})

	b.Run("with-host", func(b *testing.B) {
		opt := client.FromDockerOpt(dockerclient.WithHost("tcp://localhost:2375"))
		b.ResetTimer()
		for range b.N {
			cli, err := client.New(context.Background(), opt)
			require.NoError(b, err)
			require.NoError(b, cli.Close())
		}
	})

	b.Run("with-logger", func(b *testing.B) {
		opt := client.WithLogger(nil) // Using nil logger for benchmark
		b.ResetTimer()
		for range b.N {
			cli, err := client.New(context.Background(), opt)
			require.NoError(b, err)
			require.NoError(b, cli.Close())
		}
	})

	b.Run("with-healthcheck", func(b *testing.B) {
		noopHealthCheck := func(_ context.Context) func(c *client.Client) error {
			return func(_ *client.Client) error {
				return nil
			}
		}
		opt := client.WithHealthCheck(noopHealthCheck)
		b.ResetTimer()
		for range b.N {
			cli, err := client.New(context.Background(), opt)
			require.NoError(b, err)
			require.NoError(b, cli.Close())
		}
	})
}

func BenchmarkDefaultClient(b *testing.B) {
	b.Run("default", func(b *testing.B) {
		b.ResetTimer()
		for range b.N {
			cli := client.DefaultClient
			require.NoError(b, cli.Close())
		}
	})
}

func BenchmarkClientConcurrentCreation(b *testing.B) {
	b.Run("parallel-creation", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cli, err := client.New(context.Background())
				require.NoError(b, err)
				require.NoError(b, cli.Close())
			}
		})
	})

	b.Run("shared-client", func(b *testing.B) {
		cli, err := client.New(context.Background())
		require.NoError(b, err)
		defer cli.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Just access the client to test concurrent access
				_, err := cli.Client()
				require.NoError(b, err)
			}
		})
	})

	b.Run("shared-default-client", func(b *testing.B) {
		cli := client.DefaultClient
		defer cli.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Just access the client to test concurrent access
				_, err := cli.Client()
				require.NoError(b, err)
			}
		})
	})
}

func BenchmarkClientClose(b *testing.B) {
	b.Run("sequential-close", func(b *testing.B) {
		cli, err := client.New(context.Background())
		require.NoError(b, err)
		b.ResetTimer()
		for range b.N {
			require.NoError(b, cli.Close())
		}
	})

	b.Run("concurrent-close", func(b *testing.B) {
		cli, err := client.New(context.Background())
		require.NoError(b, err)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				require.NoError(b, cli.Close())
			}
		})
	})

	b.Run("concurrent-close-default-client", func(b *testing.B) {
		cli := client.DefaultClient
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				require.NoError(b, cli.Close())
			}
		})
	})
}
