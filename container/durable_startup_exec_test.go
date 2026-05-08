//go:build !windows

// Tests in this file actually run the rendered scripts under /bin/sh.
// They catch quoting bugs that `sh -n` (syntactic check only) cannot —
// e.g. an arg that parses fine but ends up expanded or split at runtime.
//
// Constrained to non-Windows because `/bin/sh` and POSIX shell semantics
// aren't generally available on Windows. The runtime LookPath skip
// remains as a guard for unusual non-Windows environments.

package container

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container/exec"
)

// runScriptWithSh writes content to a fresh executable file in t.TempDir(),
// runs it via /bin/sh, and returns combined stdout+stderr. Fails the test
// on non-zero exit. Skips if /bin/sh isn't available.
func runScriptWithSh(t *testing.T, content string) string {
	t.Helper()
	sh, err := osexec.LookPath("/bin/sh")
	if err != nil {
		t.Skipf("/bin/sh unavailable: %v", err)
	}
	p := filepath.Join(t.TempDir(), "script.sh")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o755))
	out, err := osexec.Command(sh, p).CombinedOutput()
	require.NoError(t, err,
		"script failed:\n--script--\n%s\n--output--\n%s",
		content, out,
	)
	return string(out)
}

func TestRenderedFilesParseAsPOSIXShell(t *testing.T) {
	// Catches quoting bugs by handing the rendered files to /bin/sh -n.
	sh, err := osexec.LookPath("/bin/sh")
	if err != nil {
		t.Skipf("/bin/sh unavailable: %v", err)
	}

	def := Definition{}
	require.NoError(t, WithDurableStartupCommand(
		exec.NewRawCommand(
			[]string{"sh", "-c", `echo 'hello world' && rm -rf /; echo "$HOME" $(whoami) ` + "`date`"},
			exec.WithWorkingDir(`/srv with space/'q'`),
			exec.WithEnv([]string{
				"K=v with $vars `cmd` \"q\" 'q'",
				"EMPTY=",
				"URL=http://x?a=1&b=2",
			}),
		),
	)(&def))
	require.NoError(t, WithDurableStartupCommandsFromDir(
		"pg",
		exec.NewRawCommand([]string{"true"}),
		exec.NewRawCommand([]string{"echo", "with\nnewline"}),
		// User wrapping introduces a second layer of quoting — must
		// still parse cleanly.
		exec.NewRawCommand(
			[]string{"sh", "-c", `printf '%s\n' "$1"`, "_", "tricky 'arg'"},
			exec.WithUser("nobody"),
			exec.WithWorkingDir("/tmp"),
			exec.WithEnv([]string{"K='quoted'"}),
		),
	)(&def))

	for _, f := range def.files {
		t.Run(f.ContainerPath, func(t *testing.T) {
			content := readFile(t, f)
			cmd := osexec.Command(sh, "-n")
			cmd.Stdin = strings.NewReader(content)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err,
				"shell parse failed for %s\n--rendered--\n%s\n--shell output--\n%s",
				f.ContainerPath, content, out)
		})
	}
}

