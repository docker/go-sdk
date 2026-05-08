package container

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container/exec"
)

func readFile(t *testing.T, f File) string {
	t.Helper()
	require.NotNil(t, f.Reader, "file %q has no reader", f.ContainerPath)
	bs, err := io.ReadAll(f.Reader)
	require.NoError(t, err)
	return string(bs)
}

func findFile(t *testing.T, files []File, containerPath string) File {
	t.Helper()
	for _, f := range files {
		if f.ContainerPath == containerPath {
			return f
		}
	}
	t.Fatalf("file %q not found in def.files", containerPath)
	return File{}
}

func filePresent(files []File, containerPath string) bool {
	for _, f := range files {
		if f.ContainerPath == containerPath {
			return true
		}
	}
	return false
}

func TestWithDurableStartupCommand(t *testing.T) {
	t.Run("writes-into-default-namespace", func(t *testing.T) {
		def := Definition{}

		err := WithDurableStartupCommand(
			exec.NewRawCommand([]string{"touch", "/tmp/.go-sdk"}),
		)(&def)
		require.NoError(t, err)

		// Default namespace at 000, single command at 000, plus dispatcher.
		require.Len(t, def.files, 2)

		script := findFile(t, def.files, DurableStartupDir+"/000-default/000-cmd.sh")
		require.Equal(t, durableStartupFileMode, script.Mode)
		body := readFile(t, script)
		require.Contains(t, body, "#!/bin/sh\n")
		require.Contains(t, body, "set -e\n")
		require.Contains(t, body, "exec 'touch' '/tmp/.go-sdk'")

		dispatcher := findFile(t, def.files, DurableStartupDispatcherPath)
		require.Equal(t, durableStartupFileMode, dispatcher.Mode)
		dispatcherBody := readFile(t, dispatcher)
		require.Contains(t, dispatcherBody, "ROOT='"+DurableStartupDir+"'")
		require.Contains(t, dispatcherBody, "LC_ALL=C")
		require.Contains(t, dispatcherBody, `for ns in "$ROOT"/*/`)
		require.Contains(t, dispatcherBody, `for f in "$ns"*-cmd.sh`)
	})

	t.Run("does-not-register-poststarts-hook", func(t *testing.T) {
		// Whole point of the durable variant: invocation is the consumer's
		// responsibility, so this option must NOT add a lifecycle hook.
		def := Definition{}

		err := WithDurableStartupCommand(
			exec.NewRawCommand([]string{"true"}),
		)(&def)
		require.NoError(t, err)
		require.Empty(t, def.lifecycleHooks)
	})

	t.Run("multiple-execs-numbered-sequentially-within-namespace", func(t *testing.T) {
		def := Definition{}

		err := WithDurableStartupCommand(
			exec.NewRawCommand([]string{"echo", "first"}),
			exec.NewRawCommand([]string{"echo", "second"}),
			exec.NewRawCommand([]string{"echo", "third"}),
		)(&def)
		require.NoError(t, err)

		require.Len(t, def.files, 4) // 3 cmds + dispatcher
		findFile(t, def.files, DurableStartupDir+"/000-default/000-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/000-default/001-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/000-default/002-cmd.sh")
	})

	t.Run("multiple-default-calls-stack", func(t *testing.T) {
		def := Definition{}

		require.NoError(t, WithDurableStartupCommand(
			exec.NewRawCommand([]string{"echo", "a"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommand(
			exec.NewRawCommand([]string{"echo", "b"}),
			exec.NewRawCommand([]string{"echo", "c"}),
		)(&def))

		findFile(t, def.files, DurableStartupDir+"/000-default/000-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/000-default/001-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/000-default/002-cmd.sh")
	})

	t.Run("translates-working-dir-and-env", func(t *testing.T) {
		def := Definition{}

		err := WithDurableStartupCommand(
			exec.NewRawCommand(
				[]string{"./run.sh", "--flag"},
				exec.WithWorkingDir("/srv/app"),
				exec.WithEnv([]string{"FOO=bar", "BAZ=qux"}),
			),
		)(&def)
		require.NoError(t, err)

		body := readFile(t, findFile(t, def.files, DurableStartupDir+"/000-default/000-cmd.sh"))
		require.Contains(t, body, "export FOO='bar'\n")
		require.Contains(t, body, "export BAZ='qux'\n")
		require.Contains(t, body, "cd '/srv/app'\n")
		require.Contains(t, body, "exec './run.sh' '--flag'\n")
		// env exports come before the cd, which comes before exec.
		require.Less(t, strings.Index(body, "export FOO"), strings.Index(body, "cd '/srv/app'"))
		require.Less(t, strings.Index(body, "cd '/srv/app'"), strings.Index(body, "exec "))
	})

	t.Run("shell-quotes-arguments-with-special-characters", func(t *testing.T) {
		def := Definition{}

		err := WithDurableStartupCommand(
			exec.NewRawCommand([]string{"sh", "-c", "echo 'hello world' && rm -rf /"}),
		)(&def)
		require.NoError(t, err)

		body := readFile(t, findFile(t, def.files, DurableStartupDir+"/000-default/000-cmd.sh"))
		// Embedded ' must be escaped via '\'' to keep the rendered script
		// from terminating the quote early — that's exactly the injection
		// the durable script is meant to prevent.
		require.Contains(t, body, `'sh' '-c' 'echo '\''hello world'\'' && rm -rf /'`)
	})

	t.Run("rejects-zero-execs", func(t *testing.T) {
		def := Definition{}
		err := WithDurableStartupCommand()(&def)
		require.ErrorContains(t, err, "at least one executable")
	})

	t.Run("rejects-empty-cmd", func(t *testing.T) {
		def := Definition{}
		err := WithDurableStartupCommand(exec.NewRawCommand(nil))(&def)
		require.ErrorContains(t, err, "empty command")
	})
}

func TestWithDurableStartupCommandsFromDir(t *testing.T) {
	t.Run("first-namespace-gets-index-001", func(t *testing.T) {
		def := Definition{}

		err := WithDurableStartupCommandsFromDir(
			"pg",
			exec.NewRawCommand([]string{"echo", "init"}),
		)(&def)
		require.NoError(t, err)

		findFile(t, def.files, DurableStartupDir+"/001-pg/000-cmd.sh")
		findFile(t, def.files, DurableStartupDispatcherPath)
		// Default namespace is NOT eagerly created — only the dispatcher
		// and the actual namespace are materialized.
		require.False(t, filePresent(def.files, DurableStartupDir+"/000-default/000-cmd.sh"))
	})

	t.Run("namespaces-indexed-by-registration-order", func(t *testing.T) {
		def := Definition{}

		require.NoError(t, WithDurableStartupCommandsFromDir(
			"pg",
			exec.NewRawCommand([]string{"echo", "pg"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommandsFromDir(
			"redis",
			exec.NewRawCommand([]string{"echo", "redis"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommandsFromDir(
			"mysql",
			exec.NewRawCommand([]string{"echo", "mysql"}),
		)(&def))

		findFile(t, def.files, DurableStartupDir+"/001-pg/000-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/002-redis/000-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/003-mysql/000-cmd.sh")
	})

	t.Run("default-always-precedes-named-namespaces-regardless-of-call-order", func(t *testing.T) {
		// Register pg first, then default, then more pg. Default must
		// still land at index 000 and pg at 001.
		def := Definition{}

		require.NoError(t, WithDurableStartupCommandsFromDir(
			"pg", exec.NewRawCommand([]string{"echo", "pg-1"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommand(
			exec.NewRawCommand([]string{"echo", "default-1"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommandsFromDir(
			"pg", exec.NewRawCommand([]string{"echo", "pg-2"}),
		)(&def))

		findFile(t, def.files, DurableStartupDir+"/000-default/000-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/001-pg/000-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/001-pg/001-cmd.sh")
	})

	t.Run("repeated-name-appends-to-same-namespace-dir", func(t *testing.T) {
		def := Definition{}

		require.NoError(t, WithDurableStartupCommandsFromDir(
			"pg",
			exec.NewRawCommand([]string{"echo", "a"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommandsFromDir(
			"pg",
			exec.NewRawCommand([]string{"echo", "b"}),
			exec.NewRawCommand([]string{"echo", "c"}),
		)(&def))

		findFile(t, def.files, DurableStartupDir+"/001-pg/000-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/001-pg/001-cmd.sh")
		findFile(t, def.files, DurableStartupDir+"/001-pg/002-cmd.sh")
		// And no spurious second pg dir.
		require.False(t, filePresent(def.files, DurableStartupDir+"/002-pg/000-cmd.sh"))
	})

	t.Run("dispatcher-rendered-once-across-many-calls", func(t *testing.T) {
		def := Definition{}

		require.NoError(t, WithDurableStartupCommand(
			exec.NewRawCommand([]string{"true"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommandsFromDir(
			"pg", exec.NewRawCommand([]string{"true"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommandsFromDir(
			"redis", exec.NewRawCommand([]string{"true"}),
		)(&def))

		var dispatchers int
		for _, f := range def.files {
			if f.ContainerPath == DurableStartupDispatcherPath {
				dispatchers++
			}
		}
		require.Equal(t, 1, dispatchers)
	})

	t.Run("rejects-default-as-explicit-name", func(t *testing.T) {
		def := Definition{}
		err := WithDurableStartupCommandsFromDir(
			"default", exec.NewRawCommand([]string{"true"}),
		)(&def)
		require.ErrorIs(t, err, ErrDurableStartupReservedNamespace)
	})

	t.Run("rejects-invalid-names", func(t *testing.T) {
		for _, name := range []string{
			"",        // empty
			"-foo",    // leading dash
			"_foo",    // leading underscore (must start alphanumeric)
			"Foo",     // uppercase
			"foo/bar", // slash
			"foo.bar", // dot
			"foo bar", // space
			"..",      // path traversal
			"foo$bar", // shell metachar
		} {
			err := WithDurableStartupCommandsFromDir(name, exec.NewRawCommand([]string{"true"}))(&Definition{})
			require.Error(t, err, "expected error for name %q", name)
		}
	})

	t.Run("accepts-valid-names", func(t *testing.T) {
		for _, name := range []string{"pg", "pg-15", "my_kit", "0kit", "kit-with-many-words"} {
			err := WithDurableStartupCommandsFromDir(name, exec.NewRawCommand([]string{"true"}))(&Definition{})
			require.NoError(t, err, "expected no error for name %q", name)
		}
	})

	t.Run("rejects-zero-execs", func(t *testing.T) {
		def := Definition{}
		err := WithDurableStartupCommandsFromDir("pg")(&def)
		require.ErrorContains(t, err, "at least one executable")
	})

	t.Run("preserves-prior-files", func(t *testing.T) {
		def := Definition{
			files: []File{
				{ContainerPath: "/already/here.txt", Mode: 0o644},
			},
		}

		err := WithDurableStartupCommandsFromDir(
			"pg",
			exec.NewRawCommand([]string{"true"}),
		)(&def)
		require.NoError(t, err)

		require.Len(t, def.files, 3) // existing + cmd + dispatcher
		require.Equal(t, "/already/here.txt", def.files[0].ContainerPath)
	})
}

// ---------------------------------------------------------------------------
// Helpers and internal-function tests
// ---------------------------------------------------------------------------

func TestShellSingleQuote(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "''"},
		{"plain", "hello", "'hello'"},
		{"with-space", "hello world", "'hello world'"},
		{"trailing-quote", "don't", `'don'\''t'`},
		{"only-single-quote", "'", `''\'''`},
		{"multiple-quotes", "a'b'c", `'a'\''b'\''c'`},
		{"backslash", `\`, `'\'`},
		{"dollar-var", "$VAR", "'$VAR'"},
		{"backticks", "`cmd`", "'`cmd`'"},
		{"double-quotes", `"q"`, `'"q"'`},
		{"newline", "a\nb", "'a\nb'"},
		{"tab", "a\tb", "'a\tb'"},
		{"all-quotes", "''", `''\'''\'''`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, shellSingleQuote(tc.in))
		})
	}
}

func TestRenderDurableScript(t *testing.T) {
	t.Run("nil-executable", func(t *testing.T) {
		_, err := renderDurableScript(nil)
		require.ErrorContains(t, err, "executable is nil")
	})

	t.Run("empty-cmd", func(t *testing.T) {
		_, err := renderDurableScript(exec.NewRawCommand(nil))
		require.ErrorContains(t, err, "empty command")
	})

	t.Run("empty-cmd-explicit-empty-slice", func(t *testing.T) {
		_, err := renderDurableScript(exec.NewRawCommand([]string{}))
		require.ErrorContains(t, err, "empty command")
	})

	t.Run("minimal-cmd", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand([]string{"true"}))
		require.NoError(t, err)
		require.Equal(t, "#!/bin/sh\nset -e\nexec 'true'\n", out)
	})

	t.Run("cmd-with-args", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand([]string{"echo", "hello", "world"}))
		require.NoError(t, err)
		require.Equal(t, "#!/bin/sh\nset -e\nexec 'echo' 'hello' 'world'\n", out)
	})

	t.Run("env-with-empty-value", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithEnv([]string{"K="}),
		))
		require.NoError(t, err)
		require.Contains(t, out, "export K=''\n")
	})

	t.Run("env-without-equals-is-skipped", func(t *testing.T) {
		// Malformed env entry — silently dropped rather than producing
		// a syntactically broken `export ` line.
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithEnv([]string{"NOEQUALS"}),
		))
		require.NoError(t, err)
		require.NotContains(t, out, "export NOEQUALS")
		require.NotContains(t, out, "export ")
	})

	t.Run("env-with-empty-key-is-skipped", func(t *testing.T) {
		// "=value" has eq==0, which our guard rejects.
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithEnv([]string{"=value"}),
		))
		require.NoError(t, err)
		require.NotContains(t, out, "export ")
	})

	t.Run("env-multiple-and-ordered", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithEnv([]string{"A=1", "B=2", "C=3"}),
		))
		require.NoError(t, err)
		// Order preserved.
		ai := strings.Index(out, "export A=")
		bi := strings.Index(out, "export B=")
		ci := strings.Index(out, "export C=")
		require.Less(t, ai, bi)
		require.Less(t, bi, ci)
	})

	t.Run("env-value-with-equals-sign-preserved", func(t *testing.T) {
		// Only the FIRST '=' separates key and value. Subsequent '=' are
		// part of the value (e.g. URLs with query strings).
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithEnv([]string{"URL=http://x?a=1&b=2"}),
		))
		require.NoError(t, err)
		require.Contains(t, out, "export URL='http://x?a=1&b=2'\n")
	})

	t.Run("env-value-with-special-chars-quoted", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithEnv([]string{`PASS=p'a"$s `}),
		))
		require.NoError(t, err)
		require.Contains(t, out, `export PASS='p'\''a"$s '`+"\n")
	})

	t.Run("working-dir-quoted", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithWorkingDir("/path with spaces/'tricky'"),
		))
		require.NoError(t, err)
		require.Contains(t, out, `cd '/path with spaces/'\''tricky'\'''`+"\n")
	})

	t.Run("nil-process-option-is-ignored", func(t *testing.T) {
		// renderDurableScript skips nil entries in Options() instead of
		// dereferencing them.
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			nil,
			exec.WithWorkingDir("/srv"),
			nil,
		))
		require.NoError(t, err)
		require.Contains(t, out, "cd '/srv'\n")
	})

	t.Run("ordering-env-then-cd-then-exec", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"./go"},
			exec.WithEnv([]string{"X=1"}),
			exec.WithWorkingDir("/opt"),
		))
		require.NoError(t, err)
		header := strings.Index(out, "set -e\n")
		envI := strings.Index(out, "export X=")
		cdI := strings.Index(out, "cd '/opt'")
		execI := strings.Index(out, "exec './go'")
		require.True(t, header < envI && envI < cdI && cdI < execI,
			"unexpected ordering in:\n%s", out)
	})

	t.Run("args-with-newlines-and-special-shell-metachars", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand([]string{
			"sh", "-c",
			"echo $HOME\n`whoami`\n\"$@\" && rm -rf /",
		}))
		require.NoError(t, err)
		// Single-quote wrapping preserves all of the above as a literal arg.
		require.Contains(t, out,
			"exec 'sh' '-c' 'echo $HOME\n`whoami`\n\"$@\" && rm -rf /'\n",
		)
	})

	t.Run("with-user-wraps-body-in-su", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithUser("nobody"),
		))
		require.NoError(t, err)
		// Top-level script just exec's su; the body is fully inside the -c arg.
		require.Contains(t, out, "exec su -s /bin/sh -c ")
		require.Contains(t, out, " 'nobody'\n")
		// The exec line for the actual command must NOT appear at the
		// top level — it lives inside the single-quoted -c body.
		topLevel := strings.SplitN(out, "exec su", 2)[0]
		require.NotContains(t, topLevel, "exec 'true'",
			"unwrapped exec leaked above the su wrapper:\n%s", out)
	})

	t.Run("with-user-includes-set-e-in-inner-body", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithUser("nobody"),
		))
		require.NoError(t, err)
		// Inner body's set -e shows up as the escaped \''set -e\n'\''
		// pattern? No — single-quoted, so it stays literal `set -e`. We
		// just check that "set -e" appears at least twice: once at the
		// top of the script, once at the start of the -c arg.
		require.GreaterOrEqual(t, strings.Count(out, "set -e\n"), 2,
			"expected set -e in both outer and inner shells:\n%s", out)
	})

	t.Run("with-user-combined-with-env-and-workingdir", func(t *testing.T) {
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"./run.sh"},
			exec.WithUser("nobody"),
			exec.WithEnv([]string{"K=v"}),
			exec.WithWorkingDir("/srv"),
		))
		require.NoError(t, err)
		// Env + cd + exec all live inside the -c arg, double-quoted.
		// Outer single-quote escaping makes them appear as 'export...''cd...''exec...'
		// after the '\'' substitutions; easiest assertion: those tokens
		// appear AFTER the `exec su -s /bin/sh -c ` prefix.
		ix := strings.Index(out, "exec su -s /bin/sh -c ")
		require.GreaterOrEqual(t, ix, 0)
		require.Contains(t, out[ix:], "export K=")
		require.Contains(t, out[ix:], "cd ")
		require.Contains(t, out[ix:], "exec ")
		require.Contains(t, out[ix:], "./run.sh")
	})

	t.Run("with-user-empty-string-not-honored", func(t *testing.T) {
		// WithUser("") leaves ExecConfig.User as "". Don't wrap in su.
		out, err := renderDurableScript(exec.NewRawCommand(
			[]string{"true"},
			exec.WithUser(""),
		))
		require.NoError(t, err)
		require.NotContains(t, out, "su ")
		require.Contains(t, out, "exec 'true'\n")
	})

	t.Run("with-user-rejects-uid-gid-and-non-login-forms", func(t *testing.T) {
		// `su` only resolves login names. Docker user specs like
		// uid:gid, user:group, and bare-UIDs won't work at runtime, so
		// reject them at definition time with a clear error pointing to
		// WithStartupCommand for full Docker user-spec semantics.
		for _, bad := range []string{
			"1000",             // bare UID
			"1000:1000",        // uid:gid
			"appuser:appgroup", // user:group
			"-leadingdash",     // not a valid login name
			"user with space",  // space
			"user'quote",       // shell metachar
			"user;rm -rf /",    // injection attempt
			"",                 // see note: "" means "no user", not invalid
		} {
			if bad == "" {
				continue // empty user means "don't switch", separately covered
			}
			_, err := renderDurableScript(exec.NewRawCommand(
				[]string{"true"},
				exec.WithUser(bad),
			))
			require.Error(t, err, "expected error for WithUser %q", bad)
			require.Contains(t, err.Error(), bad,
				"error must include the offending value for diagnosability")
			require.Contains(t, err.Error(), "WithStartupCommand",
				"error should point users to the fallback")
		}
	})

	t.Run("with-user-accepts-login-names", func(t *testing.T) {
		for _, good := range []string{
			"root",
			"nobody",
			"appuser",
			"_systemd",
			"pg-15",
			"user_with_underscore",
			"User",
		} {
			_, err := renderDurableScript(exec.NewRawCommand(
				[]string{"true"},
				exec.WithUser(good),
			))
			require.NoError(t, err, "expected no error for WithUser %q", good)
		}
	})

	t.Run("rendering-is-deterministic", func(t *testing.T) {
		// Same exec → byte-identical output. Load-bearing for the
		// "deterministic" contract documented on the option.
		mk := func() string {
			out, err := renderDurableScript(exec.NewRawCommand(
				[]string{"./run.sh", "--flag"},
				exec.WithWorkingDir("/srv"),
				exec.WithEnv([]string{"A=1", "B=2"}),
			))
			require.NoError(t, err)
			return out
		}
		first := mk()
		second := mk()
		require.Equal(t, first, second)
	})
}

