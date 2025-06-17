package network_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/network"
)

func BenchmarkNetworkOperations(b *testing.B) {
	ctx := context.Background()

	b.Run("create-network", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			nw, err := network.New(ctx)
			network.Cleanup(b, nw)
			require.NoError(b, err)
		}
	})

	b.Run("create-network-with-options", func(b *testing.B) {
		opts := []network.Option{
			network.WithInternal(),
			network.WithEnableIPv6(),
			network.WithAttachable(),
			network.WithLabels(map[string]string{
				"test": "benchmark",
				"env":  "bench",
			}),
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			nw, err := network.New(ctx, opts...)
			network.Cleanup(b, nw)
			require.NoError(b, err)
		}
	})

	b.Run("inspect-network", func(b *testing.B) {
		nw, err := network.New(ctx)
		network.Cleanup(b, nw)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := nw.Inspect(ctx)
			require.NoError(b, err)
		}
	})

	b.Run("inspect-network-with-cache", func(b *testing.B) {
		nw, err := network.New(ctx)
		network.Cleanup(b, nw)
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
		nw, err := network.New(ctx)
		network.Cleanup(b, nw)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := nw.Inspect(ctx, network.WithNoCache())
			require.NoError(b, err)
		}
	})

	b.Run("terminate-network", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			nw, err := network.New(ctx)
			network.Cleanup(b, nw)
			require.NoError(b, err)
			require.NoError(b, nw.Terminate(ctx))
		}
	})

	b.Run("create-and-terminate-network", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			nw, err := network.New(ctx)
			network.Cleanup(b, nw)
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
				nw, err := network.New(ctx)
				network.Cleanup(b, nw)
				require.NoError(b, err)
			}
		})
	})

	b.Run("concurrent-network-inspection-with-cache", func(b *testing.B) {
		nw, err := network.New(ctx)
		network.Cleanup(b, nw)
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
		nw, err := network.New(ctx)
		network.Cleanup(b, nw)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := nw.Inspect(ctx, network.WithNoCache())
				require.NoError(b, err)
			}
		})
	})

	b.Run("concurrent-network-termination", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				nw, err := network.New(ctx)
				require.NoError(b, err)
				require.NoError(b, nw.Terminate(ctx))
			}
		})
	})
}
