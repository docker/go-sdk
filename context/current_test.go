package context

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		path, err := parseURL(DefaultDockerHost)
		require.NoError(t, err)
		require.Equal(t, DefaultDockerHost, path)
	})

	t.Run("success/tcp", func(t *testing.T) {
		path, err := parseURL("tcp://localhost:2375")
		require.NoError(t, err)
		require.Equal(t, "tcp://localhost:2375", path)
	})

	t.Run("error/invalid-schema", func(t *testing.T) {
		_, err := parseURL("http://localhost:2375")
		require.Error(t, err)
	})

	t.Run("error/invalid-url", func(t *testing.T) {
		_, err := parseURL("~wrong~://**~invalid url~**")
		require.Error(t, err)
	})
}
