package network_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apinetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/network"
)

func TestInspect(t *testing.T) {
	t.Run("network-exists", func(t *testing.T) {
		nw, err := network.New(context.Background())
		network.Cleanup(t, nw)
		require.NoError(t, err)

		inspect, err := nw.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.Labels, client.LabelBase)
		require.Contains(t, inspect.Labels, client.LabelLang)
		require.Contains(t, inspect.Labels, client.LabelVersion)
		require.Contains(t, inspect.Labels, client.LabelBase+".network")
		require.Equal(t, network.Version(), inspect.Labels[client.LabelBase+".network"])
	})

	t.Run("network-does-not-exist", func(t *testing.T) {
		n := &network.Network{}

		inspect, err := n.Inspect(context.Background())
		require.Error(t, err)
		require.Zero(t, inspect)
	})

	t.Run("with-options", func(t *testing.T) {
		t.Run("option-error", func(t *testing.T) {
			dockerClient, _ := testClientWithLogger(t)
			defer dockerClient.Close()

			nw, err := network.New(context.Background(), network.WithName("test-network-option-error"), network.WithClient(dockerClient))
			network.Cleanup(t, nw)
			require.NoError(t, err)

			// Create an invalid inspect option that will cause an error
			invalidOption := network.WithInspectOptions(apinetwork.InspectOptions{
				Scope: "invalid-scope", // Using an invalid scope value
			})

			inspect, err := nw.Inspect(context.Background(), invalidOption)
			require.Error(t, err)
			require.Zero(t, inspect)
		})

		t.Run("no-cache", func(t *testing.T) {
			dockerClient, buf := testClientWithLogger(t)
			defer dockerClient.Close()

			nw, err := network.New(context.Background(), network.WithName("test-network-no-cache"), network.WithClient(dockerClient))
			network.Cleanup(t, nw)
			require.NoError(t, err)

			inspect, err := nw.Inspect(context.Background(), network.WithNoCache())
			require.NoError(t, err)
			require.NotNil(t, inspect)

			// check that the logger was not called
			require.NotContains(t, buf.String(), "network not inspected yet, inspecting now")
		})

		t.Run("with-cache", func(t *testing.T) {
			dockerClient, buf := testClientWithLogger(t)
			defer dockerClient.Close()

			nw, err := network.New(context.Background(), network.WithName("test-network-with-cache"), network.WithClient(dockerClient))
			network.Cleanup(t, nw)
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

func testClientWithLogger(t *testing.T) (*client.Client, *bytes.Buffer) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	// use a custom client with a custom logger
	dockerClient, err := client.New(context.Background(), client.WithLogger(logger))
	require.NoError(t, err)

	return dockerClient, buf
}