func TestRenderDurableDispatcher(t *testing.T) {
	out := renderDurableDispatcher("/etc/durable-startup.d")

	require.True(t, strings.HasPrefix(out, "#!/bin/sh\n"), "missing shebang")
	require.Contains(t, out, "set -e\n")
	// Locale pinned so glob sort is byte-deterministic.
	require.Contains(t, out, "LC_ALL=C\nexport LC_ALL\n")
	require.Contains(t, out, "ROOT='/etc/durable-startup.d'\n")
	// Outer loop iterates namespace subdirs only (trailing slash on glob).
	require.Contains(t, out, `for ns in "$ROOT"/*/`)
	// Inner loop only picks up *-cmd.sh, so the dispatcher script and any
	// stray non-cmd files in a namespace are ignored.
	require.Contains(t, out, `for f in "$ns"*-cmd.sh`)
	// Defends against a missing root directory.
	require.Contains(t, out, `[ -d "$ROOT" ] || exit 0`)
}

func TestRenderDurableDispatcher_quotesUnusualRoot(t *testing.T) {
	out := renderDurableDispatcher("/path with 'quotes'/d")
	require.Contains(t, out, `ROOT='/path with '\''quotes'\''/d'`+"\n")
}

func TestResolveDurableNamespaceDir(t *testing.T) {
	t.Run("empty-files-default", func(t *testing.T) {
		dir, present := resolveDurableNamespaceDir(nil, "default")
		require.Equal(t, "/etc/durable-startup.d/000-default", dir)
		require.False(t, present)
	})

	t.Run("empty-files-named", func(t *testing.T) {
		dir, present := resolveDurableNamespaceDir(nil, "pg")
		require.Equal(t, "/etc/durable-startup.d/001-pg", dir)
		require.False(t, present)
	})

	t.Run("reuses-existing-namespace-dir", func(t *testing.T) {
		files := []File{
			{ContainerPath: "/etc/durable-startup.d/001-pg/000-cmd.sh"},
		}
		dir, _ := resolveDurableNamespaceDir(files, "pg")
		require.Equal(t, "/etc/durable-startup.d/001-pg", dir)
	})

	t.Run("default-still-zero-when-named-already-present", func(t *testing.T) {
		// Even if a named namespace was registered first, default's index
		// is reserved at 000.
		files := []File{
			{ContainerPath: "/etc/durable-startup.d/001-pg/000-cmd.sh"},
		}
		dir, _ := resolveDurableNamespaceDir(files, "default")
		require.Equal(t, "/etc/durable-startup.d/000-default", dir)
	})

	t.Run("allocates-next-index-after-default", func(t *testing.T) {
		files := []File{
			{ContainerPath: "/etc/durable-startup.d/000-default/000-cmd.sh"},
		}
		dir, _ := resolveDurableNamespaceDir(files, "pg")
		require.Equal(t, "/etc/durable-startup.d/001-pg", dir)
	})

	t.Run("allocates-after-multiple-existing-namespaces", func(t *testing.T) {
		files := []File{
			{ContainerPath: "/etc/durable-startup.d/000-default/000-cmd.sh"},
			{ContainerPath: "/etc/durable-startup.d/001-pg/000-cmd.sh"},
			{ContainerPath: "/etc/durable-startup.d/002-redis/000-cmd.sh"},
		}
		dir, _ := resolveDurableNamespaceDir(files, "mysql")
		require.Equal(t, "/etc/durable-startup.d/003-mysql", dir)
	})

	t.Run("dispatcher-detection", func(t *testing.T) {
		files := []File{
			{ContainerPath: DurableStartupDispatcherPath},
		}
		_, present := resolveDurableNamespaceDir(files, "pg")
		require.True(t, present)
	})

	t.Run("ignores-unrelated-files", func(t *testing.T) {
		files := []File{
			{ContainerPath: "/etc/other.txt"},
			{ContainerPath: "/var/log/something.sh"},
		}
		dir, present := resolveDurableNamespaceDir(files, "pg")
		require.Equal(t, "/etc/durable-startup.d/001-pg", dir)
		require.False(t, present)
	})

	t.Run("ignores-files-directly-in-root-without-namespace", func(t *testing.T) {
		// Stray files under DurableStartupDir that aren't in a namespace
		// subdir don't perturb namespace allocation.
		files := []File{
			{ContainerPath: "/etc/durable-startup.d/stray-file"},
		}
		dir, _ := resolveDurableNamespaceDir(files, "pg")
		require.Equal(t, "/etc/durable-startup.d/001-pg", dir)
	})

	t.Run("ignores-malformed-namespace-dir-names", func(t *testing.T) {
		files := []File{
			// missing dash, missing prefix, trailing dash, non-numeric prefix
			{ContainerPath: "/etc/durable-startup.d/nodash/foo.sh"},
			{ContainerPath: "/etc/durable-startup.d/-leadingdash/foo.sh"},
			{ContainerPath: "/etc/durable-startup.d/abc-name/foo.sh"},
		}
		dir, _ := resolveDurableNamespaceDir(files, "pg")
		// None of the malformed entries count toward the index.
		require.Equal(t, "/etc/durable-startup.d/001-pg", dir)
	})

	t.Run("preserves-foreign-but-well-formed-namespace-dirs", func(t *testing.T) {
		// A consumer could pre-populate def.files with a well-formed
		// "NNN-name" path. We treat it as a registered namespace, so the
		// next allocation skips its index.
		files := []File{
			{ContainerPath: "/etc/durable-startup.d/005-foreign/000-cmd.sh"},
		}
		dir, _ := resolveDurableNamespaceDir(files, "pg")
		// One non-default namespace is already known → next is 002.
		// (Indices need not be contiguous; we just count registered names.)
		require.Equal(t, "/etc/durable-startup.d/002-pg", dir)
	})

	t.Run("name-with-dashes-is-handled", func(t *testing.T) {
		files := []File{
			{ContainerPath: "/etc/durable-startup.d/001-pg-15/000-cmd.sh"},
		}
		dir, _ := resolveDurableNamespaceDir(files, "pg-15")
		require.Equal(t, "/etc/durable-startup.d/001-pg-15", dir)
	})
}

