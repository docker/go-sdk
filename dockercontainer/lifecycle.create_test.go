package dockercontainer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/dockerclient"
)

const nginxAlpineImage = "nginx:alpine"

func TestMergePortBindings(t *testing.T) {
	type arg struct {
		configPortMap nat.PortMap
		parsedPortMap nat.PortMap
		exposedPorts  []string
	}
	cases := []struct {
		name     string
		arg      arg
		expected nat.PortMap
	}{
		{
			name: "empty ports",
			arg: arg{
				configPortMap: nil,
				parsedPortMap: nil,
				exposedPorts:  nil,
			},
			expected: map[nat.Port][]nat.PortBinding{},
		},
		{
			name: "config port map but not exposed",
			arg: arg{
				configPortMap: map[nat.Port][]nat.PortBinding{
					"80/tcp": {{HostIP: "1", HostPort: "2"}},
				},
				parsedPortMap: nil,
				exposedPorts:  nil,
			},
			expected: map[nat.Port][]nat.PortBinding{},
		},
		{
			name: "parsed port map without config",
			arg: arg{
				configPortMap: nil,
				parsedPortMap: map[nat.Port][]nat.PortBinding{
					"80/tcp": {{HostIP: "", HostPort: ""}},
				},
				exposedPorts: nil,
			},
			expected: map[nat.Port][]nat.PortBinding{
				"80/tcp": {{HostIP: "", HostPort: ""}},
			},
		},
		{
			name: "parsed and configured but not exposed",
			arg: arg{
				configPortMap: map[nat.Port][]nat.PortBinding{
					"80/tcp": {{HostIP: "1", HostPort: "2"}},
				},
				parsedPortMap: map[nat.Port][]nat.PortBinding{
					"80/tcp": {{HostIP: "", HostPort: ""}},
				},
				exposedPorts: nil,
			},
			expected: map[nat.Port][]nat.PortBinding{
				"80/tcp": {{HostIP: "", HostPort: ""}},
			},
		},
		{
			name: "merge both parsed and config",
			arg: arg{
				configPortMap: map[nat.Port][]nat.PortBinding{
					"60/tcp": {{HostIP: "1", HostPort: "2"}},
					"70/tcp": {{HostIP: "1", HostPort: "2"}},
					"80/tcp": {{HostIP: "1", HostPort: "2"}},
				},
				parsedPortMap: map[nat.Port][]nat.PortBinding{
					"80/tcp": {{HostIP: "", HostPort: ""}},
					"90/tcp": {{HostIP: "", HostPort: ""}},
				},
				exposedPorts: []string{"70", "80/tcp"},
			},
			expected: map[nat.Port][]nat.PortBinding{
				"70/tcp": {{HostIP: "1", HostPort: "2"}},
				"80/tcp": {{HostIP: "1", HostPort: "2"}},
				"90/tcp": {{HostIP: "", HostPort: ""}},
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

	dockerClient, err := dockerclient.New(context.Background())
	require.NoError(t, err)
	defer dockerClient.Close()

	t.Run("no-exposed-ports", func(t *testing.T) {
		def := &Definition{
			image: nginxAlpineImage, // alpine image does expose port 80
			ConfigModifier: func(config *container.Config) {
				config.Env = []string{"a=b"}
			},
			HostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = nat.PortMap{
					"80/tcp": []nat.PortBinding{
						{
							HostIP:   "1",
							HostPort: "2",
						},
					},
				}
			},
			EndpointSettingsModifier: func(endpointSettings map[string]*network.EndpointSettings) {
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
			HostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = nat.PortMap{
					"80/tcp": []nat.PortBinding{
						{
							HostIP:   "1",
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
			nat.PortSet(nat.PortSet{}),
			inputConfig.ExposedPorts,
			"Docker config's exposed ports should be empty",
		)
		require.Equal(t,
			nat.PortMap{},
			inputHostConfig.PortBindings,
			"Host config's portBinding should be empty",
		)
	})

	t.Run("definition-contains-more-than-one-network-including-aliases", func(t *testing.T) {
		networkName := "foo" + t.Name()
		nw := testCreateNetwork(t, networkName)

		def := &Definition{
			image:    nginxAlpineImage, // alpine image does expose port 80
			Networks: []string{networkName, "bar"},
			NetworkAliases: map[string][]string{
				networkName: {"foo1"}, // network aliases are needed at the moment there is a network
			},
		}

		// define empty inputs to be overwritten by the pre create hook
		inputConfig := &container.Config{
			Image: def.image,
		}
		inputHostConfig := &container.HostConfig{}
		inputNetworkingConfig := &network.NetworkingConfig{}

		err = preCreateContainerHook(ctx, dockerClient, def, inputConfig, inputHostConfig, inputNetworkingConfig)
		require.NoError(t, err)

		// assertions

		require.Equal(
			t,
			def.NetworkAliases[networkName],
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
			Networks: []string{networkName, "bar"},
		}

		// define empty inputs to be overwritten by the pre create hook
		inputConfig := &container.Config{
			Image: def.image,
		}
		inputHostConfig := &container.HostConfig{}
		inputNetworkingConfig := &network.NetworkingConfig{}

		err = preCreateContainerHook(ctx, dockerClient, def, inputConfig, inputHostConfig, inputNetworkingConfig)
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
			HostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = nat.PortMap{
					"80/tcp": []nat.PortBinding{
						{
							HostIP:   "localhost",
							HostPort: "8080",
						},
					},
				}
			},
			ExposedPorts: []string{"80"},
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
		require.Equal(t, "localhost", inputHostConfig.PortBindings["80/tcp"][0].HostIP)
		require.Equal(t, "8080", inputHostConfig.PortBindings["80/tcp"][0].HostPort)
	})

	t.Run("definition-contains-exposed-port-modifiers-with-protocol", func(t *testing.T) {
		def := &Definition{
			image: nginxAlpineImage, // alpine image does expose port 80
			HostConfigModifier: func(hostConfig *container.HostConfig) {
				hostConfig.PortBindings = nat.PortMap{
					"80/tcp": []nat.PortBinding{
						{
							HostIP:   "localhost",
							HostPort: "8080",
						},
					},
				}
			},
			ExposedPorts: []string{"80/tcp"},
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
		require.Equal(t, "localhost", inputHostConfig.PortBindings["80/tcp"][0].HostIP)
		require.Equal(t, "8080", inputHostConfig.PortBindings["80/tcp"][0].HostPort)
	})
}

func testCreateNetwork(t *testing.T, networkName string) network.CreateResponse {
	t.Helper()

	dockerClient, err := dockerclient.New(context.Background())
	require.NoError(t, err)

	nw, err := dockerClient.Client().NetworkCreate(context.Background(), networkName, network.CreateOptions{})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := dockerClient.Client().NetworkRemove(context.Background(), nw.ID)
		require.NoError(t, err)
		require.NoError(t, dockerClient.Close())
	})

	return nw
}