func TestRenderedScript_executesArgsByteExact(t *testing.T) {
	// printf '%s\n' prints each remaining arg on its own line, with no
	// expansion or interpretation. It's the cleanest way to verify that
	// argv crosses our shell-quoting layer unmolested.
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"plain", []string{"printf", "%s\n", "hello"}, "hello\n"},
		{"with-spaces", []string{"printf", "%s\n", "hello world"}, "hello world\n"},
		{"dollar-not-expanded", []string{"printf", "%s\n", "$HOME"}, "$HOME\n"},
		{"backticks-not-executed", []string{"printf", "%s\n", "`whoami`"}, "`whoami`\n"},
		{"command-substitution-not-executed", []string{"printf", "%s\n", "$(whoami)"}, "$(whoami)\n"},
		{"single-quote", []string{"printf", "%s\n", "it's"}, "it's\n"},
		{"double-quotes-literal", []string{"printf", "%s\n", `"q"`}, `"q"` + "\n"},
		{"glob-not-expanded", []string{"printf", "%s\n", "*"}, "*\n"},
		{"newline-in-arg", []string{"printf", "%s\n", "a\nb"}, "a\nb\n"},
		{"tab-in-arg", []string{"printf", "%s\n", "a\tb"}, "a\tb\n"},
		{"semicolon-not-terminator", []string{"printf", "%s\n", "a; rm -rf /"}, "a; rm -rf /\n"},
		{"pipe-not-pipe", []string{"printf", "%s\n", "a | wc"}, "a | wc\n"},
		{"empty-arg", []string{"printf", "%s\n", "", "x"}, "\nx\n"},
		{"multi-args", []string{"printf", "%s\n", "one", "two", "three"}, "one\ntwo\nthree\n"},
		{"every-flavor-of-metachar", []string{
			"printf", "%s\n",
			`hello $USER 'q' "dq" ` + "`tick`" + ` $(sub) | & ; > < * ? [a]`,
		}, `hello $USER 'q' "dq" ` + "`tick`" + ` $(sub) | & ; > < * ? [a]` + "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := renderDurableScript(exec.NewRawCommand(tc.args))
			require.NoError(t, err)
			require.Equal(t, tc.want, runScriptWithSh(t, content))
		})
	}
}

func TestRenderedScript_envIsExportedAndQuotedThroughExecution(t *testing.T) {
	// The rendered `export K='v'` lines must survive the whole
	// shell-parse-then-exec pipeline — values with spaces, single quotes,
	// dollar signs, etc., must reach the inner sh -c with byte-exact
	// content.
	content, err := renderDurableScript(exec.NewRawCommand(
		[]string{"sh", "-c", `printf '%s|%s|%s|%s\n' "$K1" "$K2" "$K3" "$K4"`},
		exec.WithEnv([]string{
			"K1=plain",
			"K2=with space",
			"K3=trick'y$dollar",
			"K4=" + "`tick`" + ` "dq"`,
		}),
	))
	require.NoError(t, err)
	out := runScriptWithSh(t, content)
	require.Equal(t,
		"plain|with space|trick'y$dollar|`tick` \"dq\"\n",
		out,
	)
}

func TestRenderedScript_workingDirIsHonored(t *testing.T) {
	dir := t.TempDir()
	content, err := renderDurableScript(exec.NewRawCommand(
		[]string{"pwd"},
		exec.WithWorkingDir(dir),
	))
	require.NoError(t, err)
	out := strings.TrimSpace(runScriptWithSh(t, content))

	// On macOS, /tmp is a symlink to /private/tmp, so pwd may resolve
	// either way depending on shell. Accept either form.
	resolved, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	require.Contains(t, []string{dir, resolved}, out,
		"expected pwd to be %q or %q, got %q", dir, resolved, out)
}

func TestRenderedDispatcher_runsNamespacesInLexicalOrder(t *testing.T) {
	// Lay out a real durable-startup tree on disk and verify the dispatcher
	// walks namespaces (and within each, *-cmd.sh files) in lexical order.
	sandbox := t.TempDir()
	log := filepath.Join(sandbox, "log")

	mkAppendScript := func(tag string) string {
		s, err := renderDurableScript(exec.NewRawCommand(
			[]string{"sh", "-c", `printf '%s\n' "$1" >> "$2"`, "_", tag, log},
		))
		require.NoError(t, err)
		return s
	}

	writeNs := func(ns string, scripts ...string) {
		nsDir := filepath.Join(sandbox, ns)
		require.NoError(t, os.MkdirAll(nsDir, 0o755))
		for i, content := range scripts {
			p := filepath.Join(nsDir, fmt.Sprintf("%03d-cmd.sh", i))
			require.NoError(t, os.WriteFile(p, []byte(content), 0o755))
		}
	}

	writeNs("000-default",
		mkAppendScript("default-0"),
		mkAppendScript("default-1"),
	)
	writeNs("001-pg", mkAppendScript("pg-0"))
	writeNs("002-redis",
		mkAppendScript("redis-0"),
		mkAppendScript("redis-1"),
	)

	runScriptWithSh(t, renderDurableDispatcher(sandbox))

	bs, err := os.ReadFile(log)
	require.NoError(t, err)
	require.Equal(t,
		"default-0\ndefault-1\npg-0\nredis-0\nredis-1\n",
		string(bs),
	)
}

