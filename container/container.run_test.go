package container_test

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	apicontainer "github.com/moby/moby/api/types/container"
	apinetwork "github.com/moby/moby/api/types/network"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/container/exec"
	"github.com/docker/go-sdk/container/wait"
	"github.com/docker/go-sdk/network"
)

func TestRun(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		t.Run("no-image", func(t *testing.T) {
			ctr, err := container.Run(context.Background())
			require.Error(t, err)
			require.Nil(t, ctr)
		})

		t.Run("invalid-ports", func(t *testing.T) {
			ctr, err := container.Run(context.Background(),
				container.WithExposedPorts("invalid-port"),
			)
			require.Error(t, err)
			require.Nil(t, ctr)
		})

		t.Run("invalid-with-image-platform", func(t *testing.T) {
			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithImagePlatform("invalid"),
			)
			container.Cleanup(t, ctr)
			require.Error(t, err)
			require.Nil(t, ctr)
		})
	})

	t.Run("with-image", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("with-client", func(t *testing.T) {
		// Initialize the docker client. It will be closed when the container is terminated,
		// so no need to close it during the entire container lifecycle.
		dockerClient, err := client.New(context.Background())
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		ctr, err := container.Run(context.Background(),
			container.WithClient(dockerClient),
			container.WithImage(nginxAlpineImage),
			container.WithExposedPorts("80/tcp"),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("with-files", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			helloSh := []byte(`#!/bin/sh
echo "hello world" > /tmp/hello.txt
echo "done"
`)

			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithFiles(container.File{
					ContainerPath: "/tmp/hello.sh",
					Reader:        bytes.NewReader(helloSh),
					Mode:          0o755,
				}),
			)
			container.Cleanup(t, ctr)
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
			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithFiles(container.File{
					ContainerPath: "/tmp/hello.sh",
					HostPath:      path.Join("testdata", "hello.sh"),
					Mode:          0o755,
				}),
			)
			container.Cleanup(t, ctr)
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
			ctr, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithFiles(container.File{
					ContainerPath: "/tmp/hello.sh",
					Reader:        nil,
					Mode:          0o755,
				}),
			)
			container.Cleanup(t, ctr)
			require.Error(t, err)
		})
	})

	t.Run("with-config-modifier", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
			container.WithConfigModifier(func(c *apicontainer.Config) {
				c.Env = append(c.Env, "ENV1=value1", "ENV2=value2")
				c.Hostname = "test-hostname"
			}),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		inspect, err := ctr.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.Container.Config.Env, "ENV1=value1")
		require.Contains(t, inspect.Container.Config.Env, "ENV2=value2")
		require.Equal(t, "test-hostname", inspect.Container.Config.Hostname)
	})

	t.Run("with-host-config-modifier", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
			container.WithHostConfigModifier(func(hc *apicontainer.HostConfig) {
				hc.CapDrop = []string{"NET_ADMIN"}
			}),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		inspect, err := ctr.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.Container.HostConfig.CapDrop, "CAP_NET_ADMIN")
	})

	t.Run("with-endpoint-settings-modifier", func(t *testing.T) {
		name := "network-name"
		_ = testCreateNetwork(t, name)

		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
			container.WithEndpointSettingsModifier(func(settings map[string]*apinetwork.EndpointSettings) {
				settings[name] = &apinetwork.EndpointSettings{
					Aliases: []string{"alias1", "alias2"},
				}
			}),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		inspect, err := ctr.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		require.Contains(t, inspect.Container.NetworkSettings.Networks, name)
		require.Contains(t, inspect.Container.NetworkSettings.Networks[name].Aliases, "alias1")
		require.Contains(t, inspect.Container.NetworkSettings.Networks[name].Aliases, "alias2")
	})

	t.Run("with-startup-command", func(t *testing.T) {
		ctx := context.Background()

		c, err := container.Run(ctx,
			container.WithImage(alpineLatest),
			container.WithEntrypoint("tail", "-f", "/dev/null"),
			container.WithStartupCommand(exec.NewRawCommand([]string{"touch", "/tmp/.container-test"})),
		)
		container.Cleanup(t, c)
		require.NoError(t, err)
		require.NotNil(t, c)

		_, reader, err := c.Exec(context.Background(), []string{"ls", "/tmp/.container-test"}, exec.Multiplexed())
		require.NoError(t, err)

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, "/tmp/.container-test\n", string(content))
	})

	t.Run("with-after-ready-command", func(t *testing.T) {
		ctx := context.Background()

		c, err := container.Run(ctx,
			container.WithImage(alpineLatest),
			container.WithEntrypoint("tail", "-f", "/dev/null"),
			container.WithAfterReadyCommand(exec.NewRawCommand([]string{"touch", "/tmp/.container-test"})),
		)
		container.Cleanup(t, c)
		require.NoError(t, err)
		require.NotNil(t, c)

		_, reader, err := c.Exec(context.Background(), []string{"ls", "/tmp/.container-test"}, exec.Multiplexed())
		require.NoError(t, err)

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, "/tmp/.container-test\n", string(content))
	})

	t.Run("with-durable-startup-command", func(t *testing.T) {
		// Load-bearing integration test: the rendered scripts are copied
		// into the container via the SDK's own file-copy path, the
		// dispatcher is invoked, and we observe the side effects. This
		// exercises in one shot:
		//   - argv passes through byte-exact (default namespace writes a
		//     tricky argv array to a marker file)
		//   - exec.WithEnv reaches the inner process as exported env vars
		//   - exec.WithWorkingDir sets the inner process's CWD
		//   - namespaces and within-namespace commands run in registration
		//     / lexical order
		ctx := context.Background()

		c, err := container.Run(ctx,
			container.WithImage(alpineLatest),
			container.WithEntrypoint("tail", "-f", "/dev/null"),

			// Default namespace: writes a tricky argv to /tmp/argv. Every
			// flavor of shell metachar — if any of these were unquoted on
			// the way through, the file content would diverge from the
			// input.
			container.WithDurableStartupCommand(
				exec.NewRawCommand([]string{
					"sh", "-c", `printf '%s\n' "$@" >> /tmp/argv`, "_",
					"hello $USER",
					"with 'quote'",
					"back`tick`",
					`with "dq"`,
					"a; rm -rf /",
					"*",
				}),
			),

			// First named namespace: just appends a tag to /tmp/log to
			// anchor the ordering check.
			container.WithDurableStartupCommandsFromDir("pg",
				exec.NewRawCommand(
					[]string{"sh", "-c", `printf '%s\n' "pg-init" >> /tmp/log`},
				),
			),

			// Second named namespace: exercises WithEnv + WithWorkingDir
			// translation. The script should see K1 in its env and pwd
			// reporting /etc.
			container.WithDurableStartupCommandsFromDir("redis",
				exec.NewRawCommand(
					[]string{"sh", "-c", `printf '%s|%s\n' "$K1" "$(pwd)" >> /tmp/log`},
					exec.WithWorkingDir("/etc"),
					exec.WithEnv([]string{"K1=hello"}),
				),
			),

			// The SDK option only renders + persists the scripts.
			// Invocation is the consumer's call: register the dispatcher
			// as a regular startup command so it fires once after the
			// container starts.
			container.WithStartupCommand(exec.NewRawCommand(
				[]string{"sh", container.DurableStartupDispatcherPath},
			)),
		)
		container.Cleanup(t, c)
		require.NoError(t, err)
		require.NotNil(t, c)

		// Quoting check: the default-namespace script's tricky argv
		// landed byte-exact in /tmp/argv.
		_, r, err := c.Exec(ctx, []string{"cat", "/tmp/argv"}, exec.Multiplexed())
		require.NoError(t, err)
		argv, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t,
			"hello $USER\nwith 'quote'\nback`tick`\nwith \"dq\"\na; rm -rf /\n*\n",
			string(argv),
		)

		// Ordering + env + workingdir check: pg's marker comes before
		// redis's, and redis sees K1 + pwd=/etc.
		_, r, err = c.Exec(ctx, []string{"cat", "/tmp/log"}, exec.Multiplexed())
		require.NoError(t, err)
		log, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "pg-init\nhello|/etc\n", string(log))
	})

	t.Run("with-durable-startup-command-with-user", func(t *testing.T) {
		// Exercises exec.WithUser end-to-end. The script runs `id -un`
		// and captures the result to a marker file; we verify both the
		// content AND the file's owner.
		ctx := context.Background()

		c, err := container.Run(ctx,
			container.WithImage(alpineLatest),
			container.WithEntrypoint("tail", "-f", "/dev/null"),
			container.WithDurableStartupCommand(
				exec.NewRawCommand(
					[]string{"sh", "-c", `id -un > /tmp/whoami`},
					exec.WithUser("nobody"),
				),
			),
			container.WithStartupCommand(exec.NewRawCommand(
				[]string{"sh", container.DurableStartupDispatcherPath},
			)),
		)
		container.Cleanup(t, c)
		require.NoError(t, err)
		require.NotNil(t, c)

		_, r, err := c.Exec(ctx, []string{"cat", "/tmp/whoami"}, exec.Multiplexed())
		require.NoError(t, err)
		out, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "nobody\n", string(out))

		// And confirm the file's owner — su really switched user, didn't
		// just simulate it. `stat -c %U` is busybox-compatible.
		_, r, err = c.Exec(ctx, []string{"stat", "-c", "%U", "/tmp/whoami"}, exec.Multiplexed())
		require.NoError(t, err)
		owner, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "nobody\n", string(owner))
	})

	t.Run("with-durable-startup-command-user-env-and-workingdir", func(t *testing.T) {
		// The env/cd directives that get bundled inside the `su -c` body
		// must take effect in the inner shell. If quoting were off, the
		// inner shell would either fail to parse or the env/cd would
		// silently drop on the floor inside the user-switch boundary.
		ctx := context.Background()

		c, err := container.Run(ctx,
			container.WithImage(alpineLatest),
			container.WithEntrypoint("tail", "-f", "/dev/null"),
			container.WithDurableStartupCommand(
				exec.NewRawCommand(
					[]string{"sh", "-c", `printf '%s|%s|%s\n' "$(id -un)" "$K1" "$(pwd)" > /tmp/probe`},
					exec.WithUser("nobody"),
					exec.WithWorkingDir("/tmp"),
					exec.WithEnv([]string{"K1=hello world"}),
				),
			),
			container.WithStartupCommand(exec.NewRawCommand(
				[]string{"sh", container.DurableStartupDispatcherPath},
			)),
		)
		container.Cleanup(t, c)
		require.NoError(t, err)

		_, r, err := c.Exec(ctx, []string{"cat", "/tmp/probe"}, exec.Multiplexed())
		require.NoError(t, err)
		out, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "nobody|hello world|/tmp\n", string(out))
	})

	t.Run("with-durable-startup-command-missing-user-fails-clearly", func(t *testing.T) {
		// Failure-propagation contract: if `su` can't switch to the
		// requested user, the dispatcher's set -e causes a non-zero
		// exit, AND the side-effect command never runs.
		//
		// We invoke the dispatcher directly to observe its exit code.
		// container.WithStartupCommand isn't suitable for this assertion:
		// its lifecycle hook only checks the Docker-level exec error,
		// not the inner process's exit code, so it silently swallows a
		// non-zero dispatcher exit. Consumers who want first-create
		// coverage to fail loudly should invoke the dispatcher
		// themselves and check the returned exit code.
		ctx := context.Background()

		c, err := container.Run(ctx,
			container.WithImage(alpineLatest),
			container.WithEntrypoint("tail", "-f", "/dev/null"),
			container.WithDurableStartupCommand(
				exec.NewRawCommand(
					// If su somehow succeeded, this would create the
					// marker. The marker's *absence* is part of the
					// contract.
					[]string{"sh", "-c", "touch /tmp/should-not-exist"},
					exec.WithUser("definitely-not-a-real-user-xyz"),
				),
			),
		)
		container.Cleanup(t, c)
		require.NoError(t, err)
		require.NotNil(t, c)

		code, reader, err := c.Exec(ctx,
			[]string{"sh", container.DurableStartupDispatcherPath},
			exec.Multiplexed(),
		)
		require.NoError(t, err) // The Docker-level exec succeeds even
		// when the inner command fails — Exec returns the inner exit
		// code as a value, not as an error. (See container.exec.go.)
		out, _ := io.ReadAll(reader)
		require.NotEqual(t, 0, code,
			"dispatcher should exit non-zero when the requested user is missing\nstderr/stdout:\n%s",
			out,
		)

		// And the side-effect script must not have run — its `touch`
		// never fired, because su aborted before reaching it.
		absent, _, err := c.Exec(ctx,
			[]string{"test", "!", "-e", "/tmp/should-not-exist"},
			exec.Multiplexed(),
		)
		require.NoError(t, err)
		require.Equal(t, 0, absent,
			"marker file should not have been created — su should have failed before the script's body ran")
	})

	t.Run("with-durable-startup-command-per-namespace-users", func(t *testing.T) {
		// Different namespaces can each switch to a different user; the
		// switches stay isolated to their namespace's commands.
		ctx := context.Background()

		c, err := container.Run(ctx,
			container.WithImage(alpineLatest),
			container.WithEntrypoint("tail", "-f", "/dev/null"),

			// Default namespace runs as root (no WithUser): expect "root".
			// Also seeds /tmp/log as world-writable so the subsequent
			// nobody scripts can append to it (otherwise the root-owned
			// 0644 log would block them — a property of the test
			// scaffolding, not anything the SDK can do anything about).
			container.WithDurableStartupCommand(
				exec.NewRawCommand(
					[]string{"sh", "-c", `: > /tmp/log && chmod 666 /tmp/log && id -un >> /tmp/log`},
				),
			),
			// pg switches to nobody.
			container.WithDurableStartupCommandsFromDir("pg",
				exec.NewRawCommand(
					[]string{"sh", "-c", `id -un >> /tmp/log`},
					exec.WithUser("nobody"),
				),
			),
			// redis registers two scripts, only the first switches user.
			// Verifies the user switch doesn't bleed to the next script.
			container.WithDurableStartupCommandsFromDir("redis",
				exec.NewRawCommand(
					[]string{"sh", "-c", `id -un >> /tmp/log`},
					exec.WithUser("nobody"),
				),
				exec.NewRawCommand(
					[]string{"sh", "-c", `id -un >> /tmp/log`},
				),
			),
			container.WithStartupCommand(exec.NewRawCommand(
				[]string{"sh", container.DurableStartupDispatcherPath},
			)),
		)
		container.Cleanup(t, c)
		require.NoError(t, err)

		_, r, err := c.Exec(ctx, []string{"cat", "/tmp/log"}, exec.Multiplexed())
		require.NoError(t, err)
		out, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "root\nnobody\nnobody\nroot\n", string(out))
	})

	t.Run("with-durable-startup-command-layout-on-disk", func(t *testing.T) {
		// The script files land at the documented paths inside the
		// container, and the dispatcher is at its well-known location.
		ctx := context.Background()

		c, err := container.Run(ctx,
			container.WithImage(alpineLatest),
			container.WithEntrypoint("tail", "-f", "/dev/null"),
			container.WithDurableStartupCommand(
				exec.NewRawCommand([]string{"true"}),
			),
			container.WithDurableStartupCommandsFromDir("pg",
				exec.NewRawCommand([]string{"true"}),
				exec.NewRawCommand([]string{"true"}),
			),
		)
		container.Cleanup(t, c)
		require.NoError(t, err)

		_, r, err := c.Exec(ctx,
			[]string{"sh", "-c", "find " + container.DurableStartupDir + " -type f | LC_ALL=C sort"},
			exec.Multiplexed(),
		)
		require.NoError(t, err)
		out, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t,
			"/etc/durable-startup.d/000-default/000-cmd.sh\n"+
				"/etc/durable-startup.d/001-pg/000-cmd.sh\n"+
				"/etc/durable-startup.d/001-pg/001-cmd.sh\n"+
				"/etc/durable-startup.d/run.sh\n",
			string(out),
		)
	})

	t.Run("no-client-uses-default", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)
	})

	t.Run("api-methods", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(nginxAlpineImage),
			container.WithImagePlatform("linux/amd64"),
			container.WithAlwaysPull(),
		)
		container.Cleanup(t, ctr)
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

			require.Equal(t, ctr.ID(), inspect.Container.ID)
			require.Equal(t, ctr.Image(), inspect.Container.Config.Image)
		})

		t.Run("endpoint", func(t *testing.T) {
			endpoint, err := ctr.Endpoint(context.Background(), "http")
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(endpoint, "http://"))
			require.False(t, strings.HasSuffix(endpoint, ":80"))
		})

		t.Run("endpoint-no-ports", func(t *testing.T) {
			ctr, err := container.Run(context.Background(),
				container.WithImage(bashImage),
				container.WithWaitStrategy(wait.ForExit().WithTimeout(3*time.Second)),
			)
			container.Cleanup(t, ctr)
			require.NoError(t, err)

			endpoint, err := ctr.Endpoint(context.Background(), "http")
			require.ErrorIs(t, err, errdefs.ErrNotFound)
			require.Empty(t, endpoint)
		})

		t.Run("port-endpoint", func(t *testing.T) {
			portEndpoint, err := ctr.PortEndpoint(context.Background(), apinetwork.MustParsePort("80/tcp"), "tcp")
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(portEndpoint, "tcp://"))
		})

		t.Run("port-endpoint-not-found", func(t *testing.T) {
			portEndpoint, err := ctr.PortEndpoint(context.Background(), apinetwork.MustParsePort("3306/tcp"), "tcp")
			require.ErrorIs(t, err, errdefs.ErrNotFound)
			require.Empty(t, portEndpoint)
		})

		t.Run("mapped-port", func(t *testing.T) {
			mappedPort, err := ctr.MappedPort(context.Background(), apinetwork.MustParsePort("80/tcp"))
			require.NoError(t, err)
			require.NotNil(t, mappedPort)
			require.NotEqual(t, "80", mappedPort)
		})

		t.Run("mapped-port-not-found", func(t *testing.T) {
			mappedPort, err := ctr.MappedPort(context.Background(), apinetwork.MustParsePort("3306/tcp"))
			require.ErrorIs(t, err, errdefs.ErrNotFound)
			require.Empty(t, mappedPort)
		})

		t.Run("state", func(t *testing.T) {
			c, err := container.Run(context.Background(),
				container.WithImage(nginxAlpineImage),
				container.WithImagePlatform("linux/amd64"),
				container.WithAlwaysPull(),
			)
			container.Cleanup(t, c)
			require.NoError(t, err)
			require.NotNil(t, c)

			state, err := c.State(context.Background())
			require.NoError(t, err)
			require.NotNil(t, state)

			require.Equal(t, apicontainer.StateRunning, state.Status)

			err = c.Stop(context.Background())
			require.NoError(t, err)

			state, err = c.State(context.Background())
			require.NoError(t, err)
			require.NotNil(t, state)
			require.Equal(t, apicontainer.StateExited, state.Status)

			err = c.Terminate(context.Background())
			require.NoError(t, err)

			state, err = c.State(context.Background())
			require.Error(t, err)
			require.Nil(t, state)
		})
	})
}

