package context

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/context/internal"
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
	t.Run("docker-context/override-host", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 3) // current context is context1
		tt.Setenv(EnvOverrideHost, "tcp://127.0.0.1:123")

		host, err := CurrentDockerHost()
		require.NoError(t, err)
		require.Equal(t, "tcp://127.0.0.1:123", host) // from context1
	})

	t.Run("docker-context/default", func(tt *testing.T) {
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

	t.Run("docker-context/not-found", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 1) // current context is context1

		metaRoot, err := metaRoot()
		require.NoError(t, err)

		host, err := internal.ExtractDockerHost("context-not-found", metaRoot)
		require.Error(t, err)
		require.Empty(t, host)
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

	t.Run("docker-context/not-found", func(tt *testing.T) {
		SetupTestDockerContexts(tt, 1, 1) // current context is context1

		host, err := DockerHostFromContext("context-not-found")
		require.Error(t, err)
		require.Empty(t, host)
	})
}
