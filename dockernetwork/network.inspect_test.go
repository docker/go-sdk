package dockernetwork_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockernetwork"
)

func TestInspect(t *testing.T) {
	t.Run("network-exists", func(t *testing.T) {
		nw, err := dockernetwork.New(context.Background())
		dockernetwork.CleanupNetwork(t, nw)
		require.NoError(t, err)

		inspect, err := nw.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)
	})

	t.Run("network-does-not-exist", func(t *testing.T) {
		n := &dockernetwork.Network{}

		inspect, err := n.Inspect(context.Background())
		require.Error(t, err)
		require.Zero(t, inspect)
	})

	t.Run("with-options", func(t *testing.T) {
		t.Run("option-error", func(t *testing.T) {
			dockerClient, _ := testClientWithLogger(t)
			defer dockerClient.Close()

			nw, err := dockernetwork.New(context.Background(), dockernetwork.WithName("test-network-option-error"), dockernetwork.WithClient(dockerClient))
			dockernetwork.CleanupNetwork(t, nw)
			require.NoError(t, err)

			// Create an invalid inspect option that will cause an error
			invalidOption := dockernetwork.WithInspectOptions(network.InspectOptions{
				Scope: "invalid-scope", // Using an invalid scope value
			})

			inspect, err := nw.Inspect(context.Background(), invalidOption)
			require.Error(t, err)
			require.Zero(t, inspect)
		})

		t.Run("no-cache", func(t *testing.T) {
			dockerClient, buf := testClientWithLogger(t)
			defer dockerClient.Close()

			nw, err := dockernetwork.New(context.Background(), dockernetwork.WithName("test-network-no-cache"), dockernetwork.WithClient(dockerClient))
			dockernetwork.CleanupNetwork(t, nw)
			require.NoError(t, err)

			inspect, err := nw.Inspect(context.Background(), dockernetwork.WithNoCache())
			require.NoError(t, err)
			require.NotNil(t, inspect)

			// check that the logger was not called
			require.NotContains(t, buf.String(), "network not inspected yet, inspecting now")
		})

		t.Run("with-cache", func(t *testing.T) {
			dockerClient, buf := testClientWithLogger(t)
			defer dockerClient.Close()

			nw, err := dockernetwork.New(context.Background(), dockernetwork.WithName("test-network-with-cache"), dockernetwork.WithClient(dockerClient))
			dockernetwork.CleanupNetwork(t, nw)
			require.NoError(t, err)

			// first time inspecting the network: it will be cached
			inspect, err := nw.Inspect(context.Background())
			require.NoError(t, err)
			require.NotNil(t, inspect)

			expectedLog := "network not inspected yet, inspecting now"

			// check that the logger was called
			require.Contains(t, buf.String(), expectedLog)

			inspect, err = nw.Inspect(context.Background())
			require.NoError(t, err)
			require.NotNil(t, inspect)

			// check that the logger was called just once
			require.Equal(t, 1, strings.Count(buf.String(), expectedLog))
		})
	})
}

func testClientWithLogger(t *testing.T) (*dockerclient.Client, *bytes.Buffer) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	// use a custom client with a custom logger
	dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
	require.NoError(t, err)

	return dockerClient, buf
}