func TestRun_addSDKLabels(t *testing.T) {
	dockerClient, err := client.New(context.TODO())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, dockerClient.Close())
	})

	ctr, err := container.Run(context.Background(),
		container.WithClient(dockerClient),
		container.WithImage(nginxAlpineImage),
	)
	container.Cleanup(t, ctr)
	require.NoError(t, err)
	require.NotNil(t, ctr)

	inspect, err := ctr.Inspect(context.Background())
	require.NoError(t, err)

	require.Contains(t, inspect.Container.Config.Labels, client.LabelBase)
	require.Contains(t, inspect.Container.Config.Labels, client.LabelLang)
	require.Contains(t, inspect.Container.Config.Labels, client.LabelVersion)
	require.Contains(t, inspect.Container.Config.Labels, client.LabelBase+".container")
	require.Equal(t, container.Version(), inspect.Container.Config.Labels[client.LabelBase+".container"])
}

//go:embed testdata/hello.sh
var helloBytes []byte

func TestRun_withFiles(t *testing.T) {
	t.Run("created-container/file", func(t *testing.T) {
		ctx, cnl := context.WithTimeout(context.Background(), 30*time.Second)
		defer cnl()

		// copyFileOnCreate {
		absPath, err := filepath.Abs(filepath.Join(".", "testdata", "hello.sh"))
		require.NoError(t, err)

		r, err := os.Open(absPath)
		require.NoError(t, err)

		ctr, err := container.Run(ctx,
			container.WithImage(bashImage),
			container.WithFiles(container.File{
				Reader:        r,
				HostPath:      absPath, // will be discarded internally because a reader is provided
				ContainerPath: "/hello.sh",
				Mode:          0o700,
			}),
			container.WithCmd("bash", "/hello.sh"),
			container.WithWaitStrategy(wait.ForLog("done")),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)
	})

	t.Run("created-container/directory", func(t *testing.T) {
		ctx, cnl := context.WithTimeout(context.Background(), 30*time.Second)
		defer cnl()

		// Not using the assertations here to avoid leaking the library into the example
		// copyDirectoryToContainer {
		dataDirectory, err := filepath.Abs(filepath.Join(".", "testdata"))
		require.NoError(t, err)

		ctr, err := container.Run(ctx,
			container.WithImage(bashImage),
			container.WithFiles(container.File{
				HostPath: dataDirectory,
				// ContainerFile cannot create the parent directory, so we copy the scripts
				// to the root of the container instead. Make sure to create the container directory
				// before you copy a host directory on create.
				ContainerPath: "/",
				Mode:          0o700,
			}),
			container.WithCmd("bash", "/testdata/hello.sh"),
			container.WithWaitStrategy(wait.ForLog("done")),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)
	})

	t.Run("running-container/file", func(t *testing.T) {
		ctx, cnl := context.WithTimeout(context.Background(), 30*time.Second)
		defer cnl()

		waitForPath, err := filepath.Abs(filepath.Join(".", "testdata", "waitForHello.sh"))
		require.NoError(t, err)

		ctr, err := container.Run(ctx,
			container.WithImage(bashImage),
			container.WithFiles(container.File{
				HostPath:      waitForPath,
				ContainerPath: "/waitForHello.sh",
				Mode:          0o700,
			}),
			container.WithCmd("bash", "/waitForHello.sh"),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)

		err = ctr.CopyToContainer(ctx, helloBytes, "/scripts/hello.sh", 0o700)
		require.NoError(t, err)

		// Give some time to the wait script to catch the hello script being created
		err = wait.ForLog("done").WithTimeout(2*time.Second).WaitUntilReady(ctx, ctr)
		require.NoError(t, err)
	})

	t.Run("running-container/directory", func(t *testing.T) {
		ctx, cnl := context.WithTimeout(context.Background(), 30*time.Second)
		defer cnl()

		// Not using the assertations here to avoid leaking the library into the example
		// copyDirectoryToRunningContainerAsDir {
		waitForPath, err := filepath.Abs(filepath.Join(".", "testdata", "waitForHello.sh"))
		require.NoError(t, err)
		dataDirectory, err := filepath.Abs(filepath.Join(".", "testdata"))
		require.NoError(t, err)

		ctr, err := container.Run(ctx,
			container.WithImage(bashImage),
			container.WithFiles(container.File{
				HostPath:      waitForPath,
				ContainerPath: "/waitForHello.sh",
				Mode:          0o700,
			}),
			container.WithCmd("bash", "/waitForHello.sh"),
		)
		container.Cleanup(t, ctr)
		require.NoError(t, err)

		// as the container is started, we can create the directory first
		_, _, err = ctr.Exec(ctx, []string{"mkdir", "-p", "/scripts"})
		require.NoError(t, err)

		err = ctr.CopyDirToContainer(ctx, dataDirectory, "/scripts/", 0o700)
		require.NoError(t, err)
	})
}