func TestNextDurableCmdIndex(t *testing.T) {
	const ns = "/etc/durable-startup.d/001-pg"

	t.Run("empty", func(t *testing.T) {
		require.Equal(t, 0, nextDurableCmdIndex(nil, ns))
	})

	t.Run("counts-only-cmd-files-in-this-dir", func(t *testing.T) {
		files := []File{
			{ContainerPath: ns + "/000-cmd.sh"},
			{ContainerPath: ns + "/001-cmd.sh"},
			{ContainerPath: ns + "/README.txt"},                            // not -cmd.sh
			{ContainerPath: "/elsewhere/000-cmd.sh"},                       // not in this dir
			{ContainerPath: "/etc/durable-startup.d/002-redis/000-cmd.sh"}, // sibling ns
		}
		require.Equal(t, 2, nextDurableCmdIndex(files, ns))
	})

	t.Run("ignores-nested-paths", func(t *testing.T) {
		files := []File{
			{ContainerPath: ns + "/sub/000-cmd.sh"}, // path.Dir != ns
		}
		require.Equal(t, 0, nextDurableCmdIndex(files, ns))
	})
}

// ---------------------------------------------------------------------------
// Higher-level scenarios and integration-style tests
// ---------------------------------------------------------------------------

func TestWithDurableStartupCommand_isTransactionalOnRenderError(t *testing.T) {
	// If an exec mid-list fails to render, def.files must not contain
	// any of the partial state from the call.
	def := Definition{
		files: []File{{ContainerPath: "/sentinel"}},
	}

	err := WithDurableStartupCommand(
		exec.NewRawCommand([]string{"good-1"}),
		exec.NewRawCommand(nil), // bad
		exec.NewRawCommand([]string{"good-2"}),
	)(&def)
	require.Error(t, err)

	// Only the pre-existing sentinel — no scripts, no dispatcher.
	require.Len(t, def.files, 1)
	require.Equal(t, "/sentinel", def.files[0].ContainerPath)
}

