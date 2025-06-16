package dockercontainer_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockercontainer"
	"github.com/docker/go-sdk/dockercontainer/exec"
	"github.com/docker/go-sdk/dockercontainer/wait"
	"github.com/docker/go-sdk/dockernetwork"
)

func TestRunContainer(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		t.Run("no-image", func(t *testing.T) {
			ctr, err := dockercontainer.Run(context.Background())
			require.Error(t, err)
			require.Nil(t, ctr)
		})

		t.Run("invalid-ports", func(t *testing.T) {
			ctr, err := dockercontainer.Run(context.Background(),
				dockercontainer.WithExposedPorts("invalid-port"),
			)
			require.Error(t, err)
			require.Nil(t, ctr)
		})

		t.Run("invalid-with-image-platform", func(t *testing.T) {
			ctr, err := dockercontainer.Run(context.Background(),
				dockercontainer.WithImage(nginxAlpineImage),
				dockercontainer.WithImagePlatform("invalid"),
			)
			dockercontainer.CleanupContainer(t, ctr)
			require.Error(t, err)
			require.Nil(t, ctr)
		})
	})

	t.Run("with-image", func(t *testing.T) {
		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("with-dockerclient", func(t *testing.T) {
		// Initialize the docker client. It will be closed when the container is terminated,
		// so no need to close it during the entire container lifecycle.
		dockerClient, err := dockerclient.New(context.Background())
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithDockerClient(dockerClient),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithExposedPorts("80/tcp"),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("with-files", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			helloSh := []byte(`#!/bin/sh
echo "hello world" > /tmp/hello.txt
echo "done"
`)

			ctr, err := dockercontainer.Run(context.Background(),
				dockercontainer.WithImage(nginxAlpineImage),
				dockercontainer.WithFiles(dockercontainer.File{
					ContainerPath: "/tmp/hello.sh",
					Reader:        bytes.NewReader(helloSh),
					Mode:          0o755,
				}),
			)
			dockercontainer.CleanupContainer(t, ctr)
			require.NoError(t, err)
			require.NotNil(t, ctr)

			code, r, err := ctr.Exec(context.Background(), []string{"/tmp/hello.sh"}, exec.Multiplexed())
			require.NoError(t, err)
			require.Equal(t, 0, code)

			buf := &bytes.Buffer{}
			_, err = io.Copy(buf, r)
			require.NoError(t, err)

			require.Equal(t, "done\n", buf.String())

			// Verify that the file can be copied out of the container.
			rc, err := ctr.CopyFromContainer(context.Background(), "/tmp/hello.txt")
			require.NoError(t, err)

			buf = &bytes.Buffer{}
			_, err = io.Copy(buf, rc)
			require.NoError(t, err)

			require.Equal(t, "hello world\n", buf.String())
		})

		t.Run("success/using-host-path", func(t *testing.T) {
			ctr, err := dockercontainer.Run(context.Background(),
				dockercontainer.WithImage(nginxAlpineImage),
				dockercontainer.WithFiles(dockercontainer.File{
					ContainerPath: "/tmp/hello.sh",
					HostPath:      path.Join("testdata", "hello.sh"),
					Mode:          0o755,
				}),
			)
			dockercontainer.CleanupContainer(t, ctr)
			require.NoError(t, err)
			require.NotNil(t, ctr)

			code, r, err := ctr.Exec(context.Background(), []string{"/tmp/hello.sh"}, exec.Multiplexed())
			require.NoError(t, err)
			require.Equal(t, 0, code)

			buf := &bytes.Buffer{}
			_, err = io.Copy(buf, r)
			require.NoError(t, err)

			require.Equal(t, "done\n", buf.String())

			// Verify that the file can be copied out of the container.
			rc, err := ctr.CopyFromContainer(context.Background(), "/tmp/hello.txt")
			require.NoError(t, err)

			buf = &bytes.Buffer{}
			_, err = io.Copy(buf, rc)
			require.NoError(t, err)

			require.Equal(t, "hello world\n", buf.String())
		})

		t.Run("error", func(t *testing.T) {
			ctr, err := dockercontainer.Run(context.Background(),
				dockercontainer.WithImage(nginxAlpineImage),
				dockercontainer.WithFiles(dockercontainer.File{
					ContainerPath: "/tmp/hello.sh",
					Reader:        nil,
					Mode:          0o755,
				}),
			)
			dockercontainer.CleanupContainer(t, ctr)
			require.Error(t, err)
		})
	})

	t.Run("with-config-modifier", func(t *testing.T) {
		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithConfigModifier(func(c *container.Config) {
				c.Env = append(c.Env, "ENV1=value1", "ENV2=value2")
				c.Hostname = "test-hostname"
			}),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		inspect, err := ctr.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.Config.Env, "ENV1=value1")
		require.Contains(t, inspect.Config.Env, "ENV2=value2")
		require.Equal(t, "test-hostname", inspect.Config.Hostname)
	})

	t.Run("with-host-config-modifier", func(t *testing.T) {
		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithHostConfigModifier(func(hc *container.HostConfig) {
				hc.CapDrop = []string{"NET_ADMIN"}
			}),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		inspect, err := ctr.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.HostConfig.CapDrop, "CAP_NET_ADMIN")
	})

	t.Run("with-endpoint-settings-modifier", func(t *testing.T) {
		name := "network-name"
		_ = testCreateNetwork(t, name)

		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithEndpointSettingsModifier(func(settings map[string]*network.EndpointSettings) {
				settings[name] = &network.EndpointSettings{
					Aliases: []string{"alias1", "alias2"},
				}
			}),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		inspect, err := ctr.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.NetworkSettings.Networks, name)
		require.Contains(t, inspect.NetworkSettings.Networks[name].Aliases, "alias1")
		require.Contains(t, inspect.NetworkSettings.Networks[name].Aliases, "alias2")
	})

	t.Run("with-log-consumer", func(t *testing.T) {
		lc := &dockercontainer.TestStringsLogConsumer{}
		ctx := context.Background()

		c, err := dockercontainer.Run(ctx,
			dockercontainer.WithImage("mysql:8.0.36"),
			dockercontainer.WithWaitStrategy(wait.ForLog("port: 3306  MySQL Community Server - GPL").WithTimeout(10*time.Second)),
			dockercontainer.WithLogConsumers(lc),
		)
		dockercontainer.CleanupContainer(t, c)
		// we expect an error because the MySQL environment variables are not set
		// but this is expected because we just want to test the log consumer
		require.Error(t, err)
		require.NotEmpty(t, lc.Messages())
	})

	t.Run("with-startup-command", func(t *testing.T) {
		ctx := context.Background()

		c, err := dockercontainer.Run(ctx,
			dockercontainer.WithImage(alpineLatest),
			dockercontainer.WithEntrypoint("tail", "-f", "/dev/null"),
			dockercontainer.WithStartupCommand(exec.NewRawCommand([]string{"touch", "/tmp/.dockercontainer-test"})),
		)
		dockercontainer.CleanupContainer(t, c)
		require.NoError(t, err)
		require.NotNil(t, c)

		_, reader, err := c.Exec(context.Background(), []string{"ls", "/tmp/.dockercontainer-test"}, exec.Multiplexed())
		require.NoError(t, err)

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, "/tmp/.dockercontainer-test\n", string(content))
	})

	t.Run("with-after-ready-command", func(t *testing.T) {
		ctx := context.Background()

		c, err := dockercontainer.Run(ctx,
			dockercontainer.WithImage(alpineLatest),
			dockercontainer.WithEntrypoint("tail", "-f", "/dev/null"),
			dockercontainer.WithAfterReadyCommand(exec.NewRawCommand([]string{"touch", "/tmp/.dockercontainer-test"})),
		)
		dockercontainer.CleanupContainer(t, c)
		require.NoError(t, err)
		require.NotNil(t, c)

		_, reader, err := c.Exec(context.Background(), []string{"ls", "/tmp/.dockercontainer-test"}, exec.Multiplexed())
		require.NoError(t, err)

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, "/tmp/.dockercontainer-test\n", string(content))
	})

	t.Run("no-dockerclient-uses-default", func(t *testing.T) {
		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("api-methods", func(t *testing.T) {
		ctr, err := dockercontainer.Run(context.Background(),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithImagePlatform("linux/amd64"),
			dockercontainer.WithAlwaysPull(),
		)
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		t.Run("host", func(t *testing.T) {
			host, err := ctr.Host(context.Background())
			require.NoError(t, err)
			require.NotEmpty(t, host)
		})

		t.Run("inspect", func(t *testing.T) {
			inspect, err := ctr.Inspect(context.Background())
			require.NoError(t, err)
			require.NotNil(t, inspect)

			require.Equal(t, ctr.ID(), inspect.ID)
			require.Equal(t, ctr.Image(), inspect.Config.Image)
		})

		t.Run("mapped-ports", func(t *testing.T) {
			port1, err := ctr.MappedPort(context.Background(), "80/tcp")
			require.NoError(t, err)
			require.NotNil(t, port1)
		})

		t.Run("state", func(t *testing.T) {
			c, err := dockercontainer.Run(context.Background(),
				dockercontainer.WithImage(nginxAlpineImage),
				dockercontainer.WithImagePlatform("linux/amd64"),
				dockercontainer.WithAlwaysPull(),
			)
			dockercontainer.CleanupContainer(t, c)
			require.NoError(t, err)
			require.NotNil(t, c)

			state, err := c.State(context.Background())
			require.NoError(t, err)
			require.NotNil(t, state)

			require.Equal(t, "running", state.Status)

			err = c.Stop(context.Background())
			require.NoError(t, err)

			state, err = c.State(context.Background())
			require.NoError(t, err)
			require.NotNil(t, state)
			require.Equal(t, "exited", state.Status)

			err = c.Terminate(context.Background())
			require.NoError(t, err)

			state, err = c.State(context.Background())
			require.Error(t, err)
			require.Nil(t, state)
		})
	})
}

func TestRunContainer_addSDKLabels(t *testing.T) {
	dockerClient, err := dockerclient.New(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	ctr, err := dockercontainer.Run(context.Background(),
		dockercontainer.WithDockerClient(dockerClient),
		dockercontainer.WithImage(nginxAlpineImage),
	)
	dockercontainer.CleanupContainer(t, ctr)
	require.NoError(t, err)
	require.NotNil(t, ctr)

	inspect, err := ctr.Inspect(context.Background())
	require.NoError(t, err)

	require.Contains(t, inspect.Config.Labels, dockerclient.LabelBase)
	require.Contains(t, inspect.Config.Labels, dockerclient.LabelLang)
	require.Contains(t, inspect.Config.Labels, dockerclient.LabelVersion)
}

func TestRunContainerWithLifecycleHooks(t *testing.T) {
	testRun := func(t *testing.T, start bool) {
		t.Helper()

		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		opts := []dockercontainer.ContainerCustomizer{
			dockercontainer.WithDockerClient(dockerClient),
			dockercontainer.WithImage(nginxAlpineImage),
			dockercontainer.WithLifecycleHooks(
				dockercontainer.LifecycleHooks{
					PreCreates: []dockercontainer.DefinitionHook{
						func(_ context.Context, def *dockercontainer.Definition) error {
							def.DockerClient().Logger().Info("pre-create hook")
							return nil
						},
					},
					PostCreates: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-create hook")
							return nil
						},
					},
					PreStarts: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("pre-start hook")
							return nil
						},
					},
					PostStarts: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-start hook")
							return nil
						},
					},
					PostReadies: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-ready hook")
							return nil
						},
					},
					PreStops: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("pre-stop hook")
							return nil
						},
					},
					PostStops: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-stop hook")
							return nil
						},
					},
					PreTerminates: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("pre-terminate hook")
							return nil
						},
					},
					PostTerminates: []dockercontainer.ContainerHook{
						func(_ context.Context, ctr *dockercontainer.Container) error {
							ctr.Logger().Info("post-terminate hook")
							return nil
						},
					},
				},
			),
		}

		if !start {
			opts = append(opts, dockercontainer.WithNoStart())
		}

		ctr, err := dockercontainer.Run(context.Background(), opts...)
		// cleanup the container: even if it's nil, it is handled by the CleanupContainer function
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		// because the container is not started, the pre-start hook, and beyond hooks, should not be called
		require.Contains(t, bufLogger.String(), "pre-create hook")
		require.Contains(t, bufLogger.String(), "post-create hook")

		if start {
			require.Contains(t, bufLogger.String(), "pre-start hook")
			require.Contains(t, bufLogger.String(), "post-start hook")
			require.Contains(t, bufLogger.String(), "post-ready hook")

			// force the container lifecycle methods to be called
			err = ctr.Stop(context.Background())
			require.NoError(t, err)
			require.Contains(t, bufLogger.String(), "pre-stop hook")
			require.Contains(t, bufLogger.String(), "post-stop hook")

			err = ctr.Terminate(context.Background())
			require.NoError(t, err)
			require.Contains(t, bufLogger.String(), "pre-terminate hook")
			require.Contains(t, bufLogger.String(), "post-terminate hook")
		}
	}

	t.Run("create-container", func(t *testing.T) {
		testRun(t, false)
	})

	t.Run("run-container", func(t *testing.T) {
		testRun(t, true)
	})
}