func TestRunWithLifecycleHooks(t *testing.T) {
	testRun := func(t *testing.T, start bool) {
		t.Helper()

		bufLogger := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(bufLogger, nil))

		dockerClient, err := client.New(context.Background(), client.WithLogger(logger))
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		opts := []container.ContainerCustomizer{
			container.WithClient(dockerClient),
			container.WithImage(nginxAlpineImage),
			container.WithLifecycleHooks(
				container.LifecycleHooks{
					PreCreates: []container.DefinitionHook{
						func(_ context.Context, def *container.Definition) error {
							def.DockerClient().Logger().Info("pre-create hook")
							return nil
						},
					},
					PostCreates: []container.ContainerHook{
						func(_ context.Context, ctr container.ContainerInfo) error {
							ctr.Logger().Info("post-create hook")
							return nil
						},
					},
					PreStarts: []container.ContainerHook{
						func(_ context.Context, ctr container.ContainerInfo) error {
							ctr.Logger().Info("pre-start hook")
							return nil
						},
					},
					PostStarts: []container.ContainerHook{
						func(_ context.Context, ctr container.ContainerInfo) error {
							ctr.Logger().Info("post-start hook")
							return nil
						},
					},
					PostReadies: []container.ContainerHook{
						func(_ context.Context, ctr container.ContainerInfo) error {
							ctr.Logger().Info("post-ready hook")
							return nil
						},
					},
					PreStops: []container.ContainerHook{
						func(_ context.Context, ctr container.ContainerInfo) error {
							ctr.Logger().Info("pre-stop hook")
							return nil
						},
					},
					PostStops: []container.ContainerHook{
						func(_ context.Context, ctr container.ContainerInfo) error {
							ctr.Logger().Info("post-stop hook")
							return nil
						},
					},
					PreTerminates: []container.ContainerHook{
						func(_ context.Context, ctr container.ContainerInfo) error {
							ctr.Logger().Info("pre-terminate hook")
							return nil
						},
					},
					PostTerminates: []container.ContainerHook{
						func(_ context.Context, ctr container.ContainerInfo) error {
							ctr.Logger().Info("post-terminate hook")
							return nil
						},
					},
				},
			),
		}

		if !start {
			opts = append(opts, container.WithNoStart())
		}

		ctr, err := container.Run(context.Background(), opts...)
		// cleanup the container: even if it's nil, it is handled by the Cleanup function
		container.Cleanup(t, ctr)
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

func TestRunWithNetworks(t *testing.T) {
	testRun := func(t *testing.T, dockerClient client.SDKClient, networkOptions []container.ContainerCustomizer) (*container.Container, error) {
		t.Helper()

		opts := []container.ContainerCustomizer{
			container.WithClient(dockerClient),
			container.WithImage(nginxAlpineImage),
		}

		opts = append(opts, networkOptions...)

		return container.Run(context.Background(), opts...)
	}

	testInspect := func(t *testing.T, ctr *container.Container) dockerclient.ContainerInspectResult {
		t.Helper()

		inspect, err := ctr.Inspect(context.Background())
		require.NoError(t, err)
		require.NotNil(t, inspect)

		return inspect
	}

	t.Run("with-network", func(t *testing.T) {
		dockerClient, err := client.New(context.TODO())
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, dockerClient.Close())
		})

		nw, err := network.New(context.Background(), network.WithClient(dockerClient))
		network.Cleanup(t, nw)
		require.NoError(t, err)

		ctr, runErr := testRun(t, dockerClient, []container.ContainerCustomizer{
			container.WithNetwork([]string{"ctr1"}, nw),
		})
		container.Cleanup(t, ctr)
		require.NoError(t, runErr)

		inspect := testInspect(t, ctr)
		require.Len(t, inspect.Container.NetworkSettings.Networks, 1)
		require.Equal(t, []string{"ctr1"}, inspect.Container.NetworkSettings.Networks[nw.Name()].Aliases)
	})

	t.Run("with-bridge-network", func(t *testing.T) {
		dockerClient, err := client.New(context.TODO())
		require.NoError(t, err)

		nw, err := network.New(context.Background(), network.WithClient(dockerClient))
		network.Cleanup(t, nw)
		require.NoError(t, err)

		ctr, runErr := testRun(t, dockerClient, []container.ContainerCustomizer{
			container.WithBridgeNetwork(),
		})
		container.Cleanup(t, ctr)
		require.NoError(t, runErr)

		inspect := testInspect(t, ctr)
		require.Len(t, inspect.Container.NetworkSettings.Networks, 1)
		require.Empty(t, inspect.Container.NetworkSettings.Networks["bridge"].Aliases) // Bridge network does not support aliases
	})

	t.Run("with-new-network", func(t *testing.T) {
		dockerClient, err := client.New(context.TODO())
		require.NoError(t, err)

		ctr, runErr := testRun(t, dockerClient, []container.ContainerCustomizer{
			// the network is going to be created using the same docker client
			container.WithNewNetwork(context.Background(), []string{"ctr1"}, network.WithClient(dockerClient)),
		})

		// We need to clean up the network first, else it fails
		// because the network would have active endpoints (containers)
		inspect := testInspect(t, ctr)
		for k := range inspect.Container.NetworkSettings.Networks {
			network.CleanupByID(t, k)
		}

		// Evaluate the run error last, as we need to clean up the network
		// before cleaning up the container
		container.Cleanup(t, ctr)
		require.NoError(t, runErr)

		require.NotNil(t, inspect)
		require.Len(t, inspect.Container.NetworkSettings.Networks, 1)
	})

	t.Run("with-network-name", func(t *testing.T) {
		dockerClient, err := client.New(context.TODO())
		require.NoError(t, err)

		newNetwork, err := network.New(context.Background(), network.WithClient(dockerClient))
		network.Cleanup(t, newNetwork)
		require.NoError(t, err)
		require.NotNil(t, newNetwork)

		ctr, err := testRun(t, dockerClient, []container.ContainerCustomizer{
			container.WithNetworkName([]string{"ctr1"}, newNetwork.Name()),
		})
		container.Cleanup(t, ctr)
		require.NoError(t, err)
		require.NotNil(t, ctr)

		inspect := testInspect(t, ctr)
		require.Len(t, inspect.Container.NetworkSettings.Networks, 1)
		require.Equal(t, []string{"ctr1"}, inspect.Container.NetworkSettings.Networks[newNetwork.Name()].Aliases)
	})

	t.Run("with-multiple-networks", func(t *testing.T) {
		dockerClient, err := client.New(context.TODO())
		require.NoError(t, err)

		nw1, err := network.New(context.Background(), network.WithClient(dockerClient))
		network.Cleanup(t, nw1)
		require.NoError(t, err)

		nw2, err := network.New(context.Background(), network.WithClient(dockerClient))
		network.Cleanup(t, nw2)
		require.NoError(t, err)

		ctr, runErr := testRun(t, dockerClient, []container.ContainerCustomizer{
			container.WithNetwork([]string{"ctr1"}, nw1),
			container.WithNetwork([]string{"ctr2"}, nw2),
		})
		container.Cleanup(t, ctr)
		require.NoError(t, runErr)

		inspect := testInspect(t, ctr)
		require.Len(t, inspect.Container.NetworkSettings.Networks, 2)
		require.Equal(t, []string{"ctr1"}, inspect.Container.NetworkSettings.Networks[nw1.Name()].Aliases)
		require.Equal(t, []string{"ctr2"}, inspect.Container.NetworkSettings.Networks[nw2.Name()].Aliases)
	})
}

func TestRunWithWaitStrategy(t *testing.T) {
	testRun := func(t *testing.T, img string, strategy wait.Strategy, expectError bool) {
		t.Helper()

		dockerClient, err := client.New(context.TODO())
		require.NoError(t, err)

		opts := []container.ContainerCustomizer{
			container.WithClient(dockerClient),
			container.WithImage(img),
			container.WithFiles(container.File{
				ContainerPath: "/tmp/hello.txt",
				Reader:        strings.NewReader(`hello world`),
				Mode:          0o644,
			}),
			container.WithWaitStrategy(strategy),
		}

		ctr, err := container.Run(context.Background(), opts...)
		container.Cleanup(t, ctr)
		if expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.NotNil(t, ctr)
		}
	}

	t.Run("for-listening-port", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForListeningPort(apinetwork.MustParsePort("80/tcp")), false)
	})

	t.Run("for-mapped-port", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForMappedPort(apinetwork.MustParsePort("80/tcp")), false)
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
		testRun(t, alpineLatest, wait.ForExit().WithTimeout(3*time.Second), false)
	})

	t.Run("for-exit/error", func(t *testing.T) {
		testRun(t, nginxAlpineImage, wait.ForExit().WithTimeout(3*time.Second), true)
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
