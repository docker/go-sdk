package network_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/network"
)

func TestTerminate(t *testing.T) {
	dockerClient, _ := testClientWithLogger(t)
	defer dockerClient.Close()

	t.Run("network-does-not-exist", func(t *testing.T) {
		n := &network.Network{}
		require.Error(t, n.Terminate(context.Background()))
	})

	t.Run("network-exist", func(t *testing.T) {
		nw, err := network.New(context.Background(),
			network.WithClient(dockerClient),
		)
		require.NoError(t, err)
		require.NoError(t, nw.Terminate(context.Background()))
	})
}
