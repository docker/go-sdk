package dockernetwork_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockernetwork"
)

func BenchmarkNetworkOperations(b *testing.B) {
	ctx := context.Background()

	b.Run("create-network", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			nw, err := dockernetwork.New(ctx)
			dockernetwork.CleanupNetwork(b, nw)
			require.NoError(b, err)
		}
	})

	b.Run("create-network-with-options", func(b *testing.B) {
		opts := []dockernetwork.Option{
			dockernetwork.WithInternal(),
			dockernetwork.WithEnableIPv6(),
			dockernetwork.WithAttachable(),
			dockernetwork.WithLabels(map[string]string{
				"test": "benchmark",
				"env":  "bench",
			}),
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			nw, err := dockernetwork.New(ctx, opts...)
			dockernetwork.CleanupNetwork(b, nw)
			require.NoError(b, err)
		}
	})

	b.Run("inspect-network", func(b *testing.B) {
		nw, err := dockernetwork.New(ctx)
		dockernetwork.CleanupNetwork(b, nw)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := nw.Inspect(ctx)
			require.NoError(b, err)
		}
	})

	b.Run("inspect-network-with-cache", func(b *testing.B) {
		nw, err := dockernetwork.New(ctx)
		dockernetwork.CleanupNetwork(b, nw)
		require.NoError(b, err)

		// First inspect to populate cache
		_, err = nw.Inspect(ctx)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := nw.Inspect(ctx)
			require.NoError(b, err)
		}
	})

	b.Run("inspect-network-without-cache", func(b *testing.B) {
		nw, err := dockernetwork.New(ctx)
		dockernetwork.CleanupNetwork(b, nw)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := nw.Inspect(ctx, dockernetwork.WithNoCache())
			require.NoError(b, err)
		}
	})

	b.Run("terminate-network", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			nw, err := dockernetwork.New(ctx)
			dockernetwork.CleanupNetwork(b, nw)
			require.NoError(b, err)
			require.NoError(b, nw.Terminate(ctx))
		}
	})

	b.Run("create-and-terminate-network", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			nw, err := dockernetwork.New(ctx)
			dockernetwork.CleanupNetwork(b, nw)
			require.NoError(b, err)
			require.NoError(b, nw.Terminate(ctx))
		}
	})
}

func BenchmarkNetworkConcurrent(b *testing.B) {
	ctx := context.Background()

	b.Run("concurrent-network-creation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				nw, err := dockernetwork.New(ctx)
				dockernetwork.CleanupNetwork(b, nw)
				require.NoError(b, err)
			}
		})
	})

	b.Run("concurrent-network-inspection-with-cache", func(b *testing.B) {
		nw, err := dockernetwork.New(ctx)
		dockernetwork.CleanupNetwork(b, nw)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := nw.Inspect(ctx)
				require.NoError(b, err)
			}
		})
	})

	b.Run("concurrent-network-inspection-with-no-cache", func(b *testing.B) {
		nw, err := dockernetwork.New(ctx)
		dockernetwork.CleanupNetwork(b, nw)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := nw.Inspect(ctx, dockernetwork.WithNoCache())
				require.NoError(b, err)
			}
		})
	})

	b.Run("concurrent-network-termination", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				nw, err := dockernetwork.New(ctx)
				require.NoError(b, err)
				require.NoError(b, nw.Terminate(ctx))
			}
		})
	})
}
