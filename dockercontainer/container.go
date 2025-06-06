package dockercontainer

import (
	"context"
	"log/slog"
	"time"

	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockercontainer/wait"
)

// Container represents a container
type Container struct {
	dockerClient *dockerclient.Client

	// ID the Container ID
	ID string

	// shortID the short Container ID, using the first 12 characters of the ID
	shortID string

	// WaitingFor the waiting strategy to use for the container.
	WaitingFor wait.Strategy

	// TODO: Remove locking and wait group once the deprecated StartLogProducer and
	// StopLogProducer have been removed and hence logging can only be started and
	// stopped once.

	// logProductionCancel is used to signal the log production to stop.
	logProductionCancel context.CancelCauseFunc
	logProductionCtx    context.Context

	logProductionTimeout *time.Duration

	// Image the image to use for the container.
	Image string

	// exposedPorts the ports exposed by the container.
	exposedPorts []string

	// logger the logger to use for the container.
	logger *slog.Logger

	// lifecycleHooks the lifecycle hooks to use for the container.
	lifecycleHooks []LifecycleHooks

	// consumers the log consumers to use for the container.
	consumers []LogConsumer

	// isRunning the flag to check if the container is running.
	isRunning bool
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
