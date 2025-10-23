package network_test

import (
	"context"
	"net/netip"
	"runtime"
	"testing"

	apinetwork "github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/network"
)

func newNetworkSuite(t *testing.T, dockerClient client.SDKClient) {
	t.Helper()

	t.Run("no-name", func(t *testing.T) {
		ctx := context.Background()

		driver := "bridge"
		if runtime.GOOS == "windows" {
			driver = "nat"
		}

		nw, err := network.New(ctx,
			network.WithClient(dockerClient),
			network.WithDriver(driver),
		)
		network.Cleanup(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
		require.NotEmpty(t, nw.Name())
		require.Equal(t, driver, nw.Driver())
	})

	t.Run("with-name", func(t *testing.T) {
		ctx := context.Background()

		nw, err := network.New(ctx,
			network.WithClient(dockerClient),
			network.WithName("test-network"),
		)
		network.Cleanup(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
		require.Equal(t, "test-network", nw.Name())
	})

	t.Run("with-empty-name", func(t *testing.T) {
		ctx := context.Background()

		nw, err := network.New(ctx,
			network.WithClient(dockerClient),
			network.WithName(""),
		)
		network.Cleanup(t, nw)
		require.Error(t, err)
		require.Nil(t, nw)
	})

	t.Run("with-ipam", func(t *testing.T) {
		ctx := context.Background()

		ipamConfig := apinetwork.IPAM{
			Driver: "default",
			Config: []apinetwork.IPAMConfig{
				{
					Subnet:  netip.MustParsePrefix("10.1.1.0/24"),
					Gateway: netip.MustParseAddr("10.1.1.254"),
				},
			},
			Options: map[string]string{
				"driver": "host-local",
			},
		}
		nw, err := network.New(ctx,
			network.WithClient(dockerClient),
			network.WithIPAM(&ipamConfig),
		)
		network.Cleanup(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
	})

	t.Run("with-attachable", func(t *testing.T) {
		ctx := context.Background()

		nw, err := network.New(ctx,
			network.WithClient(dockerClient),
			network.WithAttachable(),
		)
		network.Cleanup(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
	})

	t.Run("with-internal", func(t *testing.T) {
		ctx := context.Background()

		nw, err := network.New(ctx,
			network.WithClient(dockerClient),
			network.WithInternal(),
		)
		network.Cleanup(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
	})

	t.Run("with-enable-ipv6", func(t *testing.T) {
		ctx := context.Background()

		nw, err := network.New(ctx,
			network.WithClient(dockerClient),
			network.WithEnableIPv6(),
		)
		network.Cleanup(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
	})

	t.Run("with-labels", func(t *testing.T) {
		ctx := context.Background()

		nw, err := network.New(ctx,
			network.WithClient(dockerClient),
			network.WithLabels(map[string]string{"test": "test"}),
		)
		network.Cleanup(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)

		inspect, err := nw.Inspect(ctx)
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.Network.Labels, client.LabelBase)
		require.Contains(t, inspect.Network.Labels, client.LabelLang)
		require.Contains(t, inspect.Network.Labels, client.LabelVersion)
	})
}

func TestNew(t *testing.T) {
	t.Run("new-client", func(t *testing.T) {
		dockerClient, err := client.New(context.Background())
		require.NoError(t, err)
		defer dockerClient.Close()

		newNetworkSuite(t, dockerClient)
	})
}

func TestDuplicatedName(t *testing.T) {
	ctx := context.Background()

	nw, err := network.New(ctx,
		network.WithName("foo-network"),
	)
	network.Cleanup(t, nw)
	require.NoError(t, err)
	require.NotNil(t, nw)

	nw2, err := network.New(ctx,
		network.WithName("foo-network"),
	)
	require.Error(t, err)
	require.Nil(t, nw2)
}
