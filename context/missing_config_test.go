package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/config"
)

// setupDockerDirWithoutConfigFile creates a ~/.docker directory in a temp home
// so that config.Dir() succeeds, but does NOT create config.json, so that
// config.Load() returns ErrConfigFileNotFound.
func setupDockerDirWithoutConfigFile(tb testing.TB) {
	tb.Helper()
	tmpDir := tb.TempDir()
	tb.Setenv("HOME", tmpDir)
	tb.Setenv("USERPROFILE", tmpDir) // Windows support
	require.NoError(tb, os.MkdirAll(filepath.Join(tmpDir, ".docker"), 0o755))
}

// removeConfigFile deletes the config.json from the current DOCKER_CONFIG dir.
func removeConfigFile(tb testing.TB) {
	tb.Helper()
	dir, err := config.Dir()
	require.NoError(tb, err)
	require.NoError(tb, os.Remove(filepath.Join(dir, config.FileName)))
}

func TestCurrent_ConfigNotFound(t *testing.T) {
	setupDockerDirWithoutConfigFile(t)

	current, err := Current()
	require.NoError(t, err)
	require.Equal(t, DefaultContextName, current)
}

func TestInspect_ConfigNotFound(t *testing.T) {
	SetupTestDockerContexts(t, 1, 3) // creates config.json with currentContext=context1
	removeConfigFile(t)              // simulate a fresh install without config.json

	ctx, err := Inspect("context1")
	require.NoError(t, err)
	require.Equal(t, "context1", ctx.Name)
	require.Equal(t, "tcp://127.0.0.1:1", ctx.Endpoints["docker"].Host)

	require.NotEmpty(t, ctx.encodedName, "encodedName should be set even when config is missing")
	require.False(t, ctx.isCurrent, "isCurrent should be false when config file is missing")
}

func TestStore_Inspect_ConfigNotFound(t *testing.T) {
	SetupTestDockerContexts(t, 1, 3)
	removeConfigFile(t)

	metaDir, err := metaRoot()
	require.NoError(t, err)
	s := &store{root: metaDir}

	ctx, err := s.inspect("context1")
	require.NoError(t, err)
	require.Equal(t, "context1", ctx.Name)
	require.NotEmpty(t, ctx.encodedName)
	require.False(t, ctx.isCurrent)
}

func TestNew_AsCurrent_ConfigNotFound(t *testing.T) {
	setupDockerDirWithoutConfigFile(t)

	ctx, err := New("newctx",
		WithHost("tcp://127.0.0.1:9999"),
		AsCurrent(),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, ctx.Delete()) }()

	require.Equal(t, "newctx", ctx.Name)
	require.False(t, ctx.isCurrent, "isCurrent should be false when config file is missing")

	list, err := List()
	require.NoError(t, err)
	require.Contains(t, list, "newctx")

	current, err := Current()
	require.NoError(t, err)
	require.NotEqual(t, "newctx", current, "current should not be the new context without a config file")
}

func TestDelete_CurrentContext_ConfigNotFound(t *testing.T) {
	SetupTestDockerContexts(t, 1, 3) // creates config.json + contexts

	ctx, err := New("deleteme",
		WithHost("tcp://127.0.0.1:9999"),
		AsCurrent(),
	)
	require.NoError(t, err)
	require.True(t, ctx.isCurrent, "new context should be current")

	removeConfigFile(t)

	require.NoError(t, ctx.Delete(), "delete should not fail when config file is missing")

	_, err = Inspect("deleteme")
	require.ErrorIs(t, err, ErrDockerContextNotFound)
}
