package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveRegistryHost(t *testing.T) {
	require.Equal(t, IndexDockerIO, ResolveRegistryHost("index.docker.io"))
	require.Equal(t, IndexDockerIO, ResolveRegistryHost("index.docker.io/v1"))
	require.Equal(t, IndexDockerIO, ResolveRegistryHost("index.docker.io/v1/"))
	require.Equal(t, IndexDockerIO, ResolveRegistryHost("docker.io"))
	require.Equal(t, IndexDockerIO, ResolveRegistryHost("registry-1.docker.io"))
	require.Equal(t, "foobar.com", ResolveRegistryHost("foobar.com"))
	require.Equal(t, "http://foobar.com", ResolveRegistryHost("http://foobar.com"))
	require.Equal(t, "https://foobar.com", ResolveRegistryHost("https://foobar.com"))
	require.Equal(t, "http://foobar.com:8080", ResolveRegistryHost("http://foobar.com:8080"))
	require.Equal(t, "https://foobar.com:8080", ResolveRegistryHost("https://foobar.com:8080"))
}
