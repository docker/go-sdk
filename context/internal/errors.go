package internal

import "errors"

var (
	// ErrDockerHostNotSet is returned when the Docker host is not set in the Docker context.
	ErrDockerHostNotSet = errors.New("docker host not set in Docker context")

	// ErrDockerContextNotFound is returned when the Docker context is not found.
	ErrDockerContextNotFound = errors.New("docker context not found")
)
