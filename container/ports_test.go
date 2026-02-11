package container_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/containerd/errdefs"
	apicontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container"
)

func TestMappedPort(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Run("single-port", func(t *testing.T) {
			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithExposedPorts("80/tcp"),
			)
			container.Cleanup(t, ctr)
			require.NoError(t, err)

			t.Run("exposed-port", func(t *testing.T) {
				port, err := ctr.MappedPort(context.Background(), network.MustParsePort("80/tcp"))
				require.NoError(t, err)
				require.False(t, port.IsZero())
				require.Equal(t, "tcp", string(port.Proto()))
			})

			t.Run("port-without-proto", func(t *testing.T) {
				// Port without protocol should match any protocol
				port, err := ctr.MappedPort(context.Background(), network.MustParsePort("80"))
				require.NoError(t, err)
				require.False(t, port.IsZero())
			})
		})

		t.Run("multiple-ports", func(t *testing.T) {
			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithExposedPorts("80/tcp", "8080/tcp"),
			)
			container.Cleanup(t, ctr)
			require.NoError(t, err)

			port80, err := ctr.MappedPort(context.Background(), network.MustParsePort("80/tcp"))
			require.NoError(t, err)
			require.False(t, port80.IsZero())

			port8080, err := ctr.MappedPort(context.Background(), network.MustParsePort("8080/tcp"))
			require.NoError(t, err)
			require.False(t, port8080.IsZero())

			// Mapped ports should be different
			require.NotEqual(t, port80.Num(), port8080.Num())
		})

		t.Run("host-network-mode", func(t *testing.T) {
			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithHostConfigModifier(func(hostConfig *apicontainer.HostConfig) {
					hostConfig.NetworkMode = "host"
				}),
			)
			container.Cleanup(t, ctr)
			require.NoError(t, err)

			// In host mode, mapped port should equal the container port
			port, err := ctr.MappedPort(context.Background(), network.MustParsePort("80/tcp"))
			require.NoError(t, err)
			require.Equal(t, uint16(80), port.Num())
			require.Equal(t, "tcp", string(port.Proto()))
		})
	})

	t.Run("error", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
			container.WithExposedPorts("80/tcp"),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)

		t.Run("port-not-found", func(t *testing.T) {
			// Request a port that was not exposed
			port, err := ctr.MappedPort(context.Background(), network.MustParsePort("9999/tcp"))
			require.Error(t, err)
			require.True(t, errdefs.IsNotFound(err))
			require.True(t, port.IsZero())
		})

		t.Run("wrong-protocol", func(t *testing.T) {
			// Request TCP port with UDP protocol
			port, err := ctr.MappedPort(context.Background(), network.MustParsePort("80/udp"))
			require.Error(t, err)
			require.True(t, errdefs.IsNotFound(err))
			require.True(t, port.IsZero())
		})
	})
}

func TestEndpoint(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Run("single-port", func(t *testing.T) {
			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithExposedPorts("80/tcp"),
			)
			container.Cleanup(t, ctr)
			require.NoError(t, err)

			t.Run("with-proto", func(t *testing.T) {
				endpoint, err := ctr.Endpoint(context.Background(), "http")
				require.NoError(t, err)
				require.NotEmpty(t, endpoint)
				require.Contains(t, endpoint, "http://")
				require.Contains(t, endpoint, ":")
			})

			t.Run("without-proto", func(t *testing.T) {
				endpoint, err := ctr.Endpoint(context.Background(), "")
				require.NoError(t, err)
				require.NotEmpty(t, endpoint)
				require.NotContains(t, endpoint, "://")
				require.Contains(t, endpoint, ":")
			})
		})

		t.Run("lowest-port", func(t *testing.T) {
			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithExposedPorts("8080/tcp", "80/tcp", "9000/tcp"),
			)
			container.Cleanup(t, ctr)
			require.NoError(t, err)

			endpoint, err := ctr.Endpoint(context.Background(), "http")
			require.NoError(t, err)
			require.NotEmpty(t, endpoint)

			// Should use port 80 (lowest)
			port, err := ctr.MappedPort(context.Background(), network.MustParsePort("80/tcp"))
			require.NoError(t, err)
			require.Contains(t, endpoint, ":"+strconv.Itoa(int(port.Num())))
		})
	})

	t.Run("error", func(t *testing.T) {
		t.Run("no-ports-exposed", func(t *testing.T) {
			// Use alpine image which doesn't expose any ports by default
			ctr, err := container.Run(context.Background(),
				container.WithImage(alpineLatest),
				container.WithCmd("sleep", "300"),
			)
			container.Cleanup(t, ctr)
			require.NoError(t, err)

			endpoint, err := ctr.Endpoint(context.Background(), "http")
			require.Error(t, err)
			require.True(t, errdefs.IsNotFound(err))
			require.Empty(t, endpoint)
			require.Contains(t, err.Error(), "no ports exposed")
		})
	})
}

func TestPortEndpoint(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
			container.WithExposedPorts("80/tcp", "8080/tcp"),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)

		t.Run("with-proto", func(t *testing.T) {
			endpoint, err := ctr.PortEndpoint(context.Background(), network.MustParsePort("80/tcp"), "http")
			require.NoError(t, err)
			require.NotEmpty(t, endpoint)
			require.Contains(t, endpoint, "http://")
			require.Contains(t, endpoint, ":")
		})

		t.Run("without-proto", func(t *testing.T) {
			endpoint, err := ctr.PortEndpoint(context.Background(), network.MustParsePort("80/tcp"), "")
			require.NoError(t, err)
			require.NotEmpty(t, endpoint)
			require.NotContains(t, endpoint, "://")
			require.Contains(t, endpoint, ":")
		})

		t.Run("specific-port", func(t *testing.T) {
			// Get endpoint for port 8080 specifically
			endpoint8080, err := ctr.PortEndpoint(context.Background(), network.MustParsePort("8080/tcp"), "http")
			require.NoError(t, err)
			require.NotEmpty(t, endpoint8080)

			// Verify it's using the mapped port for 8080
			port8080, err := ctr.MappedPort(context.Background(), network.MustParsePort("8080/tcp"))
			require.NoError(t, err)
			require.Contains(t, endpoint8080, ":"+strconv.Itoa(int(port8080.Num())))
		})
	})

	t.Run("error", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
			container.WithExposedPorts("80/tcp"),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)

		t.Run("port-not-found", func(t *testing.T) {
			endpoint, err := ctr.PortEndpoint(context.Background(), network.MustParsePort("9999/tcp"), "http")
			require.Error(t, err)
			require.True(t, errdefs.IsNotFound(err))
			require.Empty(t, endpoint)
		})
	})
}
