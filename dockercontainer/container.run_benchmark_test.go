package dockercontainer_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockercontainer"
)

// BenchmarkRunContainer measures container creation time
func BenchmarkRunContainer(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	b.Run("minimal", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithImage(nginxAlpineImage),
		})
	})

	b.Run("with-env", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithEnv(map[string]string{
				"ENV1": "value1",
				"ENV2": "value2",
			}),
		})
	})

	b.Run("with-ports", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithExposedPorts("80/tcp", "443/tcp"),
		})
	})

	b.Run("with-lifecycle-hooks", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithLifecycleHooks(dockercontainer.LifecycleHooks{
				PreCreates: []dockercontainer.DefinitionHook{
					func(_ context.Context, _ *dockercontainer.Definition) error {
						return nil
					},
				},
				PostCreates: []dockercontainer.ContainerHook{
					func(_ context.Context, _ *dockercontainer.Container) error {
						return nil
					},
				},
				PreStarts: []dockercontainer.ContainerHook{
					func(_ context.Context, _ *dockercontainer.Container) error {
						return nil
					},
				},
				PostStarts: []dockercontainer.ContainerHook{
					func(_ context.Context, _ *dockercontainer.Container) error {
						return nil
					},
				},
				PostReadies: []dockercontainer.ContainerHook{
					func(_ context.Context, _ *dockercontainer.Container) error {
						return nil
					},
				},
				PreStops: []dockercontainer.ContainerHook{
					func(_ context.Context, _ *dockercontainer.Container) error {
						return nil
					},
				},
				PostStops: []dockercontainer.ContainerHook{
					func(_ context.Context, _ *dockercontainer.Container) error {
						return nil
					},
				},
				PreTerminates: []dockercontainer.ContainerHook{
					func(_ context.Context, _ *dockercontainer.Container) error {
						return nil
					},
				},
				PostTerminates: []dockercontainer.ContainerHook{
					func(_ context.Context, _ *dockercontainer.Container) error {
						return nil
					},
				},
			}),
		})
	})
}

// BenchmarkRunContainerCleanup measures container cleanup time
func BenchmarkRunContainerCleanup(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	b.Run("minimal", func(b *testing.B) {
		benchmarkRunContainerCleanup(b, ctx, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithImage(nginxAlpineImage),
		})
	})

	b.Run("with-env", func(b *testing.B) {
		benchmarkRunContainerCleanup(b, ctx, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithEnv(map[string]string{
				"ENV1": "value1",
				"ENV2": "value2",
			}),
		})
	})

	b.Run("with-ports", func(b *testing.B) {
		benchmarkRunContainerCleanup(b, ctx, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithExposedPorts("80/tcp", "443/tcp"),
		})
	})
}

// benchmarkContainerRun is a helper function to benchmark container run
func benchmarkContainerRun(b *testing.B, ctx context.Context, opts []dockercontainer.ContainerCustomizer) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		ctr, err := dockercontainer.Run(ctx, opts...)
		dockercontainer.CleanupContainer(b, ctr)
		require.NoError(b, err)
	}
}

// benchmarkRunContainerCleanup is a helper function to benchmark container cleanup
func benchmarkRunContainerCleanup(b *testing.B, ctx context.Context, opts []dockercontainer.ContainerCustomizer) {
	b.Helper()
	b.ReportAllocs()

	b.ResetTimer()
	for range b.N {
		// Create and immediately terminate one container at a time
		ctr, err := dockercontainer.Run(ctx, opts...)
		require.NoError(b, err)

		err = dockercontainer.TerminateContainer(ctr, dockercontainer.TerminateTimeout(30*time.Second))
		require.NoError(b, err)
	}
	b.StopTimer()
}