func TestWithDurableStartupCommandsFromDir_isTransactionalOnRenderError(t *testing.T) {
	def := Definition{}

	err := WithDurableStartupCommandsFromDir("pg",
		exec.NewRawCommand([]string{"good"}),
		exec.NewRawCommand(nil),
	)(&def)
	require.Error(t, err)
	require.Empty(t, def.files)
}

func TestWithDurableStartupCommandsFromDir_validationErrorsBeforeMutation(t *testing.T) {
	// Validation errors (reserved name, bad regex) must not append a
	// dispatcher or any partial state.
	for _, name := range []string{"default", "Foo", "foo/bar", ""} {
		def := Definition{}
		err := WithDurableStartupCommandsFromDir(name,
			exec.NewRawCommand([]string{"true"}),
		)(&def)
		require.Error(t, err, "name %q should be rejected", name)
		require.Empty(t, def.files, "no files should be added when name %q is rejected", name)
	}
}

func TestWithDurableStartupCommand_dispatcherIdempotentAcrossManyFlavors(t *testing.T) {
	// Mix kitless + named, multiple calls each. Exactly one dispatcher.
	def := Definition{}

	require.NoError(t, WithDurableStartupCommand(
		exec.NewRawCommand([]string{"true"}),
	)(&def))
	require.NoError(t, WithDurableStartupCommandsFromDir("pg",
		exec.NewRawCommand([]string{"true"}),
	)(&def))
	require.NoError(t, WithDurableStartupCommand(
		exec.NewRawCommand([]string{"true"}),
	)(&def))
	require.NoError(t, WithDurableStartupCommandsFromDir("redis",
		exec.NewRawCommand([]string{"true"}),
	)(&def))
	require.NoError(t, WithDurableStartupCommandsFromDir("pg",
		exec.NewRawCommand([]string{"true"}),
	)(&def))

	var dispatchers int
	for _, f := range def.files {
		if f.ContainerPath == DurableStartupDispatcherPath {
			dispatchers++
		}
	}
	require.Equal(t, 1, dispatchers)
}

