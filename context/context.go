package context

// The code in this file has been extracted from https://github.com/docker/cli,
// more especifically from https://github.com/docker/cli/blob/master/cli/context/store/metadatastore.go
// with the goal of not consuming the CLI package and all its dependencies.

import (
	"fmt"
	"path/filepath"

	"github.com/docker/go-sdk/config"
	"github.com/docker/go-sdk/context/internal"
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
)

var (
	// DefaultDockerHost is the default host to connect to the Docker socket.
	// The actual value is platform-specific and defined in host_linux.go and host_windows.go.
	DefaultDockerHost = ""
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

// Inspect returns the description of the given context.
func Inspect(ctxName string) (Context, error) {
	metaRoot, err := metaRoot()
	if err != nil {
		return Context{}, fmt.Errorf("meta root: %w", err)
	}

	return internal.Inspect(ctxName, metaRoot)
}

// List returns the list of contexts available in the Docker configuration.
func List() ([]string, error) {
	metaRoot, err := metaRoot()
	if err != nil {
		return nil, fmt.Errorf("meta root: %w", err)
	}

	return internal.List(metaRoot)
}

// metaRoot returns the root directory of the Docker context metadata.
func metaRoot() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", fmt.Errorf("docker config dir: %w", err)
	}

	return filepath.Join(dir, contextsDir, metadataDir), nil
}
