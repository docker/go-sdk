package context

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkCurrentContext(b *testing.B) {
	SetupTestDockerContexts(b, 1, 3) // current context is context1

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

func BenchmarkCurrentNestedContext(b *testing.B) {
	SetupTestDockerContexts(b, 1, 3) // Creates 3 contexts at root level

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
}

func BenchmarkContextAdd(b *testing.B) {
	SetupTestDockerContexts(b, 1, 3) // Creates 3 contexts at root level

	b.Run("success", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := range b.N {
			b.StartTimer()
			ctx, err := New(fmt.Sprintf("benchmark-%d", i))
			require.NoError(b, err)
			b.StopTimer()

			require.NoError(b, ctx.Delete())
		}
	})

	b.Run("as-current", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := range b.N {
			b.StartTimer()
			ctx, err := New(fmt.Sprintf("benchmark-current-%d", i), AsCurrent())
			require.NoError(b, err)
			b.StopTimer()

			require.NoError(b, ctx.Delete())
		}
	})
}

func BenchmarkContextDelete(b *testing.B) {
	SetupTestDockerContexts(b, 1, 3) // Creates 3 contexts at root level

	b.Run("success", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := range b.N {
			ctx, err := New(fmt.Sprintf("benchmark-delete-%d", i))
			require.NoError(b, err)

			b.StartTimer()
			require.NoError(b, ctx.Delete())
			b.StopTimer()
		}
	})
}

func BenchmarkContextList(b *testing.B) {
	SetupTestDockerContexts(b, 1, 3) // Creates 3 contexts at root level

	b.Run("context-list", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			_, err := List()
			require.NoError(b, err)
		}
	})
}

func BenchmarkContextInspect(b *testing.B) {
	SetupTestDockerContexts(b, 1, 3) // Creates 3 contexts at root level

	b.Run("context-inspect", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			_, err := Inspect("context1")
			require.NoError(b, err)
		}
	})

	b.Run("context-inspect/not-found", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for range b.N {
			_, err := Inspect("non-existent")
			require.ErrorIs(b, err, ErrDockerContextNotFound)
		}
	})
}
