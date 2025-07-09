package context

import (
	"fmt"
	"os"

	"github.com/docker/go-sdk/config"
)

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

	ctx, err := Inspect(current)
	if err != nil {
		return "", fmt.Errorf("inspect context: %w", err)
	}

	// Inspect already validates that the docker endpoint is set
	return ctx.Endpoints["docker"].Host, nil
}

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
