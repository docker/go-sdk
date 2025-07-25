package context

// The code in this file has been extracted from https://github.com/docker/cli,
// more especifically from https://github.com/docker/cli/blob/master/cli/context/store/metadatastore.go
// with the goal of not consuming the CLI package and all its dependencies.

import (
	"fmt"
	"path/filepath"

	"github.com/docker/go-sdk/config"
)

const (
	// DefaultContextName is the name reserved for the default context (config & env based)
	DefaultContextName = "default"

	// EnvOverrideContext is the name of the environment variable that can be
	// used to override the context to use. If set, it overrides the context
	// that's set in the CLI's configuration file, but takes no effect if the
	// "DOCKER_HOST" env-var is set (which takes precedence.
	EnvOverrideContext = "DOCKER_CONTEXT"

	// EnvOverrideHost is the name of the environment variable that can be used
	// to override the default host to connect to (DefaultDockerHost).
	//
	// This env-var is read by FromEnv and WithHostFromEnv and when set to a
	// non-empty value, takes precedence over the default host (which is platform
	// specific), or any host already set.
	EnvOverrideHost = "DOCKER_HOST"

	// contextsDir is the name of the directory containing the contexts
	contextsDir = "contexts"

	// metadataDir is the name of the directory containing the metadata
	metadataDir = "meta"

	// metaFile is the name of the file containing the context metadata
	metaFile = "meta.json"
)

var (
	// DefaultDockerHost is the default host to connect to the Docker socket.
	// The actual value is platform-specific and defined in host_unix.go and host_windows.go.
	DefaultDockerHost = ""

	// DefaultSchema is the default schema to use for the Docker host.
	// The actual value is platform-specific and defined in host_unix.go and host_windows.go.
	DefaultSchema = ""

	// TCPSchema is the schema to use for TCP connections.
	TCPSchema = "tcp://"
)

// DockerHostFromContext returns the Docker host from the given context.
func DockerHostFromContext(ctxName string) (string, error) {
	ctx, err := Inspect(ctxName)
	if err != nil {
		return "", fmt.Errorf("inspect context: %w", err)
	}

	// Inspect already validates that the docker endpoint is set
	return ctx.Endpoints["docker"].Host, nil
}

// Inspect returns the given context.
// It returns an error if the context is not found or if the docker endpoint is not set.
func Inspect(ctxName string) (Context, error) {
	metaRoot, err := metaRoot()
	if err != nil {
		return Context{}, fmt.Errorf("meta root: %w", err)
	}

	s := &store{root: metaRoot}

	return s.inspect(ctxName)
}

// List returns the list of contexts available in the Docker configuration.
func List() ([]string, error) {
	metaRoot, err := metaRoot()
	if err != nil {
		return nil, fmt.Errorf("meta root: %w", err)
	}

	s := &store{root: metaRoot}

	contexts, err := s.list()
	if err != nil {
		return nil, fmt.Errorf("list contexts: %w", err)
	}

	names := make([]string, len(contexts))
	for i, ctx := range contexts {
		names[i] = ctx.Name
	}
	return names, nil
}

// metaRoot returns the root directory of the Docker context metadata.
func metaRoot() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", fmt.Errorf("docker config dir: %w", err)
	}

	return filepath.Join(dir, contextsDir, metadataDir), nil
}
