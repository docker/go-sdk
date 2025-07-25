package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootlessSocketPathFromEnv(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_RUNTIME_DIR", tmpDir)

		err := os.WriteFile(filepath.Join(tmpDir, "docker.sock"), []byte("synthetic docker socket"), 0o755)
		require.NoError(t, err)

		path, err := rootlessSocketPathFromEnv()
		require.NoError(t, err)
		require.Equal(t, DefaultSchema+filepath.Join(tmpDir, "docker.sock"), path)
	})

	t.Run("env-var-not-set", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", "")
		path, err := rootlessSocketPathFromEnv()
		require.ErrorIs(t, err, ErrXDGRuntimeDirNotSet)
		require.Empty(t, path)
	})

	t.Run("docker-socket-not-found", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_RUNTIME_DIR", tmpDir)

		path, err := rootlessSocketPathFromEnv()
		require.ErrorIs(t, err, ErrRootlessDockerNotFoundXDGRuntimeDir)
		require.Empty(t, path)
	})
}
