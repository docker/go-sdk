package volume_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-sdk/volume"
)

func BenchmarkVolumeOperations(b *testing.B) {
	ctx := context.Background()

	b.Run("create-volume", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			v, err := volume.New(ctx)
			volume.Cleanup(b, v)
			require.NoError(b, err)
		}
	})

	b.Run("create-volume-with-options", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			v, err := volume.New(
				ctx,
				volume.WithName("test-volume"),
				volume.WithLabels(map[string]string{
					"test": "benchmark",
					"env":  "bench",
				}),
			)
			volume.Cleanup(b, v)
			require.NoError(b, err)
		}
	})

	b.Run("find-by-id", func(b *testing.B) {
		for i := range 10 {
			v, err := volume.New(ctx, volume.WithName(fmt.Sprintf("test-volume-%d", i)))
			volume.Cleanup(b, v)
			require.NoError(b, err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			// always retrieve the first volume by name
			_, err := volume.FindByID(context.Background(), "test-volume-0")
			require.NoError(b, err)
		}
	})

	b.Run("list-volumes", func(b *testing.B) {
		for i := range 10 {
			v, err := volume.New(ctx, volume.WithName(fmt.Sprintf("test-volume-%d", i)))
			volume.Cleanup(b, v)
			require.NoError(b, err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := volume.List(context.Background())
			require.NoError(b, err)
		}
	})

	b.Run("list-volumes/filters", func(b *testing.B) {
		for range 10 {
			v, err := volume.New(ctx, volume.WithLabels(map[string]string{"volume.type": "test"}))
			volume.Cleanup(b, v)
			require.NoError(b, err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_, err := volume.List(context.Background(), volume.WithFilters(filters.NewArgs(filters.Arg("label", "volume.type=test"))))
			require.NoError(b, err)
		}
	})
}

func BenchmarkVolumeConcurrent(b *testing.B) {
	b.Run("concurrent-volume-creation", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				v, err := volume.New(context.Background())
				volume.Cleanup(b, v)
				require.NoError(b, err)
			}
		})
	})

	b.Run("concurrent-volume-by-id", func(b *testing.B) {
		v, err := volume.New(context.Background())
		volume.Cleanup(b, v)
		require.NoError(b, err)

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := volume.FindByID(context.Background(), v.Name)
				require.NoError(b, err)
			}
		})
	})

	b.Run("concurrent-volume-termination", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				v, err := volume.New(context.Background())
				require.NoError(b, err)

				b.StartTimer()
				err = v.Terminate(context.Background())
				b.StopTimer()

				require.NoError(b, err)
			}
		})
	})
}