func TestRenderedDispatcher_skipsNonCmdEntriesAndEmptyNamespaces(t *testing.T) {
	// The dispatcher only runs *-cmd.sh files. Non-matching files in a
	// namespace, and empty namespace dirs, are skipped silently.
	sandbox := t.TempDir()
	log := filepath.Join(sandbox, "log")

	mkAppendScript := func(tag string) string {
		s, err := renderDurableScript(exec.NewRawCommand(
			[]string{"sh", "-c", `printf '%s\n' "$1" >> "$2"`, "_", tag, log},
		))
		require.NoError(t, err)
		return s
	}

	require.NoError(t, os.MkdirAll(filepath.Join(sandbox, "000-default"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(sandbox, "000-default", "000-cmd.sh"),
		[]byte(mkAppendScript("hit")), 0o755,
	))
	// Sibling files in the namespace that should NOT execute.
	require.NoError(t, os.WriteFile(
		filepath.Join(sandbox, "000-default", "README.txt"),
		[]byte("ignored"), 0o644,
	))
	// Even a *.sh file that doesn't end in -cmd.sh is skipped.
	require.NoError(t, os.WriteFile(
		filepath.Join(sandbox, "000-default", "helper.sh"),
		[]byte("#!/bin/sh\nprintf 'leak\\n' >> "+log+"\n"), 0o755,
	))
	// An empty namespace dir.
	require.NoError(t, os.MkdirAll(filepath.Join(sandbox, "001-empty"), 0o755))

	runScriptWithSh(t, renderDurableDispatcher(sandbox))

	bs, err := os.ReadFile(log)
	require.NoError(t, err)
	require.Equal(t, "hit\n", string(bs))
}

func TestRenderedDispatcher_handlesMissingRoot(t *testing.T) {
	// Dispatcher with a root that doesn't exist: should exit 0 silently.
	out := runScriptWithSh(t, renderDurableDispatcher("/path/that/does/not/exist/0987"))
	require.Empty(t, out)
}

func TestRenderedDispatcher_propagatesScriptFailure(t *testing.T) {
	// `set -e` in the dispatcher means a failing script aborts the run.
	// The script after the failing one should NOT execute.
	sandbox := t.TempDir()
	log := filepath.Join(sandbox, "log")

	mkAppendScript := func(tag string) string {
		s, err := renderDurableScript(exec.NewRawCommand(
			[]string{"sh", "-c", `printf '%s\n' "$1" >> "$2"`, "_", tag, log},
		))
		require.NoError(t, err)
		return s
	}
	mkFailScript := func() string {
		s, err := renderDurableScript(exec.NewRawCommand([]string{"false"}))
		require.NoError(t, err)
		return s
	}

	require.NoError(t, os.MkdirAll(filepath.Join(sandbox, "000-default"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(sandbox, "000-default", "000-cmd.sh"),
		[]byte(mkAppendScript("ran")), 0o755,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(sandbox, "000-default", "001-cmd.sh"),
		[]byte(mkFailScript()), 0o755,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(sandbox, "000-default", "002-cmd.sh"),
		[]byte(mkAppendScript("should-not-run")), 0o755,
	))

	sh, err := osexec.LookPath("/bin/sh")
	if err != nil {
		t.Skipf("/bin/sh unavailable: %v", err)
	}
	p := filepath.Join(t.TempDir(), "dispatcher.sh")
	require.NoError(t, os.WriteFile(p, []byte(renderDurableDispatcher(sandbox)), 0o755))
	err = osexec.Command(sh, p).Run()
	require.Error(t, err, "dispatcher should propagate the failing script's exit code")

	bs, err := os.ReadFile(log)
	require.NoError(t, err)
	require.Equal(t, "ran\n", string(bs),
		"only the script before the failure should have appended")
}
