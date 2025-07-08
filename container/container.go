package container

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/docker/api/types/container"
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

// DockerClient returns the docker client used by the container.
func (c *Container) DockerClient() *client.Client {
	return c.dockerClient
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

// WaitingFor returns the waiting strategy used by the container.
func (c *Container) WaitingFor() wait.Strategy {
	return c.waitingFor
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

// FromResponse builds a container struct from the response of the Docker API
func FromResponse(_ context.Context, response container.Summary) (*Container, error) {
	exposedPorts := make([]string, len(response.Ports))
	for i, port := range response.Ports {
		exposedPorts[i] = fmt.Sprintf("%d/%s", port.PublicPort, port.Type)
	}

	ctr := &Container{
		containerID:  response.ID,
		shortID:      response.ID[:12],
		image:        response.Image,
		isRunning:    response.State == "running",
		exposedPorts: exposedPorts,
		lifecycleHooks: []LifecycleHooks{
			DefaultLoggingHook,
		},
	}

	return ctr, nil
}
