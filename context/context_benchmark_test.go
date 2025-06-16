package context

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkContextOperations(b *testing.B) {
	setupDockerContexts(b, 1, 3) // current context is context1

	b.Run("current-context", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			current, err := Current()
			require.NoError(b, err)
			require.Equal(b, "context1", current)
		}
	})

	b.Run("current-context/context-env-override", func(b *testing.B) {
		b.Setenv(EnvOverrideContext, "context2")
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			current, err := Current()
			require.NoError(b, err)
			require.Equal(b, "context2", current)
		}
	})

	b.Run("current-context/host-env-override", func(b *testing.B) {
		b.Setenv(EnvOverrideHost, "tcp://127.0.0.1:123")
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			current, err := Current()
			require.NoError(b, err)
			require.Equal(b, DefaultContextName, current)
		}
	})

	b.Run("current-docker-host", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			host, err := CurrentDockerHost()
			require.NoError(b, err)
			require.Equal(b, "tcp://127.0.0.1:1", host)
		}
	})

	b.Run("current-docker-host/context-env-override", func(b *testing.B) {
		b.Setenv(EnvOverrideContext, "context2")
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			host, err := CurrentDockerHost()
			require.NoError(b, err)
			require.Equal(b, "tcp://127.0.0.1:2", host)
		}
	})

	b.Run("current-docker-host/host-env-override", func(b *testing.B) {
		b.Setenv(EnvOverrideHost, "tcp://127.0.0.1:123")
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			host, err := CurrentDockerHost()
			require.NoError(b, err)
			require.Equal(b, "tcp://127.0.0.1:123", host)
		}
	})

	b.Run("current-docker-host/default", func(b *testing.B) {
		b.Setenv(EnvOverrideContext, DefaultContextName)
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			host, err := CurrentDockerHost()
			require.NoError(b, err)
			require.Equal(b, DefaultDockerHost, host)
		}
	})
}

func BenchmarkContextList(b *testing.B) {
	setupDockerContexts(b, 1, 3) // Creates 3 contexts at root level

	metaDir, err := metaRoot()
	require.NoError(b, err)

	createDockerContext(b, metaDir, "nested/context", 1, "tcp://127.0.0.1:4")
	createDockerContext(b, metaDir, "nested/deep/context", 2, "tcp://127.0.0.1:5")

	b.Run("current-docker-host/nested", func(b *testing.B) {
		b.Setenv(EnvOverrideContext, "nested/context1")
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			host, err := CurrentDockerHost()
			require.NoError(b, err)
			require.Equal(b, "tcp://127.0.0.1:4", host)
		}
	})

	b.Run("current-docker-host/deep-nested", func(b *testing.B) {
		b.Setenv(EnvOverrideContext, "nested/deep/context2")
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			host, err := CurrentDockerHost()
			require.NoError(b, err)
			require.Equal(b, "tcp://127.0.0.1:5", host)
		}
	})

	b.Run("current-docker-host/not-found", func(b *testing.B) {
		b.Setenv(EnvOverrideContext, "non-existent")
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			host, err := CurrentDockerHost()
			require.Error(b, err)
			require.Empty(b, host)
		}
	})
}
