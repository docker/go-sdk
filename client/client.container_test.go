package client_test

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
)

func TestContainerList(t *testing.T) {
	dockerClient, err := client.New(context.Background())
	require.NoError(t, err)
	require.NotNil(t, dockerClient)

	img := "nginx:alpine"

	pullImage(t, dockerClient, img)

	max := 5

	wg := sync.WaitGroup{}
	wg.Add(max)

	for i := range max {
		go func(i int) {
			defer wg.Done()

			createContainer(t, dockerClient, img, fmt.Sprintf("nginx-test-name-%d", i))
		}(i)
	}

	wg.Wait()

	containers, err := dockerClient.ContainerList(context.Background(), dockerclient.ContainerListOptions{All: true, Filters: make(dockerclient.Filters).Add("label", client.LabelBase+".integration-test=TestingContainer")})
	require.NoError(t, err)
	require.NotEmpty(t, containers.Items)
	require.Len(t, containers.Items, max)
}

func TestFindContainerByName(t *testing.T) {
	dockerClient, err := client.New(context.Background())
	require.NoError(t, err)
	require.NotNil(t, dockerClient)

	createContainer(t, dockerClient, "nginx:alpine", "nginx-test-name")

	t.Run("found", func(t *testing.T) {
		found, err := dockerClient.FindContainerByName(context.Background(), "nginx-test-name")
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, "/nginx-test-name", found.Names[0])
		require.Equal(t, "nginx:alpine", found.Image)
	})

	t.Run("not-found", func(t *testing.T) {
		found, err := dockerClient.FindContainerByName(context.Background(), "nginx-test-name-not-found")
		require.ErrorIs(t, err, errdefs.ErrNotFound)
		require.Nil(t, found)
	})

	t.Run("empty-name", func(t *testing.T) {
		found, err := dockerClient.FindContainerByName(context.Background(), "")
		require.ErrorIs(t, err, errdefs.ErrInvalidArgument)
		require.Nil(t, found)
	})
}

func TestContainerPause(t *testing.T) {
	dockerClient, err := client.New(context.Background())
	require.NoError(t, err)
	require.NotNil(t, dockerClient)

	img := "nginx:alpine"

	pullImage(t, dockerClient, img)
	createContainer(t, dockerClient, img, "nginx-test-pause")

	_, err = dockerClient.ContainerStart(context.Background(), "nginx-test-pause", dockerclient.ContainerStartOptions{})
	require.NoError(t, err)

	_, err = dockerClient.ContainerPause(context.Background(), "nginx-test-pause", dockerclient.ContainerPauseOptions{})
	require.NoError(t, err)

	_, err = dockerClient.ContainerUnpause(context.Background(), "nginx-test-pause", dockerclient.ContainerUnpauseOptions{})
	require.NoError(t, err)
}

func createContainer(tb testing.TB, dockerClient client.SDKClient, img string, name string) {
	tb.Helper()

	resp, err := dockerClient.ContainerCreate(context.Background(), dockerclient.ContainerCreateOptions{
		Config: &container.Config{
			Image: img,
			ExposedPorts: network.PortSet{
				network.MustParsePort("80/tcp"): {},
			},
			Labels: map[string]string{client.LabelBase + ".integration-test": "TestingContainer"},
		},
		Name: name,
	})
	require.NoError(tb, err)
	require.NotNil(tb, resp)
	require.NotEmpty(tb, resp.ID)

	tb.Cleanup(func() {
		_, err := dockerClient.ContainerRemove(context.Background(), resp.ID, dockerclient.ContainerRemoveOptions{Force: true})
		require.NoError(tb, err)
	})
}

func pullImage(tb testing.TB, client client.SDKClient, img string) {
	tb.Helper()

	r, err := client.ImagePull(context.Background(), img, dockerclient.ImagePullOptions{})
	require.NoError(tb, err)
	defer r.Close()

	_, err = io.ReadAll(r)
	require.NoError(tb, err)
}

func TestContainerCreate_NilConfig(t *testing.T) {
	dockerClient, err := client.New(context.Background())
	require.NoError(t, err)
	require.NotNil(t, dockerClient)

	_, err = dockerClient.ContainerCreate(context.Background(), dockerclient.ContainerCreateOptions{
		Image: "nginx:alpine",
		Name:  "nil-config-test",
	})

	require.Error(t, err)
	require.True(t, errdefs.IsInvalidArgument(err))
	require.Equal(t, "config is nil", err.Error())
}
