package network_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/network"
)

func TestFindByID(t *testing.T) {
	nw, err := network.New(context.Background(), network.WithName("test-by-id"))
	network.Cleanup(t, nw)
	require.NoError(t, err)

	inspect, err := network.FindByID(context.Background(), nw.ID())
	require.NoError(t, err)
	require.Equal(t, nw.ID(), inspect.ID)

	no, err := network.FindByID(context.Background(), "not-found-id")
	require.Error(t, err)
	require.Empty(t, no.ID)
}

func TestFindByName(t *testing.T) {
	nw, err := network.New(context.Background(), network.WithName("test-by-name"))
	network.Cleanup(t, nw)
	require.NoError(t, err)

	inspect, err := network.FindByName(context.Background(), nw.Name())
	require.NoError(t, err)
	require.Equal(t, nw.Name(), inspect.Name)

	no, err := network.FindByName(context.Background(), "not-found-name")
	require.Error(t, err)
	require.Empty(t, no.Name)
}

func TestList(t *testing.T) {
	nws, err := network.List(context.Background())
	require.NoError(t, err)
	initialCount := len(nws)

	t.Run("no-filters", func(t *testing.T) {
		max := 5
		for range max {
			nw, err := network.New(context.Background())
			network.Cleanup(t, nw)
			require.NoError(t, err)
		}

		nws, err = network.List(context.Background())
		require.NoError(t, err)
		require.Len(t, nws, initialCount+max)
	})

	t.Run("with-filters", func(t *testing.T) {
		nws, err = network.List(context.Background(), network.WithFilters(filters.NewArgs(filters.Arg("driver", "bridge"))))
		require.NoError(t, err)
		require.Len(t, nws, 1)
	})

	t.Run("with-list-client", func(t *testing.T) {
		dockerClient, err := client.New(context.Background())
		require.NoError(t, err)

		nw, err := network.New(context.Background(), network.WithClient(dockerClient))
		network.Cleanup(t, nw)
		require.NoError(t, err)

		nws, err = network.List(context.Background(), network.WithListClient(dockerClient))
		require.NoError(t, err)
		require.Len(t, nws, initialCount+1)
	})
}
