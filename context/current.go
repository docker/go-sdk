package context

import (
	"fmt"
	"net/url"
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
// If the Rootless Docker socket is found, using the XDG_RUNTIME_DIR environment variable,
// it returns the path to the socket.
//
// If the current context is the default context, it returns the value of the
// DOCKER_HOST environment variable.
//
// It validates that the Docker host is a valid URL and that the schema is
// either unix, npipe (on Windows) or tcp.
func CurrentDockerHost() (string, error) {
	rootlessSocketPath, err := rootlessSocketPathFromEnv()
	if err == nil {
		return parseURL(rootlessSocketPath)
	}

	current, err := Current()
	if err != nil {
		return "", fmt.Errorf("current context: %w", err)
	}

	if current == DefaultContextName {
		dockerHost := os.Getenv(EnvOverrideHost)
		if dockerHost != "" {
			return parseURL(dockerHost)
		}

		return parseURL(DefaultDockerHost)
	}

	ctx, err := Inspect(current)
	if err != nil {
		return "", fmt.Errorf("inspect context: %w", err)
	}

	// Inspect already validates that the docker endpoint is set
	return parseURL(ctx.Endpoints["docker"].Host)
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

func parseURL(s string) (string, error) {
	hostURL, err := url.Parse(s)
	if err != nil {
		return "", err
	}

	switch hostURL.Scheme + "://" {
	case DefaultSchema:
		// return the original URL, as it is a valid socket URL
		return s, nil
	case TCPSchema:
		// return the original URL, as it is a valid TCP URL
		return s, nil
	default:
		return "", ErrInvalidSchema
	}
}
