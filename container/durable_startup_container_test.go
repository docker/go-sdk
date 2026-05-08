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
