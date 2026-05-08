package container

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/docker/go-sdk/container/exec"
)

// DurableStartupDir is the directory inside the container under which all
// durable startup scripts live. Each consumer-supplied namespace gets a
// numbered subdirectory; the dispatcher is rendered at this path's root.
//
// The layout is:
//
//	/etc/durable-startup.d/
//	  000-default/         from WithDurableStartupCommand
//	    000-cmd.sh
//	    001-cmd.sh
//	  001-<name>/          from WithDurableStartupCommandsFromDir, in
//	    000-cmd.sh         registration order
//	  002-<name>/
//	    000-cmd.sh
//	  run.sh               single dispatcher
const DurableStartupDir = "/etc/durable-startup.d"

// defaultDurableNamespace is the reserved namespace that always receives
// index 0. Commands registered via [WithDurableStartupCommand] (no name)
// land here, so they execute before any consumer-defined namespace.
const defaultDurableNamespace = "default"

// durableStartupDispatcherName is the basename of the dispatcher script.
const durableStartupDispatcherName = "run.sh"

// durableStartupFileMode is the mode used for rendered script files.
// They must be executable so the dispatcher can invoke them directly.
const durableStartupFileMode int64 = 0o755

// DurableStartupDispatcherPath is the absolute path of the dispatcher
// rendered alongside the durable startup scripts. Consumers wire it up
// however they need: as a regular [WithStartupCommand] for first-create
// coverage, or invoked directly from a reconnect path.
const DurableStartupDispatcherPath = DurableStartupDir + "/" + durableStartupDispatcherName

// durableNamespaceNameRe restricts namespace names to a safe, lexically
// well-behaved subset. Lowercase letters, digits, '-', '_'. Must start
// with an alphanumeric. This avoids slashes/dots in path components and
// keeps lexical sort intuitive across locales.
var durableNamespaceNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// durableUserRe is the set of values [exec.WithUser] is allowed to take
// in a durable startup command: a POSIX-shaped login name. The renderer
// switches user via `su`, which only resolves login names — not Docker's
// numeric UIDs or `uid:gid` / `user:group` specs. Reject those forms at
// render time so the consumer sees a clear definition-time error
// instead of a silent dispatcher failure at runtime.
var durableUserRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// ErrDurableStartupReservedNamespace is returned when a consumer attempts
// to register the reserved "default" namespace via
// [WithDurableStartupCommandsFromDir]. Use [WithDurableStartupCommand]
// instead — it writes into the default namespace by definition.
var ErrDurableStartupReservedNamespace = fmt.Errorf("durable startup namespace %q is reserved", defaultDurableNamespace)

// WithDurableStartupCommand registers startup commands persisted to script
// files inside the container under the reserved "default" namespace.
// Unlike [WithStartupCommand], which fires the in-process PostStarts hook
// only on the [Container] that created the sandbox, this variant survives
// daemon restarts and Docker engine container resurrection — the script
// files are part of the container's own filesystem state.
//
// The dispatcher that walks the directory and runs the scripts is rendered
// alongside, but invocation is the consumer's responsibility (typically:
// register the dispatcher as a regular [WithStartupCommand] for
// first-create coverage, and invoke it manually from any reconnect path
// the consumer owns — e.g. CLI's "run" or "exec" entry points).
//
// Commands registered here always execute before any namespace registered
// via [WithDurableStartupCommandsFromDir], because the default namespace
// is reserved at index 000.
//
// Each [Executable]'s [exec.WithWorkingDir], [exec.WithEnv], and
// [exec.WithUser] options are translated into the rendered script:
// WithEnv as `export` lines, WithWorkingDir as a `cd`, and WithUser by
// wrapping the body in `su -s /bin/sh -c '<body>' '<user>'`. The user
// switch fails loud (set -e propagates) when the user does not exist or
// `su` is unavailable. [exec.WithTTY] and other process options are
// ignored: the script does not go through a Docker exec.
//
// [exec.WithUser] only accepts a login name in the durable variant.
// Docker's `uid:gid`, `user:group`, and bare-UID forms are rejected at
// definition time because POSIX `su` cannot resolve them. Callers that
// need full Docker user-spec semantics should use [WithStartupCommand]
// instead.
//
// Not safe to call concurrently on the same [Definition]: collect the
// option values upstream and apply them serially.
func WithDurableStartupCommand(execs ...Executable) CustomizeDefinitionOption {
	return func(def *Definition) error {
		return appendDurableScripts(def, defaultDurableNamespace, execs)
	}
}