func TestRunContainerWithNetworks(t *testing.T) {
	testRun := func(t *testing.T, dockerClient *dockerclient.Client, networkOptions []dockercontainer.ContainerCustomizer) (*dockercontainer.Container, error) {
		t.Helper()

		opts := []dockercontainer.ContainerCustomizer{
			dockercontainer.WithDockerClient(dockerClient),
			dockercontainer.WithImage(nginxAlpineImage),
		}

		opts = append(opts, networkOptions...)

		return dockercontainer.Run(context.Background(), opts...)
	}

	testInspect := func(t *testing.T, ctr *dockercontainer.Container) *container.InspectResponse {
		t.Helper()

		inspect, err := ctr.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		return inspect
	}

	t.Run("with-network", func(t *testing.T) {
		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		nw, err := dockernetwork.New(context.Background(), dockernetwork.WithClient(dockerClient))
		require.NoError(t, err)
		dockernetwork.CleanupNetwork(t, nw)

		ctr, runErr := testRun(t, dockerClient, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithNetwork([]string{"ctr1"}, nw),
		})
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, runErr)

		inspect := testInspect(t, ctr)
		require.Len(t, inspect.NetworkSettings.Networks, 1)
		require.Equal(t, []string{"ctr1"}, inspect.NetworkSettings.Networks[nw.Name()].Aliases)
	})

	t.Run("with-bridge-network", func(t *testing.T) {
		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		nw, err := dockernetwork.New(context.Background(), dockernetwork.WithClient(dockerClient))
		require.NoError(t, err)
		dockernetwork.CleanupNetwork(t, nw)

		ctr, runErr := testRun(t, dockerClient, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithBridgeNetwork(),
		})
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, runErr)

		inspect := testInspect(t, ctr)
		require.Len(t, inspect.NetworkSettings.Networks, 1)
		require.Empty(t, inspect.NetworkSettings.Networks["bridge"].Aliases) // Bridge network does not support aliases
	})

	t.Run("with-new-network", func(t *testing.T) {
		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		ctr, runErr := testRun(t, dockerClient, []dockercontainer.ContainerCustomizer{
			// the network is going to be created using the same docker client
			dockercontainer.WithNewNetwork(context.Background(), []string{"ctr1"}, dockernetwork.WithClient(dockerClient)),
		})

		// We need to clean up the network first, else it fails
		// because the network would have active endpoints (containers)
		inspect := testInspect(t, ctr)
		for k := range inspect.NetworkSettings.Networks {
			dockernetwork.CleanupNetworkByID(t, k)
		}

		// Evaluate the run error last, as we need to clean up the network
		// before cleaning up the container
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, runErr)

		require.NotNil(t, inspect)
		require.Len(t, inspect.NetworkSettings.Networks, 1)
	})

	t.Run("with-network-name", func(t *testing.T) {
		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		newNetwork, err := dockernetwork.New(context.Background(), dockernetwork.WithClient(dockerClient))
		dockernetwork.CleanupNetwork(t, newNetwork)
		require.NoError(t, err)
		require.NotNil(t, newNetwork)

		ctr, err := testRun(t, dockerClient, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithNetworkName([]string{"ctr1"}, newNetwork.Name()),
		})
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		inspect := testInspect(t, ctr)
		require.Len(t, inspect.NetworkSettings.Networks, 1)
		require.Equal(t, []string{"ctr1"}, inspect.NetworkSettings.Networks[newNetwork.Name()].Aliases)
	})

	t.Run("with-multiple-networks", func(t *testing.T) {
		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		nw1, err := dockernetwork.New(context.Background(), dockernetwork.WithClient(dockerClient))
		require.NoError(t, err)
		dockernetwork.CleanupNetwork(t, nw1)

		nw2, err := dockernetwork.New(context.Background(), dockernetwork.WithClient(dockerClient))
		require.NoError(t, err)
		dockernetwork.CleanupNetwork(t, nw2)

		ctr, runErr := testRun(t, dockerClient, []dockercontainer.ContainerCustomizer{
			dockercontainer.WithNetwork([]string{"ctr1"}, nw1),
			dockercontainer.WithNetwork([]string{"ctr2"}, nw2),
		})
		dockercontainer.CleanupContainer(t, ctr)
		require.NoError(t, runErr)

		inspect := testInspect(t, ctr)
		require.Len(t, inspect.NetworkSettings.Networks, 2)
		require.Equal(t, []string{"ctr1"}, inspect.NetworkSettings.Networks[nw1.Name()].Aliases)
		require.Equal(t, []string{"ctr2"}, inspect.NetworkSettings.Networks[nw2.Name()].Aliases)
	})
}

