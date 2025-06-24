package container_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container"
)

func BenchmarkPortEndpoint(b *testing.B) {
	ctr, err := container.Run(context.Background(),
		container.WithImage(nginxAlpineImage),
	)
	container.Cleanup(b, ctr)
	require.NoError(b, err)
	require.NotNil(b, ctr)

	b.Run("port-endpoint", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := ctr.PortEndpoint(context.Background(), "80/tcp", "tcp")
			require.NoError(b, err)
		}
	})

	b.Run("mapped-port", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := ctr.MappedPort(context.Background(), "80/tcp")
			require.NoError(b, err)
		}
	})

	b.Run("endpoint", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := ctr.Endpoint(context.Background(), "tcp")
			require.NoError(b, err)
		}
	})
}