// WithDurableStartupCommandsFromDir registers startup commands under a
// named namespace. The first time a name is seen, it is allocated the
// next sequential index (after the reserved default namespace at 000).
// Subsequent calls with the same name append to the same namespace dir.
//
// Namespaces execute in registration order (lexical-by-prefix in the
// dispatcher); within a namespace, commands execute in the order they
// were registered across all calls.
//
// dirName must match `^[a-z0-9][a-z0-9_-]*$` and must not be "default"
// (which is reserved for [WithDurableStartupCommand]).
//
// See [WithDurableStartupCommand] for the broader contract around
// persistence, dispatcher invocation, and translated process options
// (including [exec.WithUser] / [exec.WithEnv] / [exec.WithWorkingDir]).
//
// Not safe to call concurrently on the same [Definition]: collect the
// option values upstream and apply them serially.
func WithDurableStartupCommandsFromDir(dirName string, execs ...Executable) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if dirName == defaultDurableNamespace {
			return ErrDurableStartupReservedNamespace
		}
		if !durableNamespaceNameRe.MatchString(dirName) {
			return fmt.Errorf("invalid durable startup namespace %q: must match %s", dirName, durableNamespaceNameRe)
		}
		return appendDurableScripts(def, dirName, execs)
	}
}

// appendDurableScripts is the shared core. It resolves (or allocates) the
// namespace subdirectory for name, renders each exec into the next
// MMM-cmd.sh slot in that subdir, and ensures the dispatcher is rendered
// exactly once at the root.
//
// Rendering is transactional: every exec is rendered before any file is
// appended to def.files. If any exec fails to render, the Definition is
// left untouched.
func appendDurableScripts(def *Definition, name string, execs []Executable) error {
	if len(execs) == 0 {
		return errors.New("at least one executable is required")
	}

	contents := make([]string, len(execs))
	for i, e := range execs {
		content, err := renderDurableScript(e)
		if err != nil {
			return fmt.Errorf("render durable startup command %d in namespace %q: %w", i, name, err)
		}
		contents[i] = content
	}

	nsDir, dispatcherPresent := resolveDurableNamespaceDir(def.files, name)
	cmdStart := nextDurableCmdIndex(def.files, nsDir)

	for i, content := range contents {
		def.files = append(def.files, File{
			Reader:        bytes.NewReader([]byte(content)),
			ContainerPath: path.Join(nsDir, fmt.Sprintf("%03d-cmd.sh", cmdStart+i)),
			Mode:          durableStartupFileMode,
		})
	}

	if !dispatcherPresent {
		def.files = append(def.files, File{
			Reader:        bytes.NewReader([]byte(renderDurableDispatcher(DurableStartupDir))),
			ContainerPath: DurableStartupDispatcherPath,
			Mode:          durableStartupFileMode,
		})
	}

	return nil
}

// resolveDurableNamespaceDir returns the absolute namespace subdirectory
// for name, allocating a fresh NNN index the first time name is seen. It
// also reports whether the dispatcher script is already present in
// def.files (so the caller renders it exactly once).
//
// The reserved "default" namespace is always indexed 000 regardless of
// when it is first registered. Other namespaces receive (count of
// already-seen non-default namespaces) + 1 as their index.
//
// State is derived from def.files alone — no extra field on Definition.
func resolveDurableNamespaceDir(files []File, name string) (string, bool) {
	prefix := DurableStartupDir + "/"
	dispatcherPresent := false

	// Walk files once: identify the dispatcher, find an existing dir for
	// name, and count distinct non-default namespaces seen so far.
	type seen struct{ idx int }
	knownNames := make(map[string]seen)
	for _, f := range files {
		if f.ContainerPath == DurableStartupDispatcherPath {
			dispatcherPresent = true
			continue
		}
		if !strings.HasPrefix(f.ContainerPath, prefix) {
			continue
		}
		rel := strings.TrimPrefix(f.ContainerPath, prefix)
		slash := strings.IndexByte(rel, '/')
		if slash <= 0 {
			continue // file directly in DurableStartupDir, not in a namespace
		}
		sub := rel[:slash] // "NNN-name"
		dash := strings.IndexByte(sub, '-')
		if dash <= 0 || dash == len(sub)-1 {
			continue // not our layout
		}
		nsName := sub[dash+1:]
		if _, ok := knownNames[nsName]; ok {
			continue
		}
		var idx int
		if _, err := fmt.Sscanf(sub[:dash], "%d", &idx); err != nil {
			continue
		}
		knownNames[nsName] = seen{idx: idx}
	}

	// If name is already registered, reuse its directory.
	if existing, ok := knownNames[name]; ok {
		return path.Join(DurableStartupDir, fmt.Sprintf("%03d-%s", existing.idx, name)), dispatcherPresent
	}

	// Default is always reserved at 000, even if it hasn't been used yet.
	if name == defaultDurableNamespace {
		return path.Join(DurableStartupDir, "000-"+defaultDurableNamespace), dispatcherPresent
	}

	// Otherwise allocate the next available index after default + the
	// non-default namespaces already seen.
	nonDefault := 0
	for n := range knownNames {
		if n != defaultDurableNamespace {
			nonDefault++
		}
	}
	return path.Join(DurableStartupDir, fmt.Sprintf("%03d-%s", nonDefault+1, name)), dispatcherPresent
}

