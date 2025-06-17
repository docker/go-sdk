package context

// The code in this file has been extracted from https://github.com/docker/cli,
// more especifically from https://github.com/docker/cli/blob/master/cli/context/store/metadatastore.go
// with the goal of not consuming the CLI package and all its dependencies.

import (
	"fmt"
	"os"
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

	// ErrDockerHostNotSet is the error returned when the Docker host is not set in the Docker context
	ErrDockerHostNotSet = internal.ErrDockerHostNotSet

	// ErrDockerContextNotFound is the error returned when the Docker context is not found.
	ErrDockerContextNotFound = internal.ErrDockerContextNotFound
)

// getContextFromEnv returns the context name from the environment variables.
func getContextFromEnv() string {
	if os.Getenv(EnvOverrideHost) != "" {
		return DefaultContextName
	}

	if ctxName := os.Getenv(EnvOverrideContext); ctxName != "" {
		return ctxName
	}

	return ""
}

// Current returns the current context name, based on
// environment variables and the cli configuration file. It does not
// validate if the given context exists or if it's valid.
//
// If the current context is not found, it returns the default context name.
func Current() (string, error) {
	// Check env vars first (clearer precedence)
	if ctx := getContextFromEnv(); ctx != "" {
		return ctx, nil
	}

	// Then check config
	cfg, err := config.Load()
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultContextName, nil
		}
		return "", fmt.Errorf("load docker config: %w", err)
	}

	if cfg.CurrentContext != "" {
		return cfg.CurrentContext, nil
	}

	return DefaultContextName, nil
}

// CurrentDockerHost returns the Docker host from the current Docker context.
// For that, it traverses the directory structure of the Docker configuration directory,
// looking for the current context and its Docker endpoint.
//
// If the current context is the default context, it returns the value of the
// DOCKER_HOST environment variable.
func CurrentDockerHost() (string, error) {
	current, err := Current()
	if err != nil {
		return "", fmt.Errorf("current context: %w", err)
	}

	if current == DefaultContextName {
		dockerHost := os.Getenv(EnvOverrideHost)
		if dockerHost != "" {
			return dockerHost, nil
		}

		return DefaultDockerHost, nil
	}

	metaRoot, err := metaRoot()
	if err != nil {
		return "", fmt.Errorf("meta root: %w", err)
	}

	return internal.ExtractDockerHost(current, metaRoot)
}

// DockerHostFromContext returns the Docker host from the given context.
func DockerHostFromContext(ctx string) (string, error) {
	metaRoot, err := metaRoot()
	if err != nil {
		return "", fmt.Errorf("meta root: %w", err)
	}

	return internal.ExtractDockerHost(ctx, metaRoot)
}

// metaRoot returns the root directory of the Docker context metadata.
func metaRoot() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", fmt.Errorf("docker config dir: %w", err)
	}

	return filepath.Join(dir, contextsDir, metadataDir), nil
}