func TestWithDurableStartupCommand_dispatcherSkippedIfPrePresent(t *testing.T) {
	// Consumer manually wired a custom dispatcher at our well-known path.
	// We don't overwrite it — they own the contents.
	custom := File{
		Reader:        strings.NewReader("#!/bin/sh\n# user-controlled\n"),
		ContainerPath: DurableStartupDispatcherPath,
		Mode:          0o755,
	}
	def := Definition{files: []File{custom}}

	require.NoError(t, WithDurableStartupCommand(
		exec.NewRawCommand([]string{"true"}),
	)(&def))

	// Still exactly one entry at the dispatcher path.
	var n int
	for _, f := range def.files {
		if f.ContainerPath == DurableStartupDispatcherPath {
			n++
		}
	}
	require.Equal(t, 1, n)
}

func TestWithDurableStartupCommand_layoutSnapshot(t *testing.T) {
	// Realistic composition scenario: the host registers default commands,
	// then several named namespaces in a known order. We verify the entire
	// container-path layout, end-to-end.
	def := Definition{}

	require.NoError(t, WithDurableStartupCommand(
		exec.NewRawCommand([]string{"echo", "preamble-a"}),
		exec.NewRawCommand([]string{"echo", "preamble-b"}),
	)(&def))
	require.NoError(t, WithDurableStartupCommandsFromDir("pg",
		exec.NewRawCommand([]string{"pg-init"}),
	)(&def))
	require.NoError(t, WithDurableStartupCommandsFromDir("redis",
		exec.NewRawCommand([]string{"redis-init-1"}),
		exec.NewRawCommand([]string{"redis-init-2"}),
	)(&def))
	// Re-entry into pg: appends, doesn't reallocate.
	require.NoError(t, WithDurableStartupCommandsFromDir("pg",
		exec.NewRawCommand([]string{"pg-late"}),
	)(&def))
	// Re-entry into default: appends to 000-default.
	require.NoError(t, WithDurableStartupCommand(
		exec.NewRawCommand([]string{"echo", "preamble-c"}),
	)(&def))

	// The dispatcher is appended immediately after the first option call
	// (it wasn't present yet); subsequent calls just add scripts. The
	// resulting slice order in def.files reflects that.
	expected := []string{
		"/etc/durable-startup.d/000-default/000-cmd.sh",
		"/etc/durable-startup.d/000-default/001-cmd.sh",
		DurableStartupDispatcherPath,
		"/etc/durable-startup.d/001-pg/000-cmd.sh",
		"/etc/durable-startup.d/002-redis/000-cmd.sh",
		"/etc/durable-startup.d/002-redis/001-cmd.sh",
		"/etc/durable-startup.d/001-pg/001-cmd.sh",
		"/etc/durable-startup.d/000-default/002-cmd.sh",
	}

	require.Len(t, def.files, len(expected))
	for i, want := range expected {
		require.Equal(t, want, def.files[i].ContainerPath, "files[%d]", i)
	}
}

