package container_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/network"
)

func TestContainer_ContainerIPs(t *testing.T) {
	bufLogger := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(bufLogger, nil))

	dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	nw1, err := network.New(context.Background(), network.WithClient(dockerClient))
	require.NoError(t, err)
	network.CleanupNetwork(t, nw1)

	nw2, err := network.New(context.Background(), network.WithClient(dockerClient))
	require.NoError(t, err)
	network.CleanupNetwork(t, nw2)

	ctr, err := container.Run(
		context.Background(),
		container.WithImage(nginxAlpineImage),
		container.WithDockerClient(dockerClient),
		container.WithNetwork([]string{"ctr1"}, nw1),
		container.WithNetwork([]string{"ctr2"}, nw2),
	)
	container.CleanupContainer(t, ctr)
	require.NoError(t, err)

	t.Run("container-ips", func(t *testing.T) {
		ips, err := ctr.ContainerIPs(context.Background())
		require.NoError(t, err)
		require.Len(t, ips, 2)
	})

	t.Run("container-ip/multiple-networks/empty", func(t *testing.T) {
		ip, err := ctr.ContainerIP(context.Background())
		require.NoError(t, err)
		require.Empty(t, ip)
	})

	t.Run("container-ip/one-network", func(t *testing.T) {
		ctr3, err := container.Run(
			context.Background(),
			container.WithImage(nginxAlpineImage),
			container.WithDockerClient(dockerClient),
			container.WithNetwork([]string{"ctr3"}, nw1),
		)
		container.CleanupContainer(t, ctr3)
		require.NoError(t, err)

		ip, err := ctr3.ContainerIP(context.Background())
		require.NoError(t, err)
		require.NotEmpty(t, ip)
	})
}

func TestContainer_Networks(t *testing.T) {
	bufLogger := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(bufLogger, nil))

	dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	nw, err := network.New(context.Background(), network.WithClient(dockerClient))
	network.CleanupNetwork(t, nw)
	require.NoError(t, err)

	ctr, err := container.Run(
		context.Background(),
		container.WithImage(nginxAlpineImage),
		container.WithDockerClient(dockerClient),
		container.WithNetwork([]string{"ctr1-a", "ctr1-b", "ctr1-c"}, nw),
	)
	require.NoError(t, err)
	container.CleanupContainer(t, ctr)

	networks, err := ctr.Networks(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{nw.Name()}, networks)
}

func TestContainer_NetworkAliases(t *testing.T) {
	bufLogger := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(bufLogger, nil))

	dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	nw, err := network.New(context.Background(), network.WithClient(dockerClient))
	network.CleanupNetwork(t, nw)
	require.NoError(t, err)

	ctr, err := container.Run(
		context.Background(),
		container.WithImage(nginxAlpineImage),
		container.WithDockerClient(dockerClient),
		container.WithNetwork([]string{"ctr1-a", "ctr1-b", "ctr1-c"}, nw),
	)
	require.NoError(t, err)
	container.CleanupContainer(t, ctr)

	aliases, err := ctr.NetworkAliases(context.Background())
	require.NoError(t, err)
	require.Equal(t, map[string][]string{
		nw.Name(): {"ctr1-a", "ctr1-b", "ctr1-c"},
	}, aliases)
}
