package dockercontainer_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockercontainer"
)

func TestCreateContainer(t *testing.T) {
	// Initialize the docker client. It will be closed when the container is terminated,
	// so no need to close it during the entire container lifecycle.
	dockerClient, err := dockerclient.New(context.Background())
	require.NoError(t, err)

	ctr, err := dockercontainer.Create(context.Background(),
		dockercontainer.WithImage("nginx:alpine"),
		dockercontainer.WithDockerClient(dockerClient),
	)
	dockercontainer.CleanupContainer(t, ctr)
	require.NoError(t, err)
	require.NotNil(t, ctr)
}

func TestCreateContainer_addSDKLabels(t *testing.T) {
	dockerClient, err := dockerclient.New(context.Background())
	require.NoError(t, err)

	ctr, err := dockercontainer.Create(context.Background(),
		dockercontainer.WithDockerClient(dockerClient),
		dockercontainer.WithImage("nginx:alpine"),
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

func TestCreateContainerWithLifecycleHooks(t *testing.T) {
	bufLogger := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(bufLogger, nil))

	dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
	require.NoError(t, err)

	ctr, err := dockercontainer.Create(context.Background(),
		dockercontainer.WithDockerClient(dockerClient),
		dockercontainer.WithImage("nginx:alpine"),
		dockercontainer.WithLifecycleHooks(
			dockercontainer.LifecycleHooks{
				PreCreates: []dockercontainer.DefinitionHook{
					func(ctx context.Context, def *dockercontainer.Definition) error {
						def.DockerClient.Logger().Info("pre-create hook")
						return nil
					},
				},
				PostCreates: []dockercontainer.ContainerHook{
					func(ctx context.Context, ctr *dockercontainer.Container) error {
						ctr.Logger().Info("post-create hook")
						return nil
					},
				},
				PreStarts: []dockercontainer.ContainerHook{
					func(ctx context.Context, ctr *dockercontainer.Container) error {
						ctr.Logger().Info("pre-start hook")
						return nil
					},
				},
				PostStarts: []dockercontainer.ContainerHook{
					func(ctx context.Context, ctr *dockercontainer.Container) error {
						ctr.Logger().Info("post-start hook")
						return nil
					},
				},
				PostReadies: []dockercontainer.ContainerHook{
					func(ctx context.Context, ctr *dockercontainer.Container) error {
						ctr.Logger().Info("post-ready hook")
						return nil
					},
				},
				PreStops: []dockercontainer.ContainerHook{
					func(ctx context.Context, ctr *dockercontainer.Container) error {
						ctr.Logger().Info("pre-stop hook")
						return nil
					},
				},
				PostStops: []dockercontainer.ContainerHook{
					func(ctx context.Context, ctr *dockercontainer.Container) error {
						ctr.Logger().Info("post-stop hook")
						return nil
					},
				},
				PreTerminates: []dockercontainer.ContainerHook{
					func(ctx context.Context, ctr *dockercontainer.Container) error {
						ctr.Logger().Info("pre-terminate hook")
						return nil
					},
				},
				PostTerminates: []dockercontainer.ContainerHook{
					func(ctx context.Context, ctr *dockercontainer.Container) error {
						ctr.Logger().Info("post-terminate hook")
						return nil
					},
				},
			},
		),
	)
	dockercontainer.CleanupContainer(t, ctr)
	require.NoError(t, err)
	require.NotNil(t, ctr)

	// because the container is not started, the pre-start hook, and beyond hooks, should not be called
	require.Contains(t, bufLogger.String(), "pre-create hook")
	require.Contains(t, bufLogger.String(), "post-create hook")
	require.NotContains(t, bufLogger.String(), "pre-start hook")
	require.NotContains(t, bufLogger.String(), "post-start hook")
	require.NotContains(t, bufLogger.String(), "post-ready hook")
	require.NotContains(t, bufLogger.String(), "pre-stop hook")
	require.NotContains(t, bufLogger.String(), "post-stop hook")
	require.NotContains(t, bufLogger.String(), "pre-terminate hook")
	require.NotContains(t, bufLogger.String(), "post-terminate hook")
}