func TestWithDurableStartupCommand_endToEndIsDeterministic(t *testing.T) {
	// Two builds of the same option set in the same order produce
	// byte-identical script content at the same paths.
	build := func() []File {
		def := Definition{}
		require.NoError(t, WithDurableStartupCommand(
			exec.NewRawCommand([]string{"echo", "default"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommandsFromDir("pg",
			exec.NewRawCommand([]string{"pg-init"},
				exec.WithWorkingDir("/var/lib/pg"),
				exec.WithEnv([]string{"PGDATA=/var/lib/pg/data"}),
			),
		)(&def))
		return def.files
	}

	a, b := build(), build()
	require.Len(t, a, len(b))
	for i := range a {
		require.Equal(t, a[i].ContainerPath, b[i].ContainerPath, "path[%d]", i)
		require.Equal(t, a[i].Mode, b[i].Mode, "mode[%d]", i)
		require.Equal(t, readFile(t, a[i]), readFile(t, b[i]), "content[%d]", i)
	}
}

func TestWithDurableStartupCommand_coexistsWithWithFiles(t *testing.T) {
	// User-supplied WithFiles entries land alongside ours and don't
	// perturb the dispatcher / numbering. Foreign well-formed namespace
	// directories DO consume an index slot (documented behavior).
	def := Definition{}

	require.NoError(t, WithFiles(
		File{Reader: strings.NewReader("hello"), ContainerPath: "/etc/motd", Mode: 0o644},
	)(&def))
	require.NoError(t, WithDurableStartupCommand(
		exec.NewRawCommand([]string{"true"}),
	)(&def))
	require.NoError(t, WithDurableStartupCommandsFromDir("pg",
		exec.NewRawCommand([]string{"true"}),
	)(&def))

	findFile(t, def.files, "/etc/motd")
	findFile(t, def.files, "/etc/durable-startup.d/000-default/000-cmd.sh")
	findFile(t, def.files, "/etc/durable-startup.d/001-pg/000-cmd.sh")
	findFile(t, def.files, DurableStartupDispatcherPath)
}

func TestWithDurableStartupCommand_fileMetadata(t *testing.T) {
	// Every rendered file is mode 0755 with a non-nil Reader and a
	// container-absolute path. None should rely on HostPath.
	def := Definition{}
	require.NoError(t, WithDurableStartupCommand(
		exec.NewRawCommand([]string{"true"}),
	)(&def))
	require.NoError(t, WithDurableStartupCommandsFromDir("pg",
		exec.NewRawCommand([]string{"true"}),
		exec.NewRawCommand([]string{"true"}),
	)(&def))

	for _, f := range def.files {
		require.Equal(t, durableStartupFileMode, f.Mode, "%s mode", f.ContainerPath)
		require.NotNil(t, f.Reader, "%s reader", f.ContainerPath)
		require.Empty(t, f.HostPath, "%s should not use HostPath", f.ContainerPath)
		require.True(t, strings.HasPrefix(f.ContainerPath, "/"), "%s not absolute", f.ContainerPath)
	}
}

func TestWithDurableStartupCommandsFromDir_loadOrderDeterminesIndex(t *testing.T) {
	// Same set of namespaces, registered in two different orders, produces
	// different filesystem layouts. This is the *intended* contract: the
	// host owns the load order, and the SDK reflects it. The test exists
	// to make sure that contract isn't accidentally broken in either
	// direction.
	build := func(first, second string) []string {
		def := Definition{}
		require.NoError(t, WithDurableStartupCommandsFromDir(first,
			exec.NewRawCommand([]string{"true"}),
		)(&def))
		require.NoError(t, WithDurableStartupCommandsFromDir(second,
			exec.NewRawCommand([]string{"true"}),
		)(&def))
		paths := make([]string, 0, len(def.files))
		for _, f := range def.files {
			paths = append(paths, f.ContainerPath)
		}
		return paths
	}

	pgFirst := build("pg", "redis")
	redisFirst := build("redis", "pg")
	require.Contains(t, pgFirst, "/etc/durable-startup.d/001-pg/000-cmd.sh")
	require.Contains(t, pgFirst, "/etc/durable-startup.d/002-redis/000-cmd.sh")
	require.Contains(t, redisFirst, "/etc/durable-startup.d/001-redis/000-cmd.sh")
	require.Contains(t, redisFirst, "/etc/durable-startup.d/002-pg/000-cmd.sh")
}

func TestErrDurableStartupReservedNamespace_messageMentionsName(t *testing.T) {
	// The sentinel error's message includes the reserved name so users
	// who don't unwrap with errors.Is still get a useful diagnostic.
	require.Contains(t, ErrDurableStartupReservedNamespace.Error(), `"default"`)
}
