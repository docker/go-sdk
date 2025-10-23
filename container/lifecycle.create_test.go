package container

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/netip"
	"regexp"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
)

const nginxAlpineImage = "nginx:alpine"

func TestMergePortBindings(t *testing.T) {
	type arg struct {
		configPortMap network.PortMap
		parsedPortMap network.PortMap
		exposedPorts  []string
	}
	cases := []struct {
		name     string
		arg      arg
		expected network.PortMap
	}{
		{
			name: "empty ports",
			arg: arg{
				configPortMap: nil,
				parsedPortMap: nil,
				exposedPorts:  nil,
			},
			expected: map[network.Port][]network.PortBinding{},
		},
		{
			name: "config port map but not exposed",
			arg: arg{
				configPortMap: map[network.Port][]network.PortBinding{
					network.MustParsePort("80/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "2"}},
				},
				parsedPortMap: nil,
				exposedPorts:  nil,
			},
			expected: map[network.Port][]network.PortBinding{},
		},
		{
			name: "parsed port map without config",
			arg: arg{
				configPortMap: nil,
				parsedPortMap: map[network.Port][]network.PortBinding{
					network.MustParsePort("80/tcp"): {{}},
				},
				exposedPorts: nil,
			},
			expected: map[network.Port][]network.PortBinding{
				network.MustParsePort("80/tcp"): {
					{HostPort: "0"},
				},
			},
		},
		{
			name: "parsed and configured but not exposed",
			arg: arg{
				configPortMap: map[network.Port][]network.PortBinding{
					network.MustParsePort("80/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "2"}},
				},
				parsedPortMap: map[network.Port][]network.PortBinding{
					network.MustParsePort("80/tcp"): {{}},
				},
				exposedPorts: nil,
			},
			expected: map[network.Port][]network.PortBinding{
				network.MustParsePort("80/tcp"): {
					{HostPort: "0"},
				},
			},
		},
		{
			name: "merge both parsed and config",
			arg: arg{
				configPortMap: map[network.Port][]network.PortBinding{
					network.MustParsePort("60/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "2"}},
					network.MustParsePort("70/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "2"}},
					network.MustParsePort("80/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "2"}},
				},
				parsedPortMap: map[network.Port][]network.PortBinding{
					network.MustParsePort("80/tcp"): {{}},
					network.MustParsePort("90/tcp"): {{}},
				},
				exposedPorts: []string{"70", "80/tcp"},
			},
			expected: map[network.Port][]network.PortBinding{
				network.MustParsePort("70/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "2"}},
				network.MustParsePort("80/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "2"}},
				network.MustParsePort("90/tcp"): {{HostPort: "0"}},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := mergePortBindings(c.arg.configPortMap, c.arg.parsedPortMap, c.arg.exposedPorts)
			require.Equal(t, c.expected, res)
		})
	}
}

func TestPreCreateModifierHook(t *testing.T) {
	ctx := context.Background()

	dockerClient, err := client.New(context.TODO())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	t.Run("no-exposed-ports", func(t *testing.T) {
		def := &Definition{
			image: nginxAlpineImage, // alpine image does expose port 80
			configModifier: func(config *container.Config) {
				config.Env = []string{"a=b"}
			},
			hostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = network.PortMap{
					network.MustParsePort("80/tcp"): []network.PortBinding{
						{
							HostIP:   netip.MustParseAddr("127.0.0.1"),
							HostPort: "2",
						},
					},
				}
			},
			endpointSettingsModifier: func(endpointSettings map[string]*network.EndpointSettings) {
				endpointSettings["a"] = &network.EndpointSettings{
					Aliases: []string{"b"},
					Links:   []string{"link1", "link2"},
				}
			},
		}

		// define empty inputs to be overwritten by the pre create hook
		inputConfig := &container.Config{
			Image: def.image,
		}
		inputHostConfig := &container.HostConfig{}
		inputNetworkingConfig := &network.NetworkingConfig{}

		err := preCreateContainerHook(ctx, dockerClient, def, inputConfig, inputHostConfig, inputNetworkingConfig)
		require.NoError(t, err)

		// assertions

		require.True(t, inputHostConfig.PublishAllPorts)
		require.Equal(
			t,
			[]string{"a=b"},
			inputConfig.Env,
			"Docker config's env should be overwritten by the modifier",
		)
		require.Equal(
			t,
			[]string{"b"},
			inputNetworkingConfig.EndpointsConfig["a"].Aliases,
			"Networking config's aliases should be overwritten by the modifier",
		)
		require.Equal(
			t,
			[]string{"link1", "link2"},
			inputNetworkingConfig.EndpointsConfig["a"].Links,
			"Networking config's links should be overwritten by the modifier",
		)
	})

	t.Run("no-exposed-ports-and-network-mode-is-container", func(t *testing.T) {
		def := &Definition{
			image: nginxAlpineImage, // alpine image does expose port 80
			hostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = network.PortMap{
					network.MustParsePort("80/tcp"): []network.PortBinding{
						{
							HostIP:   netip.MustParseAddr("127.0.0.1"),
							HostPort: "2",
						},
					},
				}
				hostConfig.NetworkMode = "container:foo"
			},
		}

		// define empty inputs to be overwritten by the pre create hook
		inputConfig := &container.Config{
			Image: def.image,
		}
		inputHostConfig := &container.HostConfig{}
		inputNetworkingConfig := &network.NetworkingConfig{}

		err := preCreateContainerHook(ctx, dockerClient, def, inputConfig, inputHostConfig, inputNetworkingConfig)
		require.NoError(t, err)

		// assertions

		require.Equal(
			t,
			network.PortSet(network.PortSet{}),
			inputConfig.ExposedPorts,
			"Docker config's exposed ports should be empty",
		)
		require.Equal(t,
			network.PortMap{},
			inputHostConfig.PortBindings,
			"Host config's portBinding should be empty",
		)
	})

	t.Run("definition-contains-more-than-one-network-including-aliases", func(t *testing.T) {
		networkName := "foo" + t.Name()
		nw := testCreateNetwork(t, networkName)

		def := &Definition{
			image:    nginxAlpineImage, // alpine image does expose port 80
			networks: []string{networkName, "bar"},
			networkAliases: map[string][]string{
				networkName: {"foo1"}, // network aliases are needed at the moment there is a network
			},
		}

		// define empty inputs to be overwritten by the pre create hook
		inputConfig := &container.Config{
			Image: def.image,
		}
		inputHostConfig := &container.HostConfig{}
		inputNetworkingConfig := &network.NetworkingConfig{}

		err := preCreateContainerHook(ctx, dockerClient, def, inputConfig, inputHostConfig, inputNetworkingConfig)
		require.NoError(t, err)

		// assertions

		require.Equal(
			t,
			def.networkAliases[networkName],
			inputNetworkingConfig.EndpointsConfig[networkName].Aliases,
			"Networking config's aliases should come from the container request",
		)
		require.Equal(
			t,
			nw.ID,
			inputNetworkingConfig.EndpointsConfig[networkName].NetworkID,
			"Networking config's network ID should be retrieved from Docker",
		)
	})

	t.Run("definition-contains-more-than-one-network-without-aliases", func(t *testing.T) {
		networkName := "foo" + t.Name()
		nw := testCreateNetwork(t, networkName)

		def := &Definition{
			image:    nginxAlpineImage, // alpine image does expose port 80
			networks: []string{networkName, "bar"},
		}

		// define empty inputs to be overwritten by the pre create hook
		inputConfig := &container.Config{
			Image: def.image,
		}
		inputHostConfig := &container.HostConfig{}
		inputNetworkingConfig := &network.NetworkingConfig{}

		err := preCreateContainerHook(ctx, dockerClient, def, inputConfig, inputHostConfig, inputNetworkingConfig)
		require.NoError(t, err)

		// assertions

		require.Empty(
			t,
			inputNetworkingConfig.EndpointsConfig[networkName].Aliases,
			"Networking config's aliases should be empty",
		)
		require.Equal(
			t,
			nw.ID,
			inputNetworkingConfig.EndpointsConfig[networkName].NetworkID,
			"Networking config's network ID should be retrieved from Docker",
		)
	})

	t.Run("definition-contains-exposed-port-modifiers-without-protocol", func(t *testing.T) {
		def := &Definition{
			image: nginxAlpineImage, // alpine image does expose port 80
			hostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = network.PortMap{
					network.MustParsePort("80/tcp"): []network.PortBinding{
						{
							HostIP:   netip.MustParseAddr("127.0.0.1"),
							HostPort: "8080",
						},
					},
				}
			},
			exposedPorts: []string{"80"},
		}

		// define empty inputs to be overwritten by the pre create hook
		inputConfig := &container.Config{
			Image: def.image,
		}
		inputHostConfig := &container.HostConfig{}
		inputNetworkingConfig := &network.NetworkingConfig{}

		err := preCreateContainerHook(ctx, dockerClient, def, inputConfig, inputHostConfig, inputNetworkingConfig)
		require.NoError(t, err)

		// assertions
		require.Equal(t, netip.MustParseAddr("127.0.0.1"), inputHostConfig.PortBindings[network.MustParsePort("80/tcp")][0].HostIP)
		require.Equal(t, "8080", inputHostConfig.PortBindings[network.MustParsePort("80/tcp")][0].HostPort)
	})

	t.Run("definition-contains-exposed-port-modifiers-with-protocol", func(t *testing.T) {
		def := &Definition{
			image: nginxAlpineImage, // alpine image does expose port 80
			hostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = network.PortMap{
					network.MustParsePort("80/tcp"): []network.PortBinding{
						{
							HostIP:   netip.MustParseAddr("127.0.0.1"),
							HostPort: "8080",
						},
					},
				}
			},
			exposedPorts: []string{"80/tcp"},
		}

		// define empty inputs to be overwritten by the pre create hook
		inputConfig := &container.Config{
			Image: def.image,
		}
		inputHostConfig := &container.HostConfig{}
		inputNetworkingConfig := &network.NetworkingConfig{}

		err := preCreateContainerHook(ctx, dockerClient, def, inputConfig, inputHostConfig, inputNetworkingConfig)
		require.NoError(t, err)

		// assertions
		require.Equal(t, netip.MustParseAddr("127.0.0.1"), inputHostConfig.PortBindings[network.MustParsePort("80/tcp")][0].HostIP)
		require.Equal(t, "8080", inputHostConfig.PortBindings[network.MustParsePort("80/tcp")][0].HostPort)
	})
}

func TestLifecycleHooks_withDefaultLogger(t *testing.T) {
	ctx := context.Background()

	buff := bytes.NewBuffer(nil)
	logger := slog.New(slog.NewTextHandler(buff, nil))

	cli, err := client.New(ctx, client.WithLogger(logger))
	require.NoError(t, err)

	c, err := Run(ctx,
		WithClient(cli),
		WithImage(nginxAlpineImage),
		WithLifecycleHooks(DefaultLoggingHook),
	)

	Cleanup(t, c)
	require.NoError(t, err)
	require.NotNil(t, c)

	err = c.Stop(ctx, StopTimeout(1*time.Second))
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)

	err = c.Terminate(ctx)
	require.NoError(t, err)

	// Includes four additional entries for stop (twice) when terminate is called.
	log := buff.String()
	require.Len(t, regexp.MustCompile("Starting container").FindAllString(log, -1), 4)
	require.Len(t, regexp.MustCompile("Stopping container").FindAllString(log, -1), 4)
}

func TestLifecycleHooks_WithMultipleHooks(t *testing.T) {
	ctx := context.Background()

	buff := bytes.NewBuffer(nil)
	logger := slog.New(slog.NewTextHandler(buff, nil))

	cli, err := client.New(ctx, client.WithLogger(logger))
	require.NoError(t, err)

	c, err := Run(ctx,
		WithClient(cli),
		WithImage(nginxAlpineImage),
		WithLifecycleHooks(DefaultLoggingHook, DefaultLoggingHook),
	)
	Cleanup(t, c)
	require.NoError(t, err)
	require.NotNil(t, c)

	err = c.Stop(ctx, StopTimeout(1*time.Second))
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)

	err = c.Terminate(ctx)
	require.NoError(t, err)

	// Includes six additional entries for stop (three times) when terminate is called:
	// - three default logging hooks times two for start and stop
	log := buff.String()
	require.Len(t, regexp.MustCompile("Starting container").FindAllString(log, -1), 6)
	require.Len(t, regexp.MustCompile("Stopping container").FindAllString(log, -1), 6)
}

func testCreateNetwork(t *testing.T, networkName string) dockerclient.NetworkCreateResult {
	t.Helper()

	dockerClient, err := client.New(context.TODO())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	nw, err := dockerClient.NetworkCreate(context.Background(), networkName, dockerclient.NetworkCreateOptions{})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := dockerClient.NetworkRemove(context.Background(), nw.ID, dockerclient.NetworkRemoveOptions{})
		require.NoError(t, err)
		require.NoError(t, dockerClient.Close())
	})

	return nw
}

func TestLifecycleHooks_withPullOptions(t *testing.T) {
	ctx := context.Background()

	cli, err := client.New(ctx)
	require.NoError(t, err)

	pullBuffer := bytes.NewBuffer(nil)

	c, err := Run(ctx,
		WithClient(cli),
		WithImage(nginxAlpineImage),
		WithAlwaysPull(),
		WithImagePlatform("linux/amd64"),
		WithPullHandler(func(r io.ReadCloser) error {
			_, err := io.Copy(pullBuffer, r)
			return err
		}),
	)
	Cleanup(t, c)
	require.NoError(t, err)
	require.NotNil(t, c)

	// the image should be pulled with the platform
	require.Contains(t, pullBuffer.String(), "Pulling from library/nginx")

	resp, err := cli.ImageInspect(ctx, nginxAlpineImage)
	require.NoError(t, err)
	require.Equal(t, "amd64", resp.Architecture)
	require.NotEqual(t, "arm64", resp.Architecture)
}