// nextDurableCmdIndex returns the next MMM index to use for a new
// command file inside nsDir, by counting existing *-cmd.sh siblings.
func nextDurableCmdIndex(files []File, nsDir string) int {
	n := 0
	for _, f := range files {
		if path.Dir(f.ContainerPath) == nsDir && strings.HasSuffix(f.ContainerPath, "-cmd.sh") {
			n++
		}
	}
	return n
}

// renderDurableScript builds the contents of a single durable startup
// script for the given [Executable]. The resulting POSIX shell script
// honors [exec.WithEnv] (as `export` lines), [exec.WithWorkingDir] (as
// a `cd`), and [exec.WithUser] (by wrapping the body in
// `su -s /bin/sh -c '<body>' '<user>'`). Other process options are
// not translated.
//
// When [exec.WithUser] is set, the body — env exports, cd, exec — runs
// inside the inner shell launched by `su`, with its own `set -e` so
// failures (cd into a missing dir, missing user, missing su binary) all
// propagate via the exit code. The dispatcher's own `set -e` then
// surfaces the failure to the consumer.
func renderDurableScript(e Executable) (string, error) {
	if e == nil {
		return "", errors.New("executable is nil")
	}
	cmd := e.AsCommand()
	if len(cmd) == 0 {
		return "", errors.New("executable produced empty command")
	}

	var po exec.ProcessOptions
	for _, opt := range e.Options() {
		if opt == nil {
			continue
		}
		opt.Apply(&po)
	}

	// `su` only resolves login names. Docker's full user spec
	// (`uid:gid`, `user:group`, bare UIDs without a /etc/passwd entry)
	// would silently fail at runtime when the dispatcher tried to switch
	// users. Reject those forms here so the consumer gets a clear
	// definition-time error and a pointer to the alternative.
	if u := po.ExecConfig.User; u != "" && !durableUserRe.MatchString(u) {
		return "", fmt.Errorf(
			"durable startup WithUser %q: only login names are supported "+
				"(uid:gid / user:group / bare-UID forms are not, because the "+
				"rendered script switches user via `su`); use a login name, "+
				"or fall back to WithStartupCommand for full Docker user-spec semantics",
			u,
		)
	}

	// Build the inner body: env exports, cd, exec. This is shared between
	// the kitless path (inlined) and the WithUser path (passed as the -c
	// arg to `su`, which runs it in a fresh shell).
	var body strings.Builder
	for _, env := range po.ExecConfig.Env {
		eq := strings.IndexByte(env, '=')
		if eq <= 0 {
			continue
		}
		fmt.Fprintf(&body, "export %s=%s\n", env[:eq], shellSingleQuote(env[eq+1:]))
	}
	if po.ExecConfig.WorkingDir != "" {
		fmt.Fprintf(&body, "cd %s\n", shellSingleQuote(po.ExecConfig.WorkingDir))
	}
	body.WriteString("exec")
	for _, arg := range cmd {
		body.WriteByte(' ')
		body.WriteString(shellSingleQuote(arg))
	}
	body.WriteByte('\n')

	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("set -e\n")

	if po.ExecConfig.User != "" {
		// `su` spawns a fresh shell that does NOT inherit `set -e` from
		// the outer script, so prepend `set -e` to the inner body. Order
		// is options-first then USER (POSIX getopt-strict shells stop at
		// the first non-option arg).
		inner := "set -e\n" + body.String()
		fmt.Fprintf(&b, "exec su -s /bin/sh -c %s %s\n",
			shellSingleQuote(inner),
			shellSingleQuote(po.ExecConfig.User),
		)
	} else {
		b.WriteString(body.String())
	}
	return b.String(), nil
}

// renderDurableDispatcher builds the dispatcher that walks each namespace
// subdirectory of root in lexical order and invokes its *-cmd.sh files,
// also in lexical order. LC_ALL=C is pinned so byte-sort is independent
// of the runtime locale.
func renderDurableDispatcher(root string) string {
	return fmt.Sprintf(`#!/bin/sh
set -e
LC_ALL=C
export LC_ALL
ROOT=%s
[ -d "$ROOT" ] || exit 0
for ns in "$ROOT"/*/; do
	[ -d "$ns" ] || continue
	for f in "$ns"*-cmd.sh; do
		[ -f "$f" ] || continue
		"$f"
	done
done
`, shellSingleQuote(root))
}

// shellSingleQuote returns s wrapped in POSIX single quotes, with any
// embedded single quotes escaped via the standard '\” idiom. The result
// is always a single shell word safe for substitution into a script.
func shellSingleQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
