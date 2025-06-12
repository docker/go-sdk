package dockernetwork_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockernetwork"
)

func TestNew(t *testing.T) {
	t.Run("no-name", func(t *testing.T) {
		ctx := context.Background()

		driver := "bridge"
		if runtime.GOOS == "windows" {
			driver = "nat"
		}

		nw, err := dockernetwork.New(ctx,
			dockernetwork.WithDriver(driver),
		)
		dockernetwork.CleanupNetwork(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
		require.NotEmpty(t, nw.Name())
		require.Equal(t, driver, nw.Driver())
	})

	t.Run("with-name", func(t *testing.T) {
		ctx := context.Background()

		nw, err := dockernetwork.New(ctx,
			dockernetwork.WithName("test-network"),
		)
		dockernetwork.CleanupNetwork(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
		require.Equal(t, "test-network", nw.Name())
	})

	t.Run("with-empty-name", func(t *testing.T) {
		ctx := context.Background()

		nw, err := dockernetwork.New(ctx,
			dockernetwork.WithName(""),
		)
		dockernetwork.CleanupNetwork(t, nw)
		require.Error(t, err)
		require.Nil(t, nw)
	})

	t.Run("with-ipam", func(t *testing.T) {
		ctx := context.Background()

		ipamConfig := network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet:  "10.1.1.0/24",
					Gateway: "10.1.1.254",
				},
			},
			Options: map[string]string{
				"driver": "host-local",
			},
		}
		nw, err := dockernetwork.New(ctx,
			dockernetwork.WithIPAM(&ipamConfig),
		)
		dockernetwork.CleanupNetwork(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
	})

	t.Run("with-attachable", func(t *testing.T) {
		ctx := context.Background()

		nw, err := dockernetwork.New(ctx,
			dockernetwork.WithAttachable(),
		)
		dockernetwork.CleanupNetwork(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
	})

	t.Run("with-internal", func(t *testing.T) {
		ctx := context.Background()

		nw, err := dockernetwork.New(ctx,
			dockernetwork.WithInternal(),
		)
		dockernetwork.CleanupNetwork(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
	})

	t.Run("with-enable-ipv6", func(t *testing.T) {
		ctx := context.Background()

		nw, err := dockernetwork.New(ctx,
			dockernetwork.WithEnableIPv6(),
		)
		dockernetwork.CleanupNetwork(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)
	})

	t.Run("with-labels", func(t *testing.T) {
		ctx := context.Background()

		nw, err := dockernetwork.New(ctx,
			dockernetwork.WithLabels(map[string]string{"test": "test"}),
		)
		dockernetwork.CleanupNetwork(t, nw)
		require.NoError(t, err)
		require.NotNil(t, nw)

		inspect, err := nw.Inspect(ctx)
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.Labels, dockerclient.LabelBase)
		require.Contains(t, inspect.Labels, dockerclient.LabelLang)
		require.Contains(t, inspect.Labels, dockerclient.LabelVersion)
	})
}

func TestDuplicatedName(t *testing.T) {
	ctx := context.Background()

	nw, err := dockernetwork.New(ctx,
		dockernetwork.WithName("foo-network"),
	)
	dockernetwork.CleanupNetwork(t, nw)
	require.NoError(t, err)
	require.NotNil(t, nw)

	nw2, err := dockernetwork.New(ctx,
		dockernetwork.WithName("foo-network"),
	)
	require.Error(t, err)
	require.Nil(t, nw2)
}
