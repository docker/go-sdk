package container_test

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/container/exec"
)

// TestDurableStartupCommand_inRunningContainer is the load-bearing
// integration test: it copies the rendered scripts into a real container
// via the SDK's own file-copy path, invokes the dispatcher, and verifies
// side effects observed inside the container.
//
// It exercises, in one shot, the four properties that quoting bugs would
// break:
//
//   - argv passes through byte-exact (default namespace writes a tricky
//     argv array to a marker file)
//   - exec.WithEnv reaches the inner process as exported env vars
//   - exec.WithWorkingDir sets the inner process's CWD
//   - namespaces and within-namespace commands run in registration /
//     lexical order
func TestDurableStartupCommand_inRunningContainer(t *testing.T) {
	ctx := context.Background()

	c, err := container.Run(ctx,
		container.WithImage(alpineLatest),
		container.WithEntrypoint("tail", "-f", "/dev/null"),

		// Default namespace: writes a tricky argv to /tmp/argv. Every
		// flavor of shell metachar — if any of these were unquoted on the
		// way through, the file content would diverge from the input.
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

		// First named namespace: just appends a tag to /tmp/log to anchor
		// the ordering check.
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

		// The SDK option only renders + persists the scripts. Invocation
		// is the consumer's call: register the dispatcher as a regular
		// startup command so it fires once after the container starts.
		container.WithStartupCommand(exec.NewRawCommand(
			[]string{"sh", container.DurableStartupDispatcherPath},
		)),
	)
	container.Cleanup(t, c)
	require.NoError(t, err)
	require.NotNil(t, c)

	// Quoting check: the default-namespace script's tricky argv landed
	// byte-exact in /tmp/argv.
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
}

// TestDurableStartupCommand_inRunningContainer_withUser exercises the
// exec.WithUser path end-to-end. The script runs `id -un` and captures
// the result to a marker file; we then verify the file was actually
// written by the target user.
func TestDurableStartupCommand_inRunningContainer_withUser(t *testing.T) {
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
}

// TestDurableStartupCommand_inRunningContainer_userWithEnvAndWorkingDir
// proves the env/cd directives that get bundled inside the `su -c` body
// actually take effect in the inner shell. If our quoting were off, the
// inner shell would either fail to parse, or the env/cd would silently
// drop on the floor inside the user-switch boundary.
func TestDurableStartupCommand_inRunningContainer_userWithEnvAndWorkingDir(t *testing.T) {
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
}

// TestDurableStartupCommand_inRunningContainer_userMissingFailsClearly
// proves the failure-propagation contract: if `su` can't switch to the
// requested user (because they don't exist), the dispatcher's set -e
// causes a non-zero exit, AND the side-effect command in the script
// never runs.
//
// Note: the dispatcher's exit code is observed by invoking it directly
// (via [container.Container.Exec]). [container.WithStartupCommand] is NOT
// suitable for this assertion because the SDK's lifecycle hook
// for [container.WithStartupCommand] only checks the Docker-level exec
// error, not the inner process's exit code, and so silently swallows a
// non-zero dispatcher exit. Consumers who want first-create coverage to
// fail loudly should invoke the dispatcher themselves and check the
// returned exit code.
func TestDurableStartupCommand_inRunningContainer_userMissingFailsClearly(t *testing.T) {
	ctx := context.Background()

	c, err := container.Run(ctx,
		container.WithImage(alpineLatest),
		container.WithEntrypoint("tail", "-f", "/dev/null"),
		container.WithDurableStartupCommand(
			exec.NewRawCommand(
				// If su somehow succeeded, this would create the marker.
				// The marker's *absence* is part of the contract.
				[]string{"sh", "-c", "touch /tmp/should-not-exist"},
				exec.WithUser("definitely-not-a-real-user-xyz"),
			),
		),
	)
	container.Cleanup(t, c)
	require.NoError(t, err)
	require.NotNil(t, c)

	// Invoke the dispatcher directly so we can observe its exit code.
	code, reader, err := c.Exec(ctx,
		[]string{"sh", container.DurableStartupDispatcherPath},
		exec.Multiplexed(),
	)
	require.NoError(t, err) // The Docker-level exec must succeed even
	// when the inner command fails — Exec returns the inner exit code as
	// a value, not as an error. (See package container.exec.go.)
	out, _ := io.ReadAll(reader)
	require.NotEqual(t, 0, code,
		"dispatcher should exit non-zero when the requested user is missing\nstderr/stdout:\n%s",
		out,
	)

	// And the side-effect script must not have run — its `touch` never
	// fired, because su aborted before reaching it.
	absent, _, err := c.Exec(ctx,
		[]string{"test", "!", "-e", "/tmp/should-not-exist"},
		exec.Multiplexed(),
	)
	require.NoError(t, err)
	require.Equal(t, 0, absent,
		"marker file should not have been created — su should have failed before the script's body ran")
}

// TestDurableStartupCommand_inRunningContainer_perNamespaceUsers checks
// that different namespaces can each switch to a different user and the
// switches stay isolated to that namespace's commands.
func TestDurableStartupCommand_inRunningContainer_perNamespaceUsers(t *testing.T) {
	ctx := context.Background()

	c, err := container.Run(ctx,
		container.WithImage(alpineLatest),
		container.WithEntrypoint("tail", "-f", "/dev/null"),

		// Default namespace runs as root (no WithUser): expect "root".
		// Also seeds /tmp/log as world-writable so the subsequent nobody
		// scripts can append to it (otherwise the root-owned 0644 log
		// would block them — that's a property of the test scaffolding,
		// not anything the SDK can do anything about).
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
}

// TestDurableStartupCommand_inRunningContainer_layoutOnDisk confirms the
// script files actually land at the documented paths inside the container,
// and the dispatcher is at its well-known location.
func TestDurableStartupCommand_inRunningContainer_layoutOnDisk(t *testing.T) {
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

	// Walk the durable startup tree inside the container; the output must
	// match the documented layout exactly.
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
}
