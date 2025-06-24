package client_test

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
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

	containers, err := dockerClient.ContainerList(context.Background(), container.ListOptions{All: true})
	require.NoError(t, err)
	require.NotEmpty(t, containers)
	require.Len(t, containers, max)
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

	err = dockerClient.ContainerStart(context.Background(), "nginx-test-pause", container.StartOptions{})
	require.NoError(t, err)

	err = dockerClient.ContainerPause(context.Background(), "nginx-test-pause")
	require.NoError(t, err)

	err = dockerClient.ContainerUnpause(context.Background(), "nginx-test-pause")
	require.NoError(t, err)
}

func createContainer(tb testing.TB, dockerClient *client.Client, img string, name string) {
	tb.Helper()

	resp, err := dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: img,
		ExposedPorts: nat.PortSet{
			"80/tcp": {},
		},
	}, nil, nil, nil, name)
	require.NoError(tb, err)
	require.NotNil(tb, resp)
	require.NotEmpty(tb, resp.ID)

	tb.Cleanup(func() {
		err := dockerClient.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
		require.NoError(tb, err)
	})
}

func pullImage(tb testing.TB, client *client.Client, img string) {
	tb.Helper()

	r, err := client.ImagePull(context.Background(), img, image.PullOptions{})
	require.NoError(tb, err)
	defer r.Close()

	_, err = io.ReadAll(r)
	require.NoError(tb, err)
}
