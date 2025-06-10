package dockercontainer_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockercontainer"
)

func TestRunContainer(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		t.Run("no-image", func(t *testing.T) {
			ctr, err := dockercontainer.Run(context.Background())
			require.Error(t, err)
			require.Nil(t, ctr)
		})

		t.Run("invalid-ports", func(t *testing.T) {
			ctr, err := dockercontainer.Run(context.Background(),
				dockercontainer.WithExposedPorts("invalid-port"),
			)
			require.Error(t, err)
			require.Nil(t, ctr)
		})

		t.Run("invalid-with-image-platform", func(t *testing.T) {
			ctr, err := dockercontainer.Run(context.Background(),
				dockercontainer.WithImage(nginxAlpineImage),
				dockercontainer.WithImagePlatform("invalid"),
			)
			dockercontainer.CleanupContainer(t, ctr)
			require.Error(t, err)
			require.Nil(t, ctr)
		})
	})

	t.Run("with-image", func(t *testing.T) {
		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("with-dockerclient", func(t *testing.T) {
		// Initialize the docker client. It will be closed when the container is terminated,
		// so no need to close it during the entire container lifecycle.
		dockerClient, err := dockerclient.New(context.Background())
		require.NoError(t, err)

		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithDockerClient(dockerClient),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithExposedPorts("80/tcp"),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("no-dockerclient-uses-default", func(t *testing.T) {
		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("api-methods", func(t *testing.T) {
		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithImagePlatform("linux/amd64"),
			dockercontainer.WithAlwaysPull(),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		t.Run("host", func(t *testing.T) {
			host, err := ctr.Host(context.Background())
			require.NoError(t, err)
			require.NotEmpty(t, host)
		})

		t.Run("inspect", func(t *testing.T) {
			inspect, err := ctr.Inspect(context.Background())
			require.NoError(t, err)
			require.NotNil(t, inspect)

			require.Equal(t, ctr.ID, inspect.ID)
			require.Equal(t, ctr.Image, inspect.Config.Image)
		})

		t.Run("mapped-ports", func(t *testing.T) {
			port1, err := ctr.MappedPort(context.Background(), "80/tcp")
			require.NoError(t, err)
			require.NotNil(t, port1)
		})
	})
}

func TestRunContainer_addSDKLabels(t *testing.T) {
	dockerClient, err := dockerclient.New(context.Background())
	require.NoError(t, err)

	ctr, err := dockercontainer.Run(context.Background(),
		dockercontainer.WithDockerClient(dockerClient),
		dockercontainer.WithImage(nginxAlpineImage),
	)
	dockercontainer.CleanupContainer(t, ctr)
	require.NoError(t, err)
	require.NotNil(t, ctr)

	inspect, err := ctr.Inspect(context.Background())
	require.NoError(t, err)

	require.Contains(t, inspect.Config.Labels, dockercontainer.LabelBase)
	require.Contains(t, inspect.Config.Labels, dockercontainer.LabelLang)
	require.Contains(t, inspect.Config.Labels, dockercontainer.LabelVersion)
}

func TestRunContainerWithLifecycleHooks(t *testing.T) {
	testRun := func(t *testing.T, start bool) {
		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
		require.NoError(t, err)

		opts := []dockercontainer.ContainerCustomizer{
			dockercontainer.WithDockerClient(dockerClient),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithLifecycleHooks(
				dockercontainer.LifecycleHooks{
					PreCreates: []dockercontainer.DefinitionHook{
						func(_ context.Context, def *dockercontainer.Definition) error {
							def.DockerClient.Logger().Info("pre-create hook")
							return nil
						},
					},
					PostCreates: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-create hook")
							return nil
						},
					},
					PreStarts: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("pre-start hook")
							return nil
						},
					},
					PostStarts: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-start hook")
							return nil
						},
					},
					PostReadies: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-ready hook")
							return nil
						},
					},
					PreStops: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("pre-stop hook")
							return nil
						},
					},
					PostStops: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-stop hook")
							return nil
						},
					},
					PreTerminates: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("pre-terminate hook")
							return nil
						},
					},
					PostTerminates: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-terminate hook")
							return nil
						},
					},
				},
			),
		}

		if !start {
			opts = append(opts, dockercontainer.WithNoStart())
		}

		ctr, err := dockercontainer.Run(context.Background(), opts...)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		// because the container is not started, the pre-start hook, and beyond hooks, should not be called
		require.Contains(t, bufLogger.String(), "pre-create hook")
		require.Contains(t, bufLogger.String(), "post-create hook")

		if start {
			require.Contains(t, bufLogger.String(), "pre-start hook")
			require.Contains(t, bufLogger.String(), "post-start hook")
			require.Contains(t, bufLogger.String(), "post-ready hook")

			// force the container lifecycle methods to be called
			err = ctr.Stop(context.Background(), nil)
			require.NoError(t, err)
			require.Contains(t, bufLogger.String(), "pre-stop hook")
			require.Contains(t, bufLogger.String(), "post-stop hook")

			err = ctr.Terminate(context.Background())
			require.NoError(t, err)
			require.Contains(t, bufLogger.String(), "pre-terminate hook")
			require.Contains(t, bufLogger.String(), "post-terminate hook")
		}
	}

	t.Run("create-container", func(t *testing.T) {
		testRun(t, false)
	})

	t.Run("run-container", func(t *testing.T) {
		testRun(t, true)
	})
}

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

	for i := 0; i < b.N; i++ {
		ctr, err := dockercontainer.Run(ctx, opts...)
		dockercontainer.CleanupContainer(b, ctr)
		require.NoError(b, err)
	}
}

// benchmarkRunContainerCleanup is a helper function to benchmark container cleanup
func benchmarkRunContainerCleanup(b *testing.B, ctx context.Context, opts []dockercontainer.ContainerCustomizer) {
	b.Helper()
	b.ReportAllocs()

	// Run containers first
	containers := make([]*dockercontainer.Container, b.N)
	for i := 0; i < b.N; i++ {
		ctr, err := dockercontainer.Run(ctx, opts...)
		require.NoError(b, err)
		containers[i] = ctr
	}

	// Now benchmark cleanup
	b.ResetTimer()
	var cleanupErr error
	for i := 0; i < b.N; i++ {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		cleanupErr = containers[i].Terminate(cleanupCtx)
		cleanupCancel()
	}
	b.StopTimer()

	require.NoError(b, cleanupErr)
}
