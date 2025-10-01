package container_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/network"
)

func TestContainer_ContainerIPs(t *testing.T) {
	bufLogger := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(bufLogger, nil))

	dockerClient, err := client.New(context.Background(), client.WithLogger(logger))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	nw1, err := network.New(context.Background(), network.WithClient(dockerClient))
	network.Cleanup(t, nw1)
	require.NoError(t, err)

	nw2, err := network.New(context.Background(), network.WithClient(dockerClient))
	network.Cleanup(t, nw2)
	require.NoError(t, err)

	ctr, err := container.Run(
		context.Background(),
		container.WithImage(nginxAlpineImage),
		container.WithClient(dockerClient),
		container.WithNetwork([]string{"ctr1"}, nw1),
		container.WithNetwork([]string{"ctr2"}, nw2),
	)
	container.Cleanup(t, ctr)
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
			container.WithClient(dockerClient),
			container.WithNetwork([]string{"ctr3"}, nw1),
		)
		container.Cleanup(t, ctr3)
		require.NoError(t, err)

		ip, err := ctr3.ContainerIP(context.Background())
		require.NoError(t, err)
		require.NotEmpty(t, ip)
	})
}

func TestContainer_Networks(t *testing.T) {
	bufLogger := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(bufLogger, nil))

	dockerClient, err := client.New(context.Background(), client.WithLogger(logger))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	nw, err := network.New(context.Background(), network.WithClient(dockerClient))
	network.Cleanup(t, nw)
	require.NoError(t, err)

	ctr, err := container.Run(
		context.Background(),
		container.WithImage(nginxAlpineImage),
		container.WithClient(dockerClient),
		container.WithNetwork([]string{"ctr1-a", "ctr1-b", "ctr1-c"}, nw),
	)
	require.NoError(t, err)
	container.Cleanup(t, ctr)

	networks, err := ctr.Networks(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{nw.Name()}, networks)
}

func TestContainer_NetworkAliases(t *testing.T) {
	bufLogger := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(bufLogger, nil))

	dockerClient, err := client.New(context.Background(), client.WithLogger(logger))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	nw, err := network.New(context.Background(), network.WithClient(dockerClient))
	network.Cleanup(t, nw)
	require.NoError(t, err)

	ctr, err := container.Run(
		context.Background(),
		container.WithImage(nginxAlpineImage),
		container.WithClient(dockerClient),
		container.WithNetwork([]string{"ctr1-a", "ctr1-b", "ctr1-c"}, nw),
	)
	require.NoError(t, err)
	container.Cleanup(t, ctr)

	aliases, err := ctr.NetworkAliases(context.Background())
	require.NoError(t, err)
	require.Equal(t, map[string][]string{
		nw.Name(): {"ctr1-a", "ctr1-b", "ctr1-c"},
	}, aliases)
}