func TestRunContainerWithWaitStrategy(t *testing.T) {
	testRun := func(t *testing.T, img string, strategy wait.Strategy, expectError bool) {
		t.Helper()

		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := dockerclient.New(context.Background(), dockerclient.WithLogger(logger))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		opts := []dockercontainer.ContainerCustomizer{
			dockercontainer.WithDockerClient(dockerClient),
			dockercontainer.WithImage(img),
			dockercontainer.WithFiles(dockercontainer.File{
				ContainerPath: "/tmp/hello.txt",
				Reader:        strings.NewReader(`hello world`),
				Mode:          0o644,
			}),
			dockercontainer.WithWaitStrategy(strategy),
		}

		ctr, err := dockercontainer.Run(context.Background(), opts...)
		dockercontainer.CleanupContainer(t, ctr)
		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.NotNil(t, ctr)
		}
	}

	t.Run("for-listening-port", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForListeningPort("80/tcp"), false)
	})

	t.Run("for-mapped-port", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForMappedPort("80/tcp"), false)
	})

	t.Run("for-exposed-port", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForExposedPort(), false)
	})

	t.Run("for-exec", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForExec([]string{"ls", "-l"}), false)
	})

	t.Run("for-file-exists", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForFile("/tmp/hello.txt"), false)
	})

	t.Run("for-file-does-not-exist", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForFile("/tmp/foo.txt").WithTimeout(1*time.Second), true)
	})

	t.Run("for-log", func(t *testing.T) {
		// log entry that is present in the nginx:alpine image
		testRun(t, nginxAlpineImage, wait.ForLog("start worker processes").WithTimeout(5*time.Second), false)
	})

	t.Run("for-exit/success", func(t *testing.T) {
		testRun(t, alpineLatest, wait.ForExit().WithExitTimeout(3*time.Second), false)
	})

	t.Run("for-exit/error", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForExit().WithExitTimeout(3*time.Second), true)
	})

	t.Run("for-http", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForHTTP("/"), false)
	})

	t.Run("for-http/error", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForHTTP("/not-found").WithTimeout(3*time.Second), true)
	})

	t.Run("for-http/with-status", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForHTTP("/not-found").WithStatus(http.StatusNotFound), false)
	})

	t.Run("for-http/with-status-code-matcher", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForHTTP("/").WithStatusCodeMatcher(func(status int) bool {
			return status == http.StatusOK
		}), false)
	})

	t.Run("for-http/with-response-matcher", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForHTTP("/not-found").WithStatus(http.StatusNotFound).WithResponseMatcher(func(body io.Reader) bool {
			content, err := io.ReadAll(body)
			require.NoError(t, err)

			// 404 response by the nginx:alpine image
			return strings.Contains(string(content), "<title>404 Not Found</title>")
		}), false)
	})
}

func testCreateNetwork(t *testing.T, networkName string) network.CreateResponse {
	t.Helper()

	dockerClient, err := dockerclient.New(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	nw, err := dockerClient.NetworkCreate(context.Background(), networkName, network.CreateOptions{})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := dockerClient.NetworkRemove(context.Background(), nw.ID)
		require.NoError(t, err)
		require.NoError(t, dockerClient.Close())
	})

	return nw
}
