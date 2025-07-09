package context

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/config"
)

// SetupTestDockerContexts creates a temporary directory structure for testing the Docker context functions.
// It creates the following structure, where $i is the index of the context, starting from 1:
// - $HOME/.docker
//   - config.json
//   - contexts
//   - meta
//   - context$i
//   - meta.json
//
// The config.json file contains the current context, and the meta.json files contain the metadata for each context.
// It generates the specified number of contexts, setting the current context to the one specified by currentContextIndex.
// The docker host for each context is "tcp://127.0.0.1:$i".
// Finally it always adds a context with an empty host, to validate the behavior when the host is not set,
// and a context with a custom endpoint, to validate the behavior when the endpoint is not the default "docker".
// This empty context can be used setting the currentContextIndex to a number greater than contextsCount.
func SetupTestDockerContexts(tb testing.TB, currentContextIndex int, contextsCount int) {
	tb.Helper()

	tmpDir := tb.TempDir()
	tb.Setenv("HOME", tmpDir)
	tb.Setenv("USERPROFILE", tmpDir) // Windows support

	tempMkdirAll(tb, filepath.Join(tmpDir, ".docker"))

	configDir, err := config.Dir()
	require.NoError(tb, err)

	configJSON := filepath.Join(configDir, config.FileName)

	const baseContextName = "context"

	// default config.json with no current context
	configBytes := `{"currentContext": ""}`

	if currentContextIndex <= contextsCount {
		configBytes = fmt.Sprintf(`{
	"currentContext": "%s%d"
}`, baseContextName, currentContextIndex)
	}

	err = os.WriteFile(configJSON, []byte(configBytes), 0o644)
	require.NoError(tb, err)

	metaDir, err := metaRoot()
	require.NoError(tb, err)

	tempMkdirAll(tb, metaDir)

	// first index is 1
	for i := 1; i <= contextsCount; i++ {
		createDockerContext(tb, metaDir, baseContextName, i, fmt.Sprintf("tcp://127.0.0.1:%d", i))
	}

	// add a context with no host
	createDockerContext(tb, metaDir, baseContextName, contextsCount+1, "")

	// add a context that does not have a docker endpoint
	createDockerContextWithCustomEndpoint(tb, metaDir, baseContextName, contextsCount+2, "foo", "")
}

// createDockerContext creates a Docker context with the specified name and host
func createDockerContext(tb testing.TB, metaDir, baseContextName string, index int, host string) {
	tb.Helper()

	createDockerContextWithCustomEndpoint(tb, metaDir, baseContextName, index, "docker", host)
}

// createDockerContextWithoutDockerEndpoint creates a Docker context with the specified name and no docker endpoint
func createDockerContextWithCustomEndpoint(tb testing.TB, metaDir, baseContextName string, index int, endpointName string, host string) {
	tb.Helper()

	contextDir := filepath.Join(metaDir, fmt.Sprintf("context%d", index))
	tempMkdirAll(tb, contextDir)

	context := fmt.Sprintf(`{"Name":"%s%d","Metadata":{"Description":"Docker Go SDK %d"},"Endpoints":{"%s":{"Host":"%s","SkipTLSVerify":false}}}`,
		baseContextName, index, index, endpointName, host)
	err := os.WriteFile(filepath.Join(contextDir, "meta.json"), []byte(context), 0o644)
	require.NoError(tb, err)
}

func tempMkdirAll(tb testing.TB, dir string) {
	tb.Helper()

	err := os.MkdirAll(dir, 0o755)
	require.NoError(tb, err)
}
