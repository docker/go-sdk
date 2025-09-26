package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveRegistryHost(t *testing.T) {
	require.Equal(t, IndexDockerIO, resolveRegistryHost("index.docker.io"))
	require.Equal(t, IndexDockerIO, resolveRegistryHost("docker.io"))
	require.Equal(t, IndexDockerIO, resolveRegistryHost("registry-1.docker.io"))
	require.Equal(t, "foobar.com", resolveRegistryHost("foobar.com"))
	require.Equal(t, "foobar.com", resolveRegistryHost("http://foobar.com"))
	require.Equal(t, "foobar.com", resolveRegistryHost("https://foobar.com"))
	require.Equal(t, "foobar.com:8080", resolveRegistryHost("http://foobar.com:8080"))
	require.Equal(t, "foobar.com:8080", resolveRegistryHost("https://foobar.com:8080"))
	// Test the specific case mentioned in the problem statement
	require.Equal(t, IndexDockerIO, resolveRegistryHost("https://index.docker.io/v1/"))
}
