package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrent(t *testing.T) {
	t.Run("current/1", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 3) // current context is context1

		current, err := Current()
		require.NoError(t, err)
		require.Equal(t, "context1", current)
	})

	t.Run("current/auth-error", func(tt *testing.T) {
		tt.Setenv("DOCKER_AUTH_CONFIG", "invalid-auth-config")

		current, err := Current()
		require.Error(t, err)
		require.Empty(t, current)
	})

	t.Run("current/override-host", func(tt *testing.T) {
		tt.Setenv(EnvOverrideHost, "tcp://127.0.0.1:2")

		current, err := Current()
		require.NoError(t, err)
		require.Equal(t, DefaultContextName, current)
	})

	t.Run("current/override-context", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 3)         // current context is context1
		tt.Setenv(EnvOverrideContext, "context2") // override the current context

		current, err := Current()
		require.NoError(t, err)
		require.Equal(t, "context2", current)
	})

	t.Run("current/empty-context", func(tt *testing.T) {
		contextCount := 3
		SetupTestDockerContexts(tt, contextCount+1, contextCount) // current context is the empty one

		current, err := Current()
		require.NoError(t, err)
		require.Equal(t, DefaultContextName, current)
	})
}

func TestCurrentDockerHost(t *testing.T) {
	t.Run("override-host", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 3) // current context is context1
		tt.Setenv(EnvOverrideHost, "tcp://127.0.0.1:123")

		host, err := CurrentDockerHost()
		require.NoError(t, err)
		require.Equal(t, "tcp://127.0.0.1:123", host) // from context1
	})

	t.Run("default", func(tt *testing.T) {
		tt.Setenv(EnvOverrideContext, DefaultContextName)

		host, err := CurrentDockerHost()
		require.NoError(t, err)
		require.Equal(t, DefaultDockerHost, host)
	})

	t.Run("docker-context/1", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 3) // current context is context1

		host, err := CurrentDockerHost()
		require.NoError(t, err)
		require.Equal(t, "tcp://127.0.0.1:1", host) // from context1
	})

	t.Run("docker-context/2", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 2, 3) // current context is context2

		host, err := CurrentDockerHost()
		require.NoError(t, err)
		require.Equal(t, "tcp://127.0.0.1:2", host) // from context2
	})

	t.Run("rootless", func(tt *testing.T) {
		tmpDir := tt.TempDir()
		t.Setenv("XDG_RUNTIME_DIR", tmpDir)

		err := os.WriteFile(filepath.Join(tmpDir, "docker.sock"), []byte("synthetic docker socket"), 0o755)
		require.NoError(tt, err)

		host, err := CurrentDockerHost()
		require.NoError(tt, err)
		require.Equal(tt, DefaultSchema+filepath.Join(tmpDir, "docker.sock"), host)
	})
}

func TestDockerHostFromContext(t *testing.T) {
	t.Run("docker-context/override-host", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 3) // current context is context1
		tt.Setenv(EnvOverrideHost, "tcp://127.0.0.1:123")

		host, err := DockerHostFromContext("context1")
		require.NoError(t, err)
		require.Equal(t, "tcp://127.0.0.1:1", host) // from context1
	})

	t.Run("docker-context/default", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 3) // current context is context1
		tt.Setenv(EnvOverrideContext, DefaultContextName)

		host, err := DockerHostFromContext("context1")
		require.NoError(t, err)
		require.Equal(t, "tcp://127.0.0.1:1", host)
	})

	t.Run("docker-context/1", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 3) // current context is context1

		host, err := DockerHostFromContext("context1")
		require.NoError(t, err)
		require.Equal(t, "tcp://127.0.0.1:1", host) // from context1
	})

	t.Run("docker-context/2", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 2, 3) // current context is context2

		host, err := DockerHostFromContext("context2")
		require.NoError(t, err)
		require.Equal(t, "tcp://127.0.0.1:2", host) // from context2
	})

	t.Run("docker-context/4-no-host", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 4, 3) // current context is context4

		host, err := DockerHostFromContext("context4")
		require.ErrorIs(t, err, ErrDockerHostNotSet)
		require.Empty(t, host)
	})

	t.Run("docker-context/not-found", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 1) // current context is context1

		host, err := DockerHostFromContext("context-not-found")
		require.Error(t, err)
		require.Empty(t, host)
	})

	t.Run("docker-context/5-no-host", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 5, 3) // current context is context5

		host, err := DockerHostFromContext("context5")
		require.ErrorIs(t, err, ErrDockerHostNotSet)
		require.Empty(t, host)
	})
}

func TestInspect(t *testing.T) {
	SetupTestDockerContexts(t, 1, 3) // current context is context1

	t.Run("inspect/1", func(t *testing.T) {
		c, err := Inspect("context1")
		require.NoError(t, err)
		require.Equal(t, "Docker Go SDK 1", c.Metadata.Description)
	})

	t.Run("inspect/2", func(t *testing.T) {
		c, err := Inspect("context2")
		require.NoError(t, err)
		require.Equal(t, "Docker Go SDK 2", c.Metadata.Description)
	})

	t.Run("inspect/not-found", func(t *testing.T) {
		c, err := Inspect("context-not-found")
		require.ErrorIs(t, err, ErrDockerContextNotFound)
		require.Empty(t, c)
	})

	t.Run("inspect/5-no-docker-endpoint", func(t *testing.T) {
		c, err := Inspect("context5")
		require.ErrorIs(t, err, ErrDockerHostNotSet)
		require.Empty(t, c)
	})
}

func TestList(t *testing.T) {
	t.Run("list/1", func(t *testing.T) {
		SetupTestDockerContexts(t, 1, 3) // current context is context1

		contexts, err := List()
		require.NoError(t, err)
		require.Equal(t, []string{"context1", "context2", "context3", "context4", "context5"}, contexts)
	})

	t.Run("list/empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		t.Setenv("USERPROFILE", tmpDir) // Windows support

		tempMkdirAll(t, filepath.Join(tmpDir, ".docker"))

		contexts, err := List()
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Empty(t, contexts)
	})
}
