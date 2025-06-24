package client_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/client"
)

func TestContainerList(t *testing.T) {
	dockerClient, err := client.New(context.Background())
	require.NoError(t, err)
	require.NotNil(t, dockerClient)

	max := 5

	wg := sync.WaitGroup{}
	wg.Add(max)

	for i := range max {
		go func(i int) {
			defer wg.Done()

			resp, err := dockerClient.ContainerCreate(context.Background(), &container.Config{
				Image: "nginx:alpine",
				ExposedPorts: nat.PortSet{
					"80/tcp": {},
				},
			}, nil, nil, nil, fmt.Sprintf("nginx-test-name-%d", i))
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotEmpty(t, resp.ID)

			t.Cleanup(func() {
				err := dockerClient.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{})
				require.NoError(t, err)
			})
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

	resp, err := dockerClient.ContainerCreate(context.Background(), &container.Config{
		Image: "nginx:alpine",
		ExposedPorts: nat.PortSet{
			"80/tcp": {},
		},
	}, nil, nil, nil, "nginx-test-name")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.ID)

	t.Cleanup(func() {
		err := dockerClient.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{})
		require.NoError(t, err)
	})

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
