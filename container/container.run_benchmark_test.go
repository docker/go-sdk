package container_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/container/exec"
)

// BenchmarkRun measures container creation time
func BenchmarkRun(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	b.Run("minimal", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
		})
	})

	b.Run("with-env", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
			container.WithEnv(map[string]string{
				"ENV1": "value1",
				"ENV2": "value2",
			}),
		})
	})

	b.Run("with-ports", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
			container.WithExposedPorts("80/tcp", "443/tcp"),
		})
	})

	b.Run("with-lifecycle-hooks", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
			container.WithLifecycleHooks(container.LifecycleHooks{
				PreCreates: []container.DefinitionHook{
					func(_ context.Context, _ *container.Definition) error {
						return nil
					},
				},
				PostCreates: []container.ContainerHook{
					func(_ context.Context, _ container.ContainerInfo) error {
						return nil
					},
				},
				PreStarts: []container.ContainerHook{
					func(_ context.Context, _ container.ContainerInfo) error {
						return nil
					},
				},
				PostStarts: []container.ContainerHook{
					func(_ context.Context, _ container.ContainerInfo) error {
						return nil
					},
				},
				PostReadies: []container.ContainerHook{
					func(_ context.Context, _ container.ContainerInfo) error {
						return nil
					},
				},
				PreStops: []container.ContainerHook{
					func(_ context.Context, _ container.ContainerInfo) error {
						return nil
					},
				},
				PostStops: []container.ContainerHook{
					func(_ context.Context, _ container.ContainerInfo) error {
						return nil
					},
				},
				PreTerminates: []container.ContainerHook{
					func(_ context.Context, _ container.ContainerInfo) error {
						return nil
					},
				},
				PostTerminates: []container.ContainerHook{
					func(_ context.Context, _ container.ContainerInfo) error {
						return nil
					},
				},
			}),
		})
	})

	b.Run("with-durable-startup-command", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
			container.WithDurableStartupCommand(
				exec.NewRawCommand([]string{"true"}),
			),
		})
	})

	b.Run("with-durable-startup-command-from-dir", func(b *testing.B) {
		benchmarkContainerRun(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
			container.WithDurableStartupCommandsFromDir("pg",
				exec.NewRawCommand([]string{"true"}),
			),
		})
	})
}

// BenchmarkRunCleanup measures container cleanup time
func BenchmarkRunCleanup(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	b.Run("minimal", func(b *testing.B) {
		benchmarkRunCleanup(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
		})
	})

	b.Run("with-env", func(b *testing.B) {
		benchmarkRunCleanup(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
			container.WithEnv(map[string]string{
				"ENV1": "value1",
				"ENV2": "value2",
			}),
		})
	})

	b.Run("with-ports", func(b *testing.B) {
		benchmarkRunCleanup(b, ctx, []container.ContainerCustomizer{
			container.WithImage(nginxAlpineImage),
			container.WithExposedPorts("80/tcp", "443/tcp"),
		})
	})
}

// benchmarkContainerRun is a helper function to benchmark container run
func benchmarkContainerRun(b *testing.B, ctx context.Context, opts []container.ContainerCustomizer) {
	b.Helper()
	b.ReportAllocs()

	for range b.N {
		ctr, err := container.Run(ctx, opts...)
		container.Cleanup(b, ctr)
		require.NoError(b, err)
	}
}

// benchmarkRunCleanup is a helper function to benchmark container cleanup
func benchmarkRunCleanup(b *testing.B, ctx context.Context, opts []container.ContainerCustomizer) {
	b.Helper()
	b.ReportAllocs()

	b.ResetTimer()
	for range b.N {
		// Create and immediately terminate one container at a time
		ctr, err := container.Run(ctx, opts...)
		require.NoError(b, err)

		err = container.Terminate(ctr, container.TerminateTimeout(30*time.Second))
		require.NoError(b, err)
	}
	b.StopTimer()
}
