package container

import (
	"context"
	"log/slog"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/container/wait"
)

// Container represents a container
type Container struct {
	dockerClient *client.Client

	// containerID the Container ID
	containerID string

	// shortID the short Container ID, using the first 12 characters of the ID
	shortID string

	// WaitingFor the waiting strategy to use for the container.
	waitingFor wait.Strategy

	// image the image to use for the container.
	image string

	// exposedPorts the ports exposed by the container.
	exposedPorts []string

	// logger the logger to use for the container.
	logger *slog.Logger

	// lifecycleHooks the lifecycle hooks to use for the container.
	lifecycleHooks []LifecycleHooks

	// isRunning the flag to check if the container is running.
	isRunning bool
}

// ID returns the container ID
func (c *Container) ID() string {
	return c.containerID
}

// Image returns the image used by the container.
func (c *Container) Image() string {
	return c.image
}

// ShortID returns the short container ID, using the first 12 characters of the ID
func (c *Container) ShortID() string {
	return c.shortID
}

// Host gets host (ip or name) of the docker daemon where the container port is exposed
// Warning: this is based on your Docker host setting. Will fail if using an SSH tunnel
func (c *Container) Host(ctx context.Context) (string, error) {
	host, err := c.dockerClient.DaemonHost(ctx)
	if err != nil {
		return "", err
	}
	return host, nil
}
